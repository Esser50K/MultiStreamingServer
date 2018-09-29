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
	broadcaster := broadcaster.NewBroadcaster(streamServer)
	httpStreamServer := http.NewStreamServer(broadcaster)
	go broadcaster.Start()
	httpStreamServer.PrepareStreamHandlers(streamPrefix, maxStreams)
	httpStreamServer.StartServer("", 80)
}
