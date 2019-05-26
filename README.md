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

## Development

You can set the project up wherever you want as long as the parent folder is called `src`, then you can prepend any `go` command with `GOPATH=$PWD/../..` to correctly point the compiler and dependency checkers to the project.

### Dependency Management

Use `dep` for dependency management. If you write code with newer go packages make sure to commit the changes
in the Gopkg.toml and Gopkg.lock files. Run `dep ensure` to install all dependencies and be able to compile the project.

If you're using new dependencies use `dep ensure` to install the new dependencies. If you need higher versions
of existing dependencies use `dep ensure --update` to get the newer versions.

Don't forget to prepend these commands with `GOPATH=$PWD/../..` and run them inside the root folder with a parent folder called `src` for them to work correctly.

### Compile

To compile simply run the basic `go build -o bin/streaming_server.exec main/main.go` (again with the GOPATH env variable prepended).

If you're compiling it to actually run on the raspberry specify the OS, Architecture (ARM) and ARM version: `GOOS=linux GOARCH=arm GOARM=7 go build -o bin/streaming_server.exec main/main.go`.

