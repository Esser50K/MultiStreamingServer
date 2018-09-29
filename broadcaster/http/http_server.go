package http

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"streamingServer/broadcaster"
	"streamingServer/broadcaster/http/handler"
)

type HTMLPage struct {
	Title string
	Body  []byte
}

type HttpBroadcaster struct {
	*broadcaster.Broadcaster
}

func NewBroadcaster(bc *broadcaster.Broadcaster) *HttpBroadcaster {
	return &HttpBroadcaster{
		Broadcaster: bc,
	}
}

func (hss *HttpBroadcaster) handleRootRequest(writer http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/index.html":
		indexHTML, err := hss.loadPageFromFile("index.html", "PiSurveillance")
		if err != nil {
			fmt.Fprintf(writer, "<body></body>")
			return
		}

		fmt.Fprintf(writer, "%s", indexHTML.Body)
	default:
		http.Redirect(writer, req, "/index.html", http.StatusFound)
	}
}

func (hss *HttpBroadcaster) handleStreamRequest(writer http.ResponseWriter, req *http.Request) {
	streamID := req.URL.Path
	streamClient, err := hss.AddClientStream(req.RemoteAddr, streamID)
	if err != nil {
		return
	}

	handleHttpStream, hErr := handler.GetHTTPStreamHandler(streamClient.GetStreamType())
	if hErr != nil {
		streamClient.SetDone()
		return
	}

	handler.SendHTTPHeaders(streamClient.GetStreamType(), writer)
	for !streamClient.IsDone() {
		changeQuality, cqErr := handleHttpStream(streamClient.GetOutputChannel(), writer, req)
		if cqErr != nil {
			streamClient.SetDone()
			return
		}

		indicator, qErr := streamClient.ChangeWantedQuality(changeQuality)
		fmt.Println(changeQuality, indicator, qErr)
		if qErr != nil && indicator == !changeQuality {
			streamClient.SetDone()
			return
		}
	}
}

func (hss *HttpBroadcaster) loadPageFromFile(filename, title string) (*HTMLPage, error) {
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("Error reading from file:\n", err)
		return nil, err
	}
	return &HTMLPage{Title: title, Body: body}, nil
}

func (hss *HttpBroadcaster) PrepareStreamHandlers(prepend string, nStreams int) {
	http.Handle("/", http.FileServer(http.Dir(".")))
	for i := 0; i < nStreams; i++ {
		streamID := fmt.Sprintf("/%s%d", prepend, i)
		http.HandleFunc(streamID, hss.handleStreamRequest)
	}
}

func (hss *HttpBroadcaster) StartServer(ip string, port int) {
	http.ListenAndServe(fmt.Sprintf("%s:%d", ip, port), nil)
}
