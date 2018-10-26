package tcphandler

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"time"
)

func HandleH264Stream(connection *net.TCPConn, outputChannel chan []byte) error {

	var frameBuffer []byte
	var nalSeparator = []byte{0, 0, 0, 1}
	streamChan := make(chan []byte, 32)
	go func() {
		for {
			select {
			case data, ok := <-streamChan:
				if !ok {
					return
				}

				accumulator := 0
				frameBuffer = append(frameBuffer, data...)
				nals := bytes.Split(frameBuffer, nalSeparator)
				for len(nals) > 1 {
					nalIndex := 0
					if len(nals[0]) == 0 {
						nalIndex = 1
					}
					outputChannel <- append(nalSeparator, nals[nalIndex]...)

					accumulator += 4 + len(nals[nalIndex])
					nals = nals[nalIndex+1:]
				}
				frameBuffer = frameBuffer[accumulator:]
			}
		}
	}()

	count := 0
	start := time.Now().Unix()
	finish := time.Now().Unix()
	var imgSize int32
	for {
		err := binary.Read(connection, binary.LittleEndian, &imgSize)
		if err != nil {
			fmt.Printf("Error while reading Image Size from socket: %s\n", err)
			return err
		}

		if imgSize == 0 {
			fmt.Println("Streaming Client closed the connection.")
			return nil
		}

		var nBytesRead int32
		imgBuffer := make([]byte, imgSize, imgSize)
		for {
			readBuffer := make([]byte, int64(math.Min(float64(4096), float64(imgSize-nBytesRead))))
			readCount, err := connection.Read(readBuffer)
			if err != nil {
				return nil
			}
			copy(imgBuffer[nBytesRead:], readBuffer[:readCount])
			nBytesRead += int32(readCount)
			if nBytesRead == imgSize {
				break
			}
		}

		count += len(imgBuffer)
		finish = time.Now().Unix()
		if finish-start > 10 {
			bandwidth_mbit := ((float64(count) / float64(finish-start)) / 1000000.0) * 8.0
			fmt.Println("Reading", bandwidth_mbit, "Mbits/s")
			start = time.Now().Unix()
			count = 0
		}

		// if output channel is full start discarding frames.
		select {
		//case streamChan <- imgBuffer: // Send image bytes to channel
		case outputChannel <- imgBuffer: // Send image bytes to channel
		default:
			<-outputChannel
			outputChannel <- imgBuffer
		}
	}
}
