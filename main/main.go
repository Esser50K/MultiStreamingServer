package main

import "streamingServer"

func main() {
	maxStreams := 8
	streamServer := streamingServer.NewStreamingServer(maxStreams)
	broadcaster := streamingServer.NewBroadcaster(*streamServer)
	httpStreamServer := streamingServer.NewHttpStreamServer(broadcaster)
	go broadcaster.ListenForStreamers("", 12345)
	httpStreamServer.PrepareStreamHandlers("stream", maxStreams)
	httpStreamServer.StartServer("", 80)
}
