FROM alpine:latest
COPY streamingServer /
WORKDIR streamingServer
ENTRYPOINT ["./superMain.go"]
EXPOSE 80 12345
