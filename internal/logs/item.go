package logs

import (
	"strconv"
	"time"
)

type RoomID float64

func (id RoomID) String() string {
	return strconv.FormatFloat(float64(id), 'f', -1, 64)
}

type ParticipantID uint64

func (id ParticipantID) String() string {
	return strconv.FormatUint(uint64(id), 10)
}

type Item struct {
	RoomID        RoomID        `json:"RoomID"`
	ParticipantID ParticipantID `json:"ParticipantID"`
	DisplayName   string        `json:"DisplayName"`
	Time          time.Time     `json:"Time"`
	Message       Message       `json:"Message"`
	AudioFile     string        `json:"AudioFile,omitempty"`
	Speech        string        `json:"Speech,omitempty"`
}

type Message string

const (
	MessageJoined           Message = "joined with camera"
	MessageJoinedWithoutCam Message = "joined without camera"
	MessageLeft             Message = "left"
	MessageDisableCamera    Message = "disable camera"
	MessageEnableCamera     Message = "enable camera"
	MessageEmptyRoom        Message = "every one left"
	MessageSpeech           Message = "speech"
)
