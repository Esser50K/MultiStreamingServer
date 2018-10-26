package consumer

import (
	"StreamingServer/consts"
	"StreamingServer/consumer"
	"fmt"

	cluster "github.com/bsm/sarama-cluster"
)

type KafkaStreamConnection struct {
	consumer.BaseStreamConnection
	kafkaConsumers map[consts.Quality]*cluster.Consumer
	streamChanMap  map[consts.Quality](chan []byte)
	isOpen         bool
}

func NewKafkaStreamConnection(streamID string, streamType consts.StreamType, kafkaConsumer *cluster.Consumer, quality consts.Quality) *KafkaStreamConnection {
	streamConnection := KafkaStreamConnection{
		BaseStreamConnection: consumer.NewBaseStreamConnection(streamID, streamType),
		kafkaConsumers:       make(map[consts.Quality]*cluster.Consumer),
		streamChanMap:        make(map[consts.Quality](chan []byte)),
	}

	streamConnection.kafkaConsumers[quality] = kafkaConsumer
	streamConnection.streamChanMap[quality] = make(chan []byte, 4)
	return &streamConnection
}

func (sc *KafkaStreamConnection) GetOutputChan(quality consts.Quality) (<-chan []byte, error) {
	ch, ok := sc.streamChanMap[quality]
	if !ok {
		return nil, fmt.Errorf("no stream for quality %s", quality)
	}

	return ch, nil
}

func (sc *KafkaStreamConnection) AddConnection(quality consts.Quality, kafkaConsumer interface{}) error {
	consumer, ok := kafkaConsumer.(*cluster.Consumer)
	if !ok {
		return fmt.Errorf("second arfument must be of type (chan []byte)")
	}

	sc.kafkaConsumers[quality] = consumer
	sc.streamChanMap[quality] = make(chan []byte, 4)
	return nil
}

func (sc *KafkaStreamConnection) AddDataToStream(data []byte, quality consts.Quality) error {
	qualityChannel, ok := sc.streamChanMap[quality]
	if !ok {
		return fmt.Errorf("no stream for quality %d", quality)
	}

	qualityChannel <- data
	return nil
}

func (sc *KafkaStreamConnection) HandleStream(quality consts.Quality) error {
	sc.isOpen = true
	consumer, ok := sc.kafkaConsumers[quality]
	if !ok {
		return fmt.Errorf("no consumer for quality %s", quality)
	}

	for msg := range consumer.Messages() {
		select {
		case sc.streamChanMap[quality] <- msg.Value:
		default:
			<-sc.streamChanMap[quality]
			sc.streamChanMap[quality] <- msg.Value
		}
	}
	return nil
}

func (sc *KafkaStreamConnection) Close(quality consts.Quality) error {
	channel, ok := sc.streamChanMap[quality]
	consumer, ok := sc.kafkaConsumers[quality]
	if !ok {
		return fmt.Errorf("No stream kafka consumer with quality %d", quality)
	}

	close(channel)
	err := consumer.Close()
	if err != nil {
		return err
	}

	return nil
}

func (sc *KafkaStreamConnection) CloseAll() error {
	for quality, _ := range sc.streamChanMap {
		err := sc.Close(quality)
		if err != nil {
			return err
		}
	}

	sc.isOpen = false
	return nil
}

func (sc *KafkaStreamConnection) GetQualities() []consts.Quality {
	var qualities []consts.Quality
	for key, _ := range sc.streamChanMap {
		qualities = append(qualities, key)
	}

	return qualities
}

func (sc *KafkaStreamConnection) IsOpen() bool {
	return sc.isOpen
}
