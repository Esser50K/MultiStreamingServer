package tcphandler

import (
	"StreamingServer/consts"
	"errors"
	"fmt"
	"net"
)

type TCPStreamHandler func(connection *net.TCPConn, outputChan chan []byte) error

func GetTCPStreamHandleFunc(streamType consts.StreamType) (TCPStreamHandler, error) {
	switch streamType {
	case consts.StreamMJPG:
		return HandleJpegStream, nil
	case consts.StreamH264:
		return HandleH264Stream, nil
	default:
		return nil, errors.New(fmt.Sprintf("No handler for stream type %d", streamType))
	}
}
