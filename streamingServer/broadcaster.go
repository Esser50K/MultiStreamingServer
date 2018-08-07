package streamingServer

import (
	"errors"
	"fmt"
	"streamingServer/consts"
	"streamingServer/tcpHandlers"
	"sync"
	"sync/atomic"
)

type streamClient struct {
	clientID       string
	streamType     int32
	currentQuality consts.Quality
	wantedQuality  consts.Quality
	inputChan      chan []byte
	done           uint32
	sync.Mutex
}

func (c *streamClient) GetStreamType() int32 {
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

func (c *streamClient) ChangeWantedQuality(higher bool) (bool, error) {
	c.Lock()
	defer c.Unlock()
	qualityIndex, err := tcpHandlers.GetQualityIndex(c.currentQuality)
	if err != nil {
		return false, err
	}

	var errorIndicator bool
	fmt.Printf("qualityIndex before %d\n", qualityIndex)
	if higher {
		qualityIndex++
		errorIndicator = higher
	} else {
		qualityIndex--
		errorIndicator = !higher
	}
	fmt.Printf("qualityIndex after %d\n", qualityIndex)

	wQuality, qErr := tcpHandlers.GetStreamQuality(qualityIndex)
	if qErr != nil {
		c.wantedQuality = c.currentQuality
		return errorIndicator, qErr
	}

	c.wantedQuality = wQuality
	return true, nil
}

func (c *streamClient) wantSwitchQuality() bool {
	c.Lock()
	defer c.Unlock()

	if c.wantedQuality != c.currentQuality {
		return true
	}

	return false
}

type streamBroadcaster struct {
	sBcastHandler *streamBroadcastHandler
	quality       consts.Quality
	inputStream   chan []byte
	clientStreams []*streamClient

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
	for {
		image, ok := <-sb.inputStream
		if !ok {
			fmt.Println("The stream", sb.sBcastHandler.streamID+"_"+string(sb.quality), "stopped.")
			sb.setClientsDone()
			return
		}

		sb.Lock()
		for index := len(sb.clientStreams) - 1; index >= 0; index-- {
			streamClient := sb.clientStreams[index]
			if streamClient.IsDone() {
				fmt.Println("Removing streamClient", streamClient.clientID, "from", sb.sBcastHandler.streamID, "broadcast")
				sb.clientStreams[index] = nil
				sb.clientStreams = append(sb.clientStreams[:index], sb.clientStreams[index+1:]...)
				continue
			}

			if streamClient.wantSwitchQuality() {
				streamBroadcaster, exists := sb.sBcastHandler.streamBroadcasters[streamClient.wantedQuality]
				if !exists {
					fmt.Printf("No %s quality stream for client %s\n", streamClient.wantedQuality, streamClient.clientID)
					streamClient.wantedQuality = streamClient.currentQuality
					continue
				}

				fmt.Printf("Swapping client %s from %s to %s quality\n", streamClient.clientID,
					streamClient.currentQuality,
					streamClient.wantedQuality)

				streamClient.currentQuality = streamClient.wantedQuality
				streamBroadcaster.addClient(streamClient)
				sb.clientStreams = append(sb.clientStreams[:index], sb.clientStreams[index+1:]...)
				continue
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
}

type streamBroadcastHandler struct {
	streamID           string
	streamBroadcasters map[consts.Quality]*streamBroadcaster
}

func (sbh *streamBroadcastHandler) addClient(c *streamClient) {
	for _, quality := range consts.Qualities {
		streamBroadcaster, ok := sbh.streamBroadcasters[quality]
		if ok {
			c.currentQuality = quality
			c.wantedQuality = quality
			streamBroadcaster.addClient(c)
			return
		}
	}
}

type Broadcaster struct {
	StreamServer
	streamBroadcastHandlers map[string]*streamBroadcastHandler
	sync.Mutex
}

func NewBroadcaster(streamServer StreamServer) *Broadcaster {
	return &Broadcaster{
		streamBroadcastHandlers: make(map[string]*streamBroadcastHandler),
		StreamServer:            streamServer,
	}
}

func (bc *Broadcaster) cleanBroadcaster(streamID string, quality consts.Quality) {
	bc.Lock()
	defer bc.Unlock()

	sbh, ok := bc.streamBroadcastHandlers[streamID]
	if !ok {
		return
	}

	_, sOk := sbh.streamBroadcasters[quality]
	if !sOk {
		return
	}

	delete(sbh.streamBroadcasters, quality)
	if len(sbh.streamBroadcasters) == 0 {
		delete(bc.streamBroadcastHandlers, streamID)
	}
}

func (bc *Broadcaster) countClients() int {
	count := 0
	for _, sbh := range bc.streamBroadcastHandlers {
		for _, sb := range sbh.streamBroadcasters {
			count += len(sb.clientStreams)
		}
	}
	return count
}

func (bc *Broadcaster) AddClientStream(clientID, streamID string) (*streamClient, error) {
	bc.Lock()
	defer bc.Unlock()

	streamers := bc.getStreamersWithPrefix(streamID)
	if len(streamers) == 0 {
		fmt.Println("No broadcasts for stream:", streamID)
		return nil, errors.New(fmt.Sprintf("No broadcasts for stream: %s", streamID))
	}

	sBroadcastHandler, broadcasterOK := bc.streamBroadcastHandlers[streamID]
	if broadcasterOK {
		fmt.Println("Adding streamClient", clientID, "to existing broadcast on stream", streamID)
		for _, quality := range consts.Qualities {
			streamBroadcaster, ok := sBroadcastHandler.streamBroadcasters[quality]
			if !ok {
				continue
			}

			newClient := streamClient{
				clientID:       clientID,
				currentQuality: quality,
				wantedQuality:  quality,
				streamType:     streamers[quality].GetStreamType(),
				done:           0,
				inputChan:      make(chan []byte, 16),
			}
			streamBroadcaster.addClient(&newClient)
			fmt.Printf("Broadcaster currently has %d streamers and %d clients\n", len(bc.activeStreamers), bc.countClients())
			return &newClient, nil
		}
	}

	var newClient streamClient
	first := true
	streamBroadcasters := make(map[consts.Quality]*streamBroadcaster)
	for _, quality := range consts.Qualities {
		streamer, ok := streamers[quality]
		if !ok {
			continue
		}

		var cStreams []*streamClient
		fmt.Println("Streamer Type:", streamer.GetStreamType())
		if first {
			fmt.Println("Adding streamClient", clientID, "to new broadcast on stream", streamID, "with", quality, "quality")
			newClient = streamClient{
				clientID:       clientID,
				currentQuality: quality,
				wantedQuality:  quality,
				streamType:     streamer.GetStreamType(),
				done:           0,
				inputChan:      make(chan []byte, 16),
			}
			cStreams = append(cStreams, &newClient)
			first = false
		}

		sBroadcaster := &streamBroadcaster{
			quality:       streamer.GetStreamQuality(),
			inputStream:   streamer.GetOutputChannel(),
			clientStreams: cStreams,
		}

		streamBroadcasters[streamer.GetStreamQuality()] = sBroadcaster
	}

	sBcastHandler := &streamBroadcastHandler{
		streamID:           streamID,
		streamBroadcasters: streamBroadcasters,
	}

	// Associate all brodcasters with broadcastHandler
	for _, v := range streamBroadcasters {
		v.sBcastHandler = sBcastHandler

		// Start broadcasting routine
		go func(sBroadcaster *streamBroadcaster) {
			sBroadcaster.Broadcast()
			fmt.Println("Removing Broadcaster with quality", sBroadcaster.quality, "from stream", streamID)
			bc.cleanBroadcaster(streamID, sBroadcaster.quality)
		}(v)
	}

	bc.streamBroadcastHandlers[streamID] = sBcastHandler

	fmt.Printf("Broadcaster currently has %d streamers and %d clients\n", len(bc.activeStreamers), bc.countClients())
	return &newClient, nil
}
