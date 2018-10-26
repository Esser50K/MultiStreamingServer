package consumer

import (
	"StreamingServer/consts"
	"StreamingServer/consumer"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	cluster "github.com/bsm/sarama-cluster"
)

const (
	// Offset Keywords
	offsetOldest    = "oldest"
	offsetNewest    = "newest"
	defaultOffset   = offsetNewest
	consumerOffsets = "__consumer_offsets"

	// Version Keywords
	version0_10_1 = "0.10.1"
	version1_1_0  = "1.1.0"

	// Keyword Args Keys
	topicsKey         = "topics"
	groupIDKey        = "group_id"
	consumerOffsetKey = "consumer_offset"
	backOffTimeKey    = "backoff_time"
)

type KafkaArgs map[string]string

func getOrDefault(key, default_val string, kwargs KafkaArgs) string {
	val, ok := kwargs[key]
	if !ok {
		return default_val
	}

	return val
}

func randomGroupID(len int) string {
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		bytes[i] = byte(65 + rand.Intn(25)) //A=65 and Z = 65+25
	}
	return string(bytes)
}

type KafkaConsumer struct {
	topicStreams map[string]*KafkaStreamConnection
}

// TODO add support for TLS Config
func NewKafkaConsumer(kafkaBrokers string, kwargs ...KafkaArgs) (*KafkaConsumer, error) {
	var args KafkaArgs
	if len(kwargs) > 0 {
		args = kwargs[0]
	}

	config := cluster.NewConfig()
	config.Config.Version = sarama.V0_10_1_0 // TODO add config key
	config.Consumer.Return.Errors = false
	config.Consumer.Fetch.Default = 1024 * 1024 * 2

	consumerOffset := getOrDefault(consumerOffsetKey, defaultOffset, args)
	switch consumerOffset {
	case offsetOldest:
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	case offsetNewest:
		config.Consumer.Offsets.Initial = sarama.OffsetNewest
	default:
		return nil, fmt.Errorf("Offset must be either newest or oldest")
	}

	backoff, err := time.ParseDuration(getOrDefault(backOffTimeKey, "100ms", args))
	if err != nil {
		config.Config.Consumer.Retry.Backoff = backoff
	}

	topics := getOrDefault(topicsKey, "", args)
	if topics == "" {
		return nil, fmt.Errorf("topics must exist in KafkaArgs as comma seperated string")
	}

	groupID := getOrDefault(groupIDKey, randomGroupID(6), args)
	streamConnections := make(map[string]*KafkaStreamConnection)
	for _, topic := range strings.Split(topics, ",") {
		// Create Kafka Consumer
		kafkaConsumer, err := cluster.NewConsumer(
			strings.Split(kafkaBrokers, ","),
			groupID,
			[]string{topic},
			config,
		)

		if err != nil {
			return nil, err
		}

		// Create/Add StreamConnection
		streamInfo := strings.Split(topic, "_")
		if len(streamInfo) < 2 {
			return nil, fmt.Errorf("topic name needs to specify stream quality at the end seperated by '_'")
		}

		streamName := strings.Join(streamInfo[:len(streamInfo)-2], "")
		streamType := streamInfo[len(streamInfo)-2]
		qualityString := streamInfo[len(streamInfo)-1]

		_, exists := consts.StreamTypes[consts.StreamType(streamType)]
		if !exists {
			return nil, fmt.Errorf("no such stream type %s", streamType)
		}

		quality, err := consts.GetQualityFromString(qualityString)
		if err != nil {
			return nil, err
		}

		stream, ok := streamConnections[streamName]
		if !ok {
			stream = NewKafkaStreamConnection(streamName, consts.StreamType(streamType), kafkaConsumer, quality)
			streamConnections[streamName] = stream
		}
		stream.AddConnection(quality, kafkaConsumer)
	}

	consumer := KafkaConsumer{streamConnections}
	return &consumer, nil
}

func (sc *KafkaConsumer) GetStream(streamID string) (consumer.StreamConnection, error) {
	stream, ok := sc.topicStreams[streamID]
	if !ok {
		return nil, fmt.Errorf("No stream registered with id '%s\n", streamID)
	}

	return stream, nil
}

func (sc *KafkaConsumer) Start() error {
	for _, connection := range sc.topicStreams {
		for _, quality := range connection.GetQualities() {
			go func(c *KafkaStreamConnection) {
				c.HandleStream(quality)
			}(connection)
		}
	}

	return nil
}

func (sc *KafkaConsumer) Stop() error {
	for _, connection := range sc.topicStreams {
		err := connection.CloseAll()
		if err != nil {
			return err
		}
	}

	return nil
}
