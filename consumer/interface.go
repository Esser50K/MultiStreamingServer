package consumer

import "StreamingServer/consts"

// StreamConsumer must be implemented by strcuts representing stream consumers
type StreamConsumer interface {
	Start() error
	GetStream(streamID string) (StreamConnection, error)
	Stop() error
}

// StreamConnection must be implemented by structs representing stream connections
type StreamConnection interface {
	GetID() string
	GetType() consts.StreamType
	GetOutputChan(consts.Quality) (<-chan []byte, error)
	HandleStream(consts.Quality) error
	AddConnection(consts.Quality, interface{}) error
	Close(consts.Quality) error
	IsOpen() bool
}

type BaseStreamConnection struct {
	streamID   string
	streamType consts.StreamType
}

func (sc *BaseStreamConnection) GetID() string {
	return sc.streamID
}

func (sc *BaseStreamConnection) GetType() consts.StreamType {
	return sc.streamType
}

func NewBaseStreamConnection(streamID string, streamType consts.StreamType) BaseStreamConnection {
	return BaseStreamConnection{
		streamID:   streamID,
		streamType: streamType,
	}
}
