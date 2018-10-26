package main

import (
	broadcaster "StreamingServer/broadcaster/http"
	consumer "StreamingServer/consumer/kafka"
)

func main() {
	maxStreams := 8
	streamPrefix := "stream"
	kwargs := make(consumer.KafkaArgs)
	kwargs["group_id"] = "rpi2"
	kwargs["topics"] = "stream0_h264_low"
	consumer, _ := consumer.NewKafkaConsumer("kafka02:9092,kafka03:9092,kafka04:9092", kwargs)
	httpBroadcaster := broadcaster.NewHTTPBroadcaster(consumer)
	go httpBroadcaster.Start()
	httpBroadcaster.PrepareStreamHandlers(streamPrefix, maxStreams)
	httpBroadcaster.StartServer("", 80)
}
