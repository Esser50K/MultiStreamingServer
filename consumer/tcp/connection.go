package consumer

import (
	"StreamingServer/consts"
	"StreamingServer/consumer"
	"StreamingServer/consumer/tcp/handler"
	"fmt"
	"net"
	"sync"
)

type connectionStream struct {
	conn    *net.TCPConn
	outChan chan []byte
}

// TCPStreamConnection represents an in use and read-only stream connection
type TCPStreamConnection struct {
	consumer.BaseStreamConnection
	streamChanMap map[consts.Quality]connectionStream
	isOpen        bool
	sync.Mutex
}

func NewTCPStreamConnection(streamID string, streamType consts.StreamType, streamQuality consts.Quality, connection *net.TCPConn) *TCPStreamConnection {
	tsc := &TCPStreamConnection{
		BaseStreamConnection: consumer.NewBaseStreamConnection(streamID, streamType),
		streamChanMap:        make(map[consts.Quality]connectionStream),
		isOpen:               false,
	}

	tsc.streamChanMap[streamQuality] = connectionStream{connection, make(chan []byte, 32)}
	return tsc
}

func (sc *TCPStreamConnection) AddConnection(quality consts.Quality, conn interface{}) error {
	connection, ok := conn.(*net.TCPConn)
	if !ok {
		return fmt.Errorf("conn argument must be of type *net.TCPConn")
	}

	sc.streamChanMap[quality] = connectionStream{
		conn:    connection,
		outChan: make(chan []byte, 32),
	}
	return nil
}

func (sc *TCPStreamConnection) Close(quality consts.Quality) error {
	sc.Lock()
	defer sc.Unlock()

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
		delete(sc.streamChanMap, quality)
	}

	if len(sc.streamChanMap) == 0 {
		sc.isOpen = false
	}

	return nil
}

func (sc *TCPStreamConnection) GetNextChunk(quality consts.Quality) ([]byte, error) {
	outputChan, err := sc.GetOutputChan(quality)
	if err != nil {
		return nil, err
	}

	return <-outputChan, nil
}

func (sc *TCPStreamConnection) GetOutputChan(quality consts.Quality) (<-chan []byte, error) {
	sc.Lock()
	defer sc.Unlock()
	streamChan, ok := sc.streamChanMap[quality]
	if !ok {
		return nil, fmt.Errorf("No stream for %s with quality %s", sc.GetID(), quality)
	}

	return streamChan.outChan, nil
}

func (sc *TCPStreamConnection) HandleStream(quality consts.Quality) error {
	sc.isOpen = true
	streamHandleFunc, err := tcphandler.GetTCPStreamHandleFunc(sc.GetType())
	if err != nil {
		return err
	}

	streamConn, ok := sc.streamChanMap[quality]
	if !ok {
		return fmt.Errorf("no stream connection for quality %d", quality)
	}

	return streamHandleFunc(streamConn.conn, streamConn.outChan)
}

func (sc *TCPStreamConnection) IsOpen() bool {
	return sc.isOpen
}
