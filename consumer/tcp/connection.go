package tcp

import (
	"consts"
	"fmt"
	"net"
	"streamingServer/consumer/tcp/handler"
	"sync"
)

type connectionStream struct {
	conn    *net.TCPConn
	outChan chan []byte
}

// StreamConnection represents an in use and read-only stream connection
type StreamConnection struct {
	streamID      string
	streamType    int32
	streamChanMap map[consts.Quality]connectionStream
	isOpen        bool
	sync.Mutex
}

func NewStreamConnection(streamID string, streamType int32, streamQuality consts.Quality, connection *net.TCPConn) *StreamConnection {
	tsc := &StreamConnection{
		streamID:      streamID,
		streamType:    streamType,
		streamChanMap: make(map[consts.Quality]connectionStream),
		isOpen:        false,
	}

	tsc.streamChanMap[streamQuality] = connectionStream{connection, make(chan []byte, 32)}
	return tsc
}

func (sc *StreamConnection) GetID() string {
	return sc.streamID
}

func (sc *StreamConnection) GetType() int32 {
	return sc.streamType
}

func (sc *StreamConnection) close(quality consts.Quality) error {
	sc.Lock()
	if sc.isOpen {
		streamConn, ok := sc.streamChanMap[quality]
		if !ok {
			return fmt.Errorf("Connection for stream %s with quality %d does not exist", sc.GetID, quality)
		}

		err := streamConn.conn.Close()
		if err != nil {
			return err
		}

		close(streamConn.outChan)
		sc.isOpen = false
	}
	sc.Unlock()

	return nil
}

func (sc *StreamConnection) Close() error {
	for k, _ := range sc.streamChanMap {
		err := sc.close(k)
		if err != nil {
			return err
		}
	}

	return nil
}

func (sc *StreamConnection) GetNextChunk(quality consts.Quality) ([]byte, error) {
	outputChan, err := sc.GetOutputChan(quality)
	if err != nil {
		return nil, err
	}

	return <-outputChan, nil
}

func (sc *StreamConnection) GetOutputChan(quality consts.Quality) (<-chan []byte, error) {
	streamChan, ok := sc.streamChanMap[quality]
	if !ok {
		return nil, fmt.Errorf("No stream for %s with quality %s", sc.GetID(), quality)
	}

	return streamChan.outChan, nil
}

func (sc *StreamConnection) HandleStream(quality consts.Quality) error {
	streamHandleFunc, err := handler.GetTCPStreamHandleFunc(sc.GetType())
	if err != nil {
		return err
	}

	streamConn, ok := sc.streamChanMap[quality]
	if !ok {
		return err
	}

	streamHandleFunc(streamConn.conn, streamConn.outChan)
	return nil
}
