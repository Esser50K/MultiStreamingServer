package broadcaster

import (
	"StreamingServer/consts"
	"StreamingServer/consumer"
	"fmt"
	"sync"
	"sync/atomic"
)

type streamClient struct {
	clientID      string
	streamType    consts.StreamType
	wantedQuality consts.Quality
	inputChan     chan []byte
	done          uint32
	sync.Mutex
}

func (c *streamClient) GetStreamType() consts.StreamType {
	return c.streamType
}

func (c *streamClient) GetOutputChannel() chan []byte {
	return c.inputChan
}

func (c *streamClient) SetDone() {
	atomic.StoreUint32(&c.done, 1)
}

func (c *streamClient) IsDone() bool {
	return atomic.LoadUint32(&c.done) == 1
}

func (c *streamClient) ChangeWantedQuality(higher bool) error {
	c.Lock()
	defer c.Unlock()
	if higher {
		c.wantedQuality++
	} else {
		c.wantedQuality--
	}

	// Maintain at highest quality
	if c.wantedQuality > consts.HighQuality {
		c.wantedQuality = consts.HighQuality
		return nil
	}

	fmt.Println("Wanted Quality:", c.wantedQuality)
	if c.wantedQuality < consts.LowQuality {
		return fmt.Errorf("requesting to low quality: %d", c.wantedQuality)
	}

	return nil
}

type streamBroadcaster struct {
	streamID       string
	inputStream    consumer.StreamConnection
	clientStreams  []*streamClient
	isBroadcasting bool
	sync.Mutex
}

func (sb *streamBroadcaster) addClient(c *streamClient) {
	sb.Lock()
	sb.clientStreams = append(sb.clientStreams, c)
	sb.Unlock()
}

func (sb *streamBroadcaster) setClientsDone() {
	sb.Lock()
	defer sb.Unlock()

	for _, s := range sb.clientStreams {
		s.SetDone()
	}
}

func (sb *streamBroadcaster) Broadcast() {
	sb.isBroadcasting = true
	defer func() {
		sb.isBroadcasting = false
		for _, client := range sb.clientStreams {
			client.SetDone()
		}
	}()

	for {
		fmt.Println("InputStream", sb.streamID, "is open:", sb.inputStream.IsOpen())
		if !sb.inputStream.IsOpen() {
			fmt.Println("Input Stream is closed. Stopping Broadcast.")
			return
		}

		// Get images of all qualities
		dataQualityMap := make(map[consts.Quality][]byte)
		for quality, _ := range consts.Qualities {
			qualityChan, err := sb.inputStream.GetOutputChan(quality)
			if err != nil {
				continue
			}

			image, ok := <-qualityChan
			if ok {
				dataQualityMap[quality] = image
			}
		}

		// If no image was retrieved then stop broadcasting
		if len(dataQualityMap) == 0 {
			return
		}

		sb.Lock()
		for index := len(sb.clientStreams) - 1; index >= 0; index-- {
			streamClient := sb.clientStreams[index]
			if streamClient.IsDone() {
				fmt.Println("Removing streamClient", streamClient.clientID, "from", sb.streamID, "broadcast")
				sb.clientStreams[index] = nil
				sb.clientStreams = append(sb.clientStreams[:index], sb.clientStreams[index+1:]...)
				continue
			}

			image, ok := dataQualityMap[streamClient.wantedQuality]
			if !ok {
				// If image for wanted quality does not exist send LowQuality
				image = dataQualityMap[consts.LowQuality]
				streamClient.wantedQuality = consts.LowQuality
			}

			select {
			case streamClient.inputChan <- image:
			default:
				<-streamClient.inputChan
				streamClient.inputChan <- image
			}
		}
		sb.Unlock()
	}

	fmt.Println("Finished Broadcasting.")
}

type Broadcaster struct {
	consumer.StreamConsumer
	streamBroadcasters map[string]*streamBroadcaster
	sync.Mutex
}

func NewBroadcaster(streamConsumer consumer.StreamConsumer) *Broadcaster {
	return &Broadcaster{
		streamBroadcasters: make(map[string]*streamBroadcaster),
		StreamConsumer:     streamConsumer,
	}
}

func (bc *Broadcaster) cleanBroadcaster(streamID string) {
	bc.Lock()
	defer bc.Unlock()

	_, ok := bc.streamBroadcasters[streamID]
	if !ok {
		return
	}

	delete(bc.streamBroadcasters, streamID)
}

func (bc *Broadcaster) countClients() int {
	count := 0
	for _, sb := range bc.streamBroadcasters {
		count += len(sb.clientStreams)
	}
	return count
}

func (bc *Broadcaster) AddClientStream(clientID, streamID string) (*streamClient, error) {
	bc.Lock()
	defer bc.Unlock()

	stream, err := bc.GetStream(streamID)
	if err != nil {
		return nil, err
	}

	newClient := &streamClient{
		clientID:      clientID,
		wantedQuality: consts.HighQuality,
		streamType:    stream.GetType(),
		done:          0,
		inputChan:     make(chan []byte, 4),
	}

	// Check if broadcaster for that specific stream exists
	sBroadcaster, broadcasterOK := bc.streamBroadcasters[streamID]
	if broadcasterOK {
		fmt.Println("Adding streamClient", clientID, "to existing broadcast on stream", streamID)

		sBroadcaster.addClient(newClient)
		fmt.Printf("Broadcaster currently has %d streamers and %d clients\n", len(bc.streamBroadcasters), bc.countClients())
		return newClient, nil
	}

	// In case the broadcaster does not exist then create it and add the client to it.
	var cStreams []*streamClient
	cStreams = append(cStreams, newClient)
	sBroadcaster = &streamBroadcaster{
		streamID:      streamID,
		inputStream:   stream,
		clientStreams: cStreams,
	}

	bc.streamBroadcasters[streamID] = sBroadcaster

	// Start broadcasting routine
	go func(sBroadcaster *streamBroadcaster) {
		sBroadcaster.Broadcast()
		fmt.Println("Removing Broadcaster of stream", streamID)
		bc.cleanBroadcaster(streamID)
	}(sBroadcaster)

	fmt.Printf("Broadcaster currently has %d streamers and %d clients\n", len(bc.streamBroadcasters), bc.countClients())
	return newClient, nil
}
