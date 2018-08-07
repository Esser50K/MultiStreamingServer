# MultiStreamingServer
Golang Server that can receive multiple video streams in different formats and serve them over http

Note: This is still work in progress

## Motivation
I was working on a smart home/iot project with a cluster of raspberryPis (using docker and docker swarm). 
My goal was to set up multiple surveillance cameras (raspberryPi cameras) and stream the video data to a server which would then show these live feeds over http.

## How it Works
The server is double-ended.

On one side it listens for tcp connections on port 12345.
This is where the streaming clients connect to and send the video data (either in mjpeg or h264 format, in the future it will also support mpeg format)
The supported formats were chosen because these can deliver low-latency video streams to a browser. 
The browser can natively read and display a mjpeg stream.
For the other formats (h264 and in the future mpeg) we need to use javascript decoders that I found in the following projects:
- [H264 decoder](https://github.com/mbebenita/Broadway)
- [MPEG Decoder](https://github.com/phoboslab/jsmpeg)

(The html and javascript code from these projects is not yet included in this one)

On the other side there is an http server listening on port 80.
The server then distributes the incoming video streams to the various http clients that request it.

## Docker
A Dockerfile and .yml file for docker-swarm are included in the project.

The server is configured as a global service, this means it will have one running instance on each node of the swarm.

The streaming clients send their video data to all running servers on port 12345.
Docker will automatically load-balance the multiple server instances when someone tries to see the feed over http via the browser.

## Future Work
- Publish client side code that sends video streams to the server
- Develop a Kafka consumer for a more stable and less bandwidth consuming setup.
