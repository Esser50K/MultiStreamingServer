package main

import (
	broadcaster "StreamingServer/broadcaster/http"
	consumer "StreamingServer/consumer/tcp"
)

func main() {
	maxStreams := 8
	streamPrefix := "stream"
	streamServer := consumer.NewTCPConsumer("", 12345, maxStreams, streamPrefix)
	httpBroadcaster := broadcaster.NewHTTPBroadcaster(streamServer)
	go httpBroadcaster.Start()
	httpBroadcaster.PrepareStreamHandlers(streamPrefix, maxStreams)
	httpBroadcaster.StartServer("", 80)
}
