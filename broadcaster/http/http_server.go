package broadcaster

import (
	"StreamingServer/broadcaster"
	"StreamingServer/broadcaster/http/httphandler"
	"StreamingServer/consumer"
	"fmt"
	"io/ioutil"
	"net/http"
)

type HTMLPage struct {
	Title string
	Body  []byte
}

type HttpBroadcaster struct {
	*broadcaster.Broadcaster
}

func NewHTTPBroadcaster(streamConsumer consumer.StreamConsumer) *HttpBroadcaster {
	return &HttpBroadcaster{
		Broadcaster: broadcaster.NewBroadcaster(streamConsumer),
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
	streamID := req.URL.Path[1:] // cut off the / at the beginning
	streamClient, err := hss.AddClientStream(req.RemoteAddr, streamID)
	if err != nil {
		return
	}

	handleHttpStream, hErr := httphandler.GetHTTPStreamHandler(streamClient.GetStreamType())
	if hErr != nil {
		streamClient.SetDone()
		return
	}

	httphandler.SendHTTPHeaders(streamClient.GetStreamType(), writer)
	for !streamClient.IsDone() {
		// Need to be wary of raising h264 quality since it will throw an error about using a hjacked connection
		changeQuality, cqErr := handleHttpStream(streamClient.GetOutputChannel(), writer, req)
		if cqErr != nil {
			streamClient.SetDone()
			return
		}

		qErr := streamClient.ChangeWantedQuality(changeQuality)
		fmt.Println(changeQuality, qErr)
		if qErr != nil {
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
		fmt.Println(streamID)
		http.HandleFunc(streamID, hss.handleStreamRequest)
	}
}

func (hss *HttpBroadcaster) StartServer(ip string, port int) {
	http.ListenAndServe(fmt.Sprintf("%s:%d", ip, port), nil)
}
