package consts

import (
	"errors"
	"fmt"
)

type Quality int32
type StreamType string

const (
	// Streaming Quality Consts
	HighQuality Quality = 1
	LowQuality  Quality = 0

	// Stream Type Consts
	StreamH264 StreamType = "h264"
	StreamMJPG StreamType = "mjpg"
)

var (
	Qualities = map[Quality]bool{
		LowQuality:  true,
		HighQuality: true,
	}

	StreamTypeIDs = map[int]StreamType{
		1: StreamH264,
		0: StreamMJPG,
	}

	StreamTypes = map[StreamType]bool{
		StreamH264: true,
		StreamMJPG: true,
	}
)

func GetQualityFromString(qual string) (Quality, error) {
	switch qual {
	case "low":
		return LowQuality, nil
	case "high":
		return HighQuality, nil
	default:
		return -1, errors.New(fmt.Sprintf("No Quality index for: %s", qual))
	}
}
