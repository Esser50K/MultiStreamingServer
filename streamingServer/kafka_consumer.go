package streamingServer

import (
	"fmt"
	"github.com/Shopify/sarama"
)

struct KafkaConsumer {
	streamID			string
	streamQuality   	consts.Quality
	streamOutputChannel chan []byte
	isOpen              bool
	cons   				*cluster.Consumer
}

func NewKafkaConsumer(streamID string, streamQuality consts.Quality, ) *KafkaConsumer {
	
}