FROM alpine:latest
COPY public /
COPY bin/streaming_server.exec /
ENTRYPOINT ["./streaming_server.exec"]
EXPOSE 80 12345