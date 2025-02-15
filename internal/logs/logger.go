package logs

import (
	"encoding/json"
	"io"
	"os"
	"time"
)

type Writer interface {
	io.Closer
	Write(LogItem)
}

func NewWriter(fname string) (*writer, error) {
	file, err := os.Create(fname + ".jsonl")
	if err != nil {
		return nil, err
	}

	return &writer{
		file: file,
	}, nil
}

type writer struct {
	file io.WriteCloser
}

func (l *writer) Close() error {
	return l.file.Close()
}

func (l *writer) Write(p LogItem) {
	p.Time = time.Now()

	encoder := json.NewEncoder(os.Stdout)
	if l.file != nil {
		encoder = json.NewEncoder(l.file)
	}

	encoder.Encode(p)
}

type LogItem struct {
	RoomID        float64
	ParticipantID uint64
	DisplayName   string
	Time          time.Time
	Message       Message
	AudioFile     string `json:",omitempty"`
}

type Message string

const (
	MessageJoined           Message = "joined with camera"
	MessageJoinedWithoutCam Message = "joined without camera"
	MessageLeft             Message = "left"
	MessageDisableCamera    Message = "disable camera"
	MessageEnableCamera     Message = "enable camera"
	MessageEmptyRoom        Message = "every one left"
)
