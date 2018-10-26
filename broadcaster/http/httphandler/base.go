package httphandler

import (
	"StreamingServer/consts"
	"fmt"
	"net/http"
)

type HttpStreamHandler func(streamChan chan []byte, writer http.ResponseWriter, request *http.Request) (bool, error)

func GetHTTPStreamHandler(streamType consts.StreamType) (HttpStreamHandler, error) {
	switch streamType {
	case consts.StreamMJPG:
		return HandleJpegStreamRequest, nil
	case consts.StreamH264:
		return HandleH264StreamRequest, nil
	default:
		return nil, fmt.Errorf("No handler for stream type %d", streamType)
	}
}

func SendHTTPHeaders(streamType consts.StreamType, writer http.ResponseWriter) error {
	switch streamType {
	case consts.StreamMJPG:
		SendJpegHeaders(writer)
		return nil
	case consts.StreamH264:
		SendH264Headers(writer)
		return nil
	default:
		return fmt.Errorf("No headers for stream type %d", streamType)
	}
}
