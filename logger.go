package main

import (
	"encoding/json"
	"io"
	"os"
	"time"
)

func NewLogWriter(fname string) (*logWriter, error) {
	file, err := os.Create(fname + ".jsonl")
	if err != nil {
		return nil, err
	}

	return &logWriter{
		file: file,
	}, nil
}

type logWriter struct {
	file io.WriteCloser
}

func (l *logWriter) Close() error {
	return l.file.Close()
}

func (l *logWriter) Write(p LogItem) {
	p.Time = time.Now()
	encoder := json.NewEncoder(os.Stdout)
	if l.file != nil {
		json.NewEncoder(l.file)

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
)
