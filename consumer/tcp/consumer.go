package tcp

import (
	"encoding/binary"
	"fmt"
	"net"
	"streamingServer/consumer"
	"streamingServer/consumer/tcp/handler"
	"sync"
)

type Consumer struct {
	maxStreamers    int
	readersReady    int32
	activeStreamers map[string]*StreamConnection
	listenIP        string
	listenPort      int
	streamPrefix    string
	running         bool
}

func NewConsumer(ip string, port, maxStreamers int, streamPrefix string) (ss *Consumer) {
	return &Consumer{
		activeStreamers: make(map[string]*StreamConnection),
		maxStreamers:    maxStreamers,
		listenIP:        ip,
		listenPort:      port,
		streamPrefix:    streamPrefix,
		readersReady:    0,
	}
}

func (ss *Consumer) GetStream(streamID string) (consumer.StreamConnection, error) {
	stream, ok := ss.activeStreamers[streamID]
	if !ok {
		return nil, fmt.Errorf("No stream registered with id '%s\n", streamID)
	}

	return stream, nil
}

func (ss *Consumer) Stop() error {
	ss.running = false
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ss.listenIP, ss.listenPort))
	if err != nil {
		ss.running = true
		return err
	}
	conn.Close()
	return nil
}

func (ss *Consumer) Start() error {
	address := fmt.Sprintf("%s:%d", ss.listenIP, ss.listenPort)
	tcpAddr, _ := net.ResolveTCPAddr("tcp4", address)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		fmt.Printf("Unable to start TCP Server on port 1235. Aborting due to error: %s\n", err)
		return err
	}

	mutex := &sync.Mutex{}
	cond := sync.NewCond(mutex)
	ss.running = true
	for ss.running {
		fmt.Println("Listening for connection...")
		conn, err := listener.AcceptTCP()
		if err != nil {
			fmt.Printf("Error occurred when accepting connection, not handling this client: %s\n", err)
			continue
		}

		var streamIDTypeQuality [3]int32
		for i := 0; i < 3; i++ {
			streamIDTypeQuality[i], err = ss.readInt32(conn)
			if err != nil {
				fmt.Printf("Error occurred when reading stream type, not handling this client: %s\n", err)
				conn.Close()
				continue
			}
		}

		streamQuality, qErr := handler.GetStreamQuality(streamIDTypeQuality[2])
		if qErr != nil {
			return qErr
		}

		fmt.Println("Received connection successfully, passing to handler.")
		streamID := fmt.Sprintf("/%s%d", ss.streamPrefix, streamIDTypeQuality[0])
		connection := NewStreamConnection(streamID, streamIDTypeQuality[1], streamQuality, conn)

		fmt.Println("Registering stream with id:", streamID)
		ss.activeStreamers[streamID] = connection

		go func(sConnection *StreamConnection) {
			defer func() {
				sConnection.Close()
				delete(ss.activeStreamers, streamID)
				mutex.Lock()
				cond.Broadcast()
				mutex.Unlock()
			}()

			sConnection.HandleStream(streamQuality)
		}(connection)

		for len(ss.activeStreamers) == ss.maxStreamers {
			mutex.Lock()
			cond.Wait()
			mutex.Unlock()
		}
	}
	listener.Close()
	return nil
}

func (ss *Consumer) readInt32(connection *net.TCPConn) (int32, error) {
	var val int32
	err := binary.Read(connection, binary.LittleEndian, &val)
	if err != nil {
		return -1, err
	}

	return val, nil
}

func (ss *Consumer) getStreamConnection(streamID string) (*StreamConnection, error) {
	for k, v := range ss.activeStreamers {
		if k == streamID {
			return v, nil
		}
	}

	return nil, fmt.Errorf("No stream with ID %s", streamID)
}
