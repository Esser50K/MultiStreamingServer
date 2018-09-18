package streamingServer

import (
	"streamingServer/consts"

	cluster "github.com/bsm/sarama-cluster"
)

type KafkaConsumer struct {
	streamID            string
	streamQuality       consts.Quality
	streamOutputChannel chan []byte
	isOpen              bool
	cons                *cluster.Consumer
}

func NewKafkaConsumer(streamID string, streamQuality consts.Quality) *KafkaConsumer {
	return nil
}
