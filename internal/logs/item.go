package logs

import "time"

type Item struct {
	RoomID        float64
	ParticipantID uint64
	DisplayName   string
	Time          time.Time
	Message       Message
	AudioFile     string `json:",omitempty"`
	Speech        string `json:",omitempty"`
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
