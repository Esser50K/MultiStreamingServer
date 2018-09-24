package main

import (
	"streamingServer/broadcaster"
	"streamingServer/broadcaster/http"
	"streamingServer/consumer/tcp"
)

func main() {
	maxStreams := 8
	streamServer := tcp.NewConsumer("", 12345, maxStreams)
	broadcaster := broadcaster.NewBroadcaster(streamServer)
	httpStreamServer := http.NewHttpStreamServer(broadcaster)
	go broadcaster.Start()
	httpStreamServer.PrepareStreamHandlers("stream", maxStreams)
	httpStreamServer.StartServer("", 80)
}
