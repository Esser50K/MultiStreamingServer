package handler

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"time"
)

func HandleJpegStream(connection *net.TCPConn, outputChannel chan []byte) error {
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

		var nBytesRead int32 = 0
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

		count++
		finish = time.Now().Unix()
		if finish-start > 10 {
			readFPS := float64(count) / float64(finish-start)
			fmt.Println("Reading", readFPS, "fps")
			start = time.Now().Unix()
			count = 0
		}

		// if output channel is full start discarding frames.
		select {
		case outputChannel <- imgBuffer: // Send image bytes to channel
		default:
			<-outputChannel
			outputChannel <- imgBuffer
		}
	}
}
