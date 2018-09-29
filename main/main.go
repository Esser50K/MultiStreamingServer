package main

import (
	"streamingServer/broadcaster"
	"streamingServer/broadcaster/http"
	"streamingServer/consumer/tcp"
)

func main() {
	maxStreams := 8
	streamPrefix := "stream"
	streamServer := tcp.NewConsumer("", 12345, maxStreams, streamPrefix)
	httpBroadcaster := http.NewBroadcaster(broadcaster.NewBroadcaster(streamServer))
	go httpBroadcaster.Start()
	httpBroadcaster.PrepareStreamHandlers(streamPrefix, maxStreams)
	httpBroadcaster.StartServer("", 80)
}
