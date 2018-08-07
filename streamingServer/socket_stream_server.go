package streamingServer

import (
	"encoding/binary"
	"fmt"
	"net"
	"streamingServer/consts"
	"streamingServer/tcpHandlers"
	"strings"
	"sync"
)

type streamConnection struct {
	streamID            string
	streamType          int32
	streamQuality       consts.Quality
	handleStream        tcpHandlers.TCPStreamHandler
	streamConnection    *net.TCPConn
	streamOutputChannel chan []byte
	isOpen              bool
	sync.Mutex
}

func (sc *streamConnection) GetStreamType() int32 {
	return sc.streamType
}

func (sc *streamConnection) GetStreamQuality() consts.Quality {
	return sc.streamQuality
}

func (sc *streamConnection) Close() error {
	sc.Lock()
	if sc.isOpen {
		err := sc.streamConnection.Close()
		if err != nil {
			return err
		}

		close(sc.streamOutputChannel)
		sc.isOpen = false
	}
	sc.Unlock()

	return nil
}

func (sc *streamConnection) CheckGetNextImage() ([]byte, bool) {
	img, ok := <-sc.streamOutputChannel
	return img, ok
}

func (sc *streamConnection) GetNextImage() []byte {
	return <-sc.streamOutputChannel
}

func (sc *streamConnection) GetOutputChannel() chan []byte {
	return sc.streamOutputChannel
}

type StreamServer struct {
	maxStreamers    int
	readersReady    int32
	activeStreamers map[string]*streamConnection
}

func NewStreamingServer(maxStreamers int) (ss *StreamServer) {
	return &StreamServer{
		activeStreamers: make(map[string]*streamConnection),
		maxStreamers:    maxStreamers,
		readersReady:    0,
	}
}

func (ss *StreamServer) GetStreamingConnections() map[string]*streamConnection {
	return ss.activeStreamers
}

func (ss *StreamServer) GetStream(streamID string) (sc *streamConnection) {
	return ss.activeStreamers[streamID]
}

func (ss *StreamServer) ListenForStreamers(ip string, port int) error {
	address := fmt.Sprintf("%s:%d", ip, port)
	tcpAddr, _ := net.ResolveTCPAddr("tcp4", address)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		fmt.Println("Unable to start TCP Server on port 1235. Aborting due to error: %s", err)
		return err
	}

	mutex := &sync.Mutex{}
	cond := sync.NewCond(mutex)
	for {
		fmt.Println("Listening for connection...")
		conn, err := listener.AcceptTCP()
		if err != nil {
			fmt.Println("Error occurred when accepting connection, not handling this client: %s", err)
			continue
		}

		var streamIDTypeQuality [3]int32
		for i := 0; i < 3; i++ {
			streamIDTypeQuality[i], err = ss.readInt32(conn)
			if err != nil {
				fmt.Println("Error occurred when reading stream type, not handling this client: %s", err)
				conn.Close()
				continue
			}
		}

		streamHandleFunc, sErr := tcpHandlers.GetTCPStreamHandleFunc(streamIDTypeQuality[1])
		if sErr != nil {
			return sErr
		}

		streamQuality, qErr := tcpHandlers.GetStreamQuality(streamIDTypeQuality[2])
		if qErr != nil {
			return qErr
		}

		fmt.Println("Received connection successfully, passing to handler.")
		streamID := fmt.Sprintf("/stream%d_%s", streamIDTypeQuality[0], streamQuality)
		outputChannel := make(chan []byte, 16)
		connection := &streamConnection{
			streamID:            streamID,
			streamType:          streamIDTypeQuality[1],
			handleStream:        streamHandleFunc,
			streamQuality:       streamQuality,
			streamConnection:    conn,
			streamOutputChannel: outputChannel,
			isOpen:              true,
		}

		fmt.Println("Registering stream with id:", streamID)
		ss.activeStreamers[streamID] = connection

		go func(sConnection *streamConnection) {
			defer func() {
				sConnection.Close()
				delete(ss.activeStreamers, streamID)
				mutex.Lock()
				cond.Broadcast()
				mutex.Unlock()
			}()

			sConnection.handleStream(sConnection.streamConnection, sConnection.streamOutputChannel)
		}(connection)

		for len(ss.activeStreamers) == ss.maxStreamers {
			mutex.Lock()
			cond.Wait()
			mutex.Unlock()
		}
	}
}

func (ss *StreamServer) readInt32(connection *net.TCPConn) (int32, error) {
	var val int32
	err := binary.Read(connection, binary.LittleEndian, &val)
	if err != nil {
		return -1, err
	}

	return val, nil
}

func (ss *StreamServer) getStreamersWithPrefix(prefix string) map[consts.Quality]*streamConnection {
	streamers := make(map[consts.Quality]*streamConnection)
	for k, v := range ss.activeStreamers {
		if strings.HasPrefix(k, prefix) {
			streamers[v.streamQuality] = v
		}
	}

	return streamers
}
