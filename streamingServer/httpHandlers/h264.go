package httpHandlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

func SendH264Headers(writer http.ResponseWriter) {
	return
}

func HandleH264StreamRequest(streamChan chan []byte, writer http.ResponseWriter, request *http.Request) (bool, error) {
	closeChannel := writer.(http.CloseNotifier).CloseNotify()
	webConn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		return false, err
	}
	defer webConn.Close()

	/*
		initJson, _ := json.NewJson([]byte(`{}`))
		initJson.Set("action", "init")
		initJson.Set("width", 1280)
		initJson.Set("height", 720)
		encodedJson, _ := initJson.Encode()

		webConn.WriteMessage(websocket.TextMessage, encodedJson)

		// NAL Parser Routine
		auxCloseChan := make(chan struct{})
		outChan := make(chan []byte, 32)
		var frameBuffer []byte
		var nalSeparator = []byte{0, 0, 0, 1}
		go func() {
			for {
				select {
				case data := <-streamChan:
					accumulator := 0
					frameBuffer = append(frameBuffer, data...)
					nals := bytes.Split(frameBuffer, nalSeparator)
					for len(nals) > 1 {
						nalIndex := 0
						if len(nals[0]) == 0 {
							nalIndex = 1
						}
						outChan <- append(nalSeparator, nals[nalIndex]...)

						accumulator += 4 + len(nals[nalIndex])
						nals = nals[nalIndex+1:]
					}
					frameBuffer = frameBuffer[accumulator:]
				case <-auxCloseChan:
					close(outChan)
					return
				case <-closeChannel:
					close(outChan)
					return
				default:
				}
			}
		}()
	*/

	for {
		select {
		case <-time.After(5 * time.Second):
			fmt.Println(request.RemoteAddr, " has too poor connectivity to the server, removing from stream.", request.URL.Path)
			return false, nil

		case data, ok := <-streamChan:
			//case data, ok := <-streamChan:
			if !ok {
				return false, nil
			}

			err = webConn.WriteMessage(websocket.BinaryMessage, data)
			if err != nil {
				return false, err
			}

		case <-closeChannel:
			return false, nil
		}
	}
}
