package httpHandlers

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"
)

var (
	BOUNDARY         = "--BOUNDARY"
	minFPS   float64 = 8
	maxFPS   float64 = 29
)

func SendJpegHeaders(writer http.ResponseWriter) {
	writer.Header().Add("Content-Type", "multipart/x-mixed-replace;boundary="+BOUNDARY)
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
}

func HandleJpegStreamRequest(streamChan chan []byte, writer http.ResponseWriter, req *http.Request) (bool, error) {
	closeChannel := writer.(http.CloseNotifier).CloseNotify()
	startTime := time.Now().Unix()
	nPushedFrames := 0

	//Frame cleaner routine to avoid long delays
	auxCloseChan := make(chan struct{})
	auxBuffer := make(chan []byte, 16)
	go func() {
		failCount := 0
		for {
			select {
			case auxBuffer <- <-streamChan:
				if failCount > 0 {
					failCount--
				}
			case <-auxCloseChan:
				close(auxBuffer)
				return
			case <-closeChannel:
				close(auxBuffer)
				return
			default:
				failCount = int(math.Min(float64(len(auxBuffer)/2), float64(failCount+1)))
				for i := 0; i < failCount; i++ {
					<-auxBuffer
					auxBuffer <- <-streamChan
				}
			}
		}
	}()

	writeBuffer := new(bytes.Buffer)
	first := true
	qualityCounter := 0
	for {
		select {
		case <-time.After(5 * time.Second):
			fmt.Println(req.RemoteAddr, " has too poor connectivity to the server, removing from stream.", req.URL.Path)
			close(auxCloseChan)
			return false, nil

		case img, ok := <-auxBuffer:

			if !ok {
				fmt.Println(req.RemoteAddr, " has left the stream", req.URL.Path)
				return false, errors.New(req.RemoteAddr + " has left the stream" + req.URL.Path)
			}

			if !first {
				writeBuffer.Write([]byte("\r\n"))
			}

			fmt.Fprintf(writeBuffer, "%s\r\n", BOUNDARY)
			writeBuffer.Write([]byte("Content-Type: image/jpeg\r\n"))
			fmt.Fprintf(writeBuffer, "Content-Length: %d\r\n", len(img))
			writeBuffer.Write([]byte("\r\n"))
			writeBuffer.Write(img)
			nWrittenBytes, err := writer.Write(writeBuffer.Bytes())
			if err != nil || nWrittenBytes != writeBuffer.Len() {
				close(auxCloseChan)
				return false, err
			}

			writeBuffer.Reset()

			first = false

			// Client FPS Calculation
			nPushedFrames += 1
			timePassed := time.Now().Unix() - startTime
			frameRate := float64(nPushedFrames) / float64(timePassed)
			if timePassed > 30 {
				fmt.Printf("Pushed %d frames in %d seconds at %.2f fps on stream %s for client %s\n",
					nPushedFrames,
					timePassed,
					frameRate,
					req.URL.Path,
					req.RemoteAddr,
				)

				if frameRate < minFPS {
					fmt.Println(req.RemoteAddr, " has too poor connectivity to the server,  trying to reduce quality of stream.", req.URL.Path)
					close(auxCloseChan)
					return false, nil
				}

				if frameRate > maxFPS {
					qualityCounter++
					if qualityCounter > 5 {
						fmt.Println(req.RemoteAddr, " has good connectivity to the server, trying to improve quality of stream.", req.URL.Path)
						close(auxCloseChan)
						return true, nil
					}
				} else {
					qualityCounter = 0
				}

				nPushedFrames = 0
				startTime = time.Now().Unix()
			}
		}
	}
}
