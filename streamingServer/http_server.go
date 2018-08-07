package streamingServer

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"streamingServer/httpHandlers"
)

type HTMLPage struct {
	Title string
	Body  []byte
}

type HttpStreamServer struct {
	streamBroadcaster *Broadcaster
}

func NewHttpStreamServer(bc *Broadcaster) *HttpStreamServer {
	return &HttpStreamServer{bc}
}

func (hss *HttpStreamServer) handleRootRequest(writer http.ResponseWriter, req *http.Request) {
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

func (hss *HttpStreamServer) handleStreamRequest(writer http.ResponseWriter, req *http.Request) {
	streamID := req.URL.Path
	streamClient, err := hss.streamBroadcaster.AddClientStream(req.RemoteAddr, streamID)
	if err != nil {
		return
	}

	handleHttpStream, hErr := httpHandlers.GetHttpStreamHandler(streamClient.GetStreamType())
	if hErr != nil {
		streamClient.SetDone()
		return
	}

	httpHandlers.SendHttpHeaders(streamClient.GetStreamType(), writer)
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

func (hss *HttpStreamServer) loadPageFromFile(filename, title string) (*HTMLPage, error) {
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("Error reading from file:\n", err)
		return nil, err
	}
	return &HTMLPage{Title: title, Body: body}, nil
}

func (hss *HttpStreamServer) PrepareStreamHandlers(prepend string, nStreams int) {
	http.Handle("/", http.FileServer(http.Dir(".")))
	for i := 0; i < nStreams; i++ {
		streamID := fmt.Sprintf("/%s%d", prepend, i)
		http.HandleFunc(streamID, hss.handleStreamRequest)
	}
}

func (hss *HttpStreamServer) StartServer(ip string, port int) {
	http.ListenAndServe(fmt.Sprintf("%s:%d", ip, port), nil)
}
