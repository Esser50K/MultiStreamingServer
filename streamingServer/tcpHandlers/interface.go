package tcpHandlers

import (
  "fmt"
  "net"
  "errors"
  "streamingServer/consts"
)

type TCPStreamHandler func(connection *net.TCPConn, outputChan chan []byte) error

func GetTCPStreamHandleFunc(streamType int32) (TCPStreamHandler, error) {
  switch streamType {
  case 0:
    return HandleJpegStream, nil
  case 1:
    return HandleH264Stream, nil
  default:
    return nil, errors.New(fmt.Sprintf("No handler for stream type %d", streamType))
  }
}

func GetStreamQuality(qualityIndex int32) (consts.Quality, error) {
  switch qualityIndex {
  case 0:
    return consts.LowQuality, nil
  case 1:
    return consts.HighQuality, nil
  default:
    return "", errors.New(fmt.Sprintf("No consts.Quality indication for index: %d", qualityIndex))
  }
}

func GetQualityIndex(qual consts.Quality) (int32, error) {
  fmt.Println("Quality is: " + string(qual) + ".")
  switch qual {
  case consts.LowQuality:
    return 0, nil
  case consts.HighQuality:
    return 1, nil
  default:
    return -1, errors.New(fmt.Sprintf("No consts.Quality index for: %s", qual))
  }
}
