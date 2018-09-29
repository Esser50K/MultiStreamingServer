package handler

import (
	"fmt"
	"net/http"
)

type HttpStreamHandler func(streamChan chan []byte, writer http.ResponseWriter, request *http.Request) (bool, error)

func GetHTTPStreamHandler(streamType int32) (HttpStreamHandler, error) {
	switch streamType {
	case 0:
		return HandleJpegStreamRequest, nil
	case 1:
		return HandleH264StreamRequest, nil
	default:
		return nil, fmt.Errorf("No handler for stream type %d", streamType)
	}
}

func SendHTTPHeaders(streamType int32, writer http.ResponseWriter) error {
	switch streamType {
	case 0:
		SendJpegHeaders(writer)
		return nil
	case 1:
		SendH264Headers(writer)
		return nil
	default:
		return fmt.Errorf("No headers for stream type %d", streamType)
	}
}
