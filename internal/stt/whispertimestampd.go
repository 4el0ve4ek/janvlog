package stt

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

type whisperTimestampdClient struct {
	endpoint string
}

func NewWhisperTimestampdClient(endpoint string) Client {
	return &whisperTimestampdClient{
		endpoint: endpoint,
	}
}

func (w *whisperTimestampdClient) Process(fname string) (Response, error) {
	req, err := http.NewRequest("POST", w.endpoint, nil)
	if err != nil {
		return Response{}, err
	}

	pwd, err := os.Executable()
	if err != nil {
		return Response{}, err
	}

	cgi := req.URL.Query()
	cgi.Set("audio", filepath.Join(filepath.Dir(pwd), fname))
	req.URL.RawQuery = cgi.Encode()

	rawResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer rawResponse.Body.Close()

	var parsed whisperTimestampdResponse
	err = json.NewDecoder(rawResponse.Body).Decode(&parsed)
	if err != nil {
		return Response{}, err
	}

	var ret Response
	for _, seg := range parsed.Segments {
		ret.Parts = append(ret.Parts, ResponsePart{
			Offsets: struct {
				From int `json:"from"`
				To   int `json:"to"`
			}{
				From: int(seg.Start * 1000),
				To:   int(seg.End * 1000),
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
