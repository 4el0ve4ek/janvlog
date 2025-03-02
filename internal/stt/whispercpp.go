package stt

import (
	"encoding/json"
	"fmt"
	"janvlog/internal/libs/xerrors"
	"os"
	"os/exec"
)

func NewWhisperCPPClient(path string, weights string) *whisperCPPClient {
	return &whisperCPPClient{
		path:    path,
		weights: weights,
	}
}

type whisperCPPClient struct {
	path, weights string
}

func (w *whisperCPPClient) Process(fname string) (Response, error) {
	preprocessedFile, err := whisperPreprocess(fname)
	if err != nil {
		return Response{}, err
	}
	defer os.Remove(preprocessedFile)

	cmd := exec.Command(w.path, "-m", w.weights, "-f", preprocessedFile, "-oj", "--language", "ru") //nolint:gosec

	stderr := attachStderr(cmd)
	if err := cmd.Run(); err != nil {
		return Response{}, fmt.Errorf("ffmpeg error: %s, %w", stderr.String(), err)
	}

	whisperRes, err := os.Open(preprocessedFile + ".json")
	if err != nil {
		return Response{}, xerrors.Wrap(err, "os.Open")
	}
	defer whisperRes.Close()

	var whisperResponse whispercliReponse

	err = json.NewDecoder(whisperRes).Decode(&whisperResponse)
	if err != nil {
		return Response{}, xerrors.Wrap(err, "json.Decode whisper response")
	}

	var ret Response

	for _, part := range whisperResponse.Transcription {
		ret.Parts = append(ret.Parts, ResponsePart{
			Offsets: part.Offsets,
			Text:    part.Text,
		})
	}

	return ret, nil
}

type whispercliReponse struct {
	Transcription []struct {
		Timestamps struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"timestamps"`
		Offsets struct {
			From int `json:"from"`
			To   int `json:"to"`
		} `json:"offsets"`
		Text string `json:"text"`
	} `json:"transcription"`
}
