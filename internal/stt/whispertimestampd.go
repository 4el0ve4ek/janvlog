package stt

import (
	"context"
	"encoding/json"
	"janvlog/internal/libs/xerrors"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type whisperTimestampdClient struct {
	endpoint string
}

func NewWhisperTimestampdClient(endpoint string) *whisperTimestampdClient {
	return &whisperTimestampdClient{
		endpoint: endpoint,
	}
}

func (w *whisperTimestampdClient) Process(fname string) (Response, error) {
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, w.endpoint, nil)
	if err != nil {
		return Response{}, xerrors.Wrap(err, "http.NewRequestWithContext")
	}

	pwd, err := os.Getwd()
	if err != nil {
		return Response{}, xerrors.Wrap(err, "os.Getwd")
	}

	cgi := req.URL.Query()
	cgi.Set("audio", filepath.Join(pwd, fname))
	req.URL.RawQuery = cgi.Encode()

	rawResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		return Response{}, xerrors.Wrap(err, "http.DefaultClient.Do")
	}
	defer rawResponse.Body.Close()

	var parsed whisperTimestampdResponse

	err = json.NewDecoder(rawResponse.Body).Decode(&parsed)
	if err != nil {
		return Response{}, xerrors.Wrap(err, "json.Decode whisper response")
	}

	var ret Response
	for _, seg := range parsed.Segments {
		ret.Parts = append(ret.Parts, ResponsePart{
			Offsets: struct {
				From int `json:"from"`
				To   int `json:"to"`
			}{
				From: int(seg.Start * float64(time.Second/time.Millisecond)),
				To:   int(seg.End * float64(time.Second/time.Millisecond)),
			},
			Text: seg.Text,
		})
	}

	return ret, nil
}

type whisperTimestampdResponse struct {
	Text     string `json:"text"`
	Segments []struct {
		ID    int     `json:"id"`
		Seek  int     `json:"seek"`
		Start float64 `json:"start"`
		End   float64 `json:"end"`
		Text  string  `json:"text"`
	} `json:"segments"`
}
