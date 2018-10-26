package consumer

import (
	"StreamingServer/consts"
	"StreamingServer/consumer"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
)

type TCPConsumer struct {
	maxStreamers    int
	readersReady    int32
	activeStreamers map[string]*TCPStreamConnection
	listenIP        string
	listenPort      int
	streamPrefix    string
	running         bool
}

func NewTCPConsumer(ip string, port, maxStreamers int, streamPrefix string) *TCPConsumer {
	return &TCPConsumer{
		activeStreamers: make(map[string]*TCPStreamConnection),
		maxStreamers:    maxStreamers,
		listenIP:        ip,
		listenPort:      port,
		streamPrefix:    streamPrefix,
		readersReady:    0,
	}
}

func (sc *TCPConsumer) GetStream(streamID string) (consumer.StreamConnection, error) {
	fmt.Println("StreamRequest", streamID)
	fmt.Println("Active_streamers", sc.activeStreamers)
	stream, ok := sc.activeStreamers[streamID]
	if !ok {
		return nil, fmt.Errorf("No stream registered with id '%s\n", streamID)
	}

	return stream, nil
}

func (sc *TCPConsumer) Stop() error {
	sc.running = false
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", sc.listenIP, sc.listenPort))
	if err != nil {
		sc.running = true
		return err
	}
	conn.Close()
	return nil
}

func (sc *TCPConsumer) Start() error {
	address := fmt.Sprintf("%s:%d", sc.listenIP, sc.listenPort)
	tcpAddr, _ := net.ResolveTCPAddr("tcp4", address)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		fmt.Printf("Unable to start TCP Server on port 1235. Aborting due to error: %s\n", err)
		return err
	}

	mutex := &sync.Mutex{}
	cond := sync.NewCond(mutex)
	sc.running = true
	for sc.running {
		fmt.Println("Listening for connection...")
		conn, err := listener.AcceptTCP()
		if err != nil {
			fmt.Printf("Error occurred when accepting connection, not handling this client: %s\n", err)
			continue
		}

		var streamIDTypeQuality [3]int32
		for i := 0; i < 3; i++ {
			streamIDTypeQuality[i], err = sc.readInt32(conn)
			if err != nil {
				fmt.Printf("Error occurred when reading stream type, not handling this client: %s\n", err)
				conn.Close()
				continue
			}
		}

		fmt.Println("Received connection successfully, passing to handler.")
		streamID := fmt.Sprintf("%s%d", sc.streamPrefix, streamIDTypeQuality[0])
		connection := NewTCPStreamConnection(streamID, consts.StreamTypeIDs[int(streamIDTypeQuality[1])], consts.Quality(streamIDTypeQuality[2]), conn)

		fmt.Println("Registering stream with id:", streamID)
		sc.activeStreamers[streamID] = connection

		go func(sConnection *TCPStreamConnection) {
			defer func() {
				// No error means it is not handling anymore streams
				err = sConnection.Close(consts.Quality(streamIDTypeQuality[2]))
				if err != nil {
					fmt.Println("Error closing stream channel")
				}

				if !sConnection.IsOpen() {
					fmt.Printf("Removing handler for %s\n", streamID)
					delete(sc.activeStreamers, streamID)
				}

				mutex.Lock()
				cond.Broadcast()
				mutex.Unlock()
			}()

			sConnection.HandleStream(consts.Quality(streamIDTypeQuality[2]))
		}(connection)

		for len(sc.activeStreamers) == sc.maxStreamers {
			mutex.Lock()
			cond.Wait()
			mutex.Unlock()
		}
	}
	listener.Close()
	return nil
}

func (sc *TCPConsumer) readInt32(connection *net.TCPConn) (int32, error) {
	var val int32
	err := binary.Read(connection, binary.LittleEndian, &val)
	if err != nil {
		return -1, err
	}

	return val, nil
}

func (sc *TCPConsumer) getStreamConnection(streamID string) (*TCPStreamConnection, error) {
	for k, v := range sc.activeStreamers {
		if k == streamID {
			return v, nil
		}
	}

	return nil, fmt.Errorf("No stream with ID %s", streamID)
}
