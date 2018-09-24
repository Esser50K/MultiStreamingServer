package consumer

import (
	"consts"
)

// StreamConsumer must be implemented by strcuts representing stream consumers
type StreamConsumer interface {
	Start() error
	GetStream(streamID string) (StreamConnection, error)
	Stop() error
}

// StreamConnection must be implemented by structs representing stream connections
type StreamConnection interface {
	GetID() string
	GetType() int32
	GetOutputChan(consts.Quality) (<-chan []byte, error)
	HandleStream(consts.Quality) error
	Close() error
}
