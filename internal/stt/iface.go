package stt

type Client interface {
	Process(fname string) (Response, error)
}

type Response struct {
	Parts []ResponsePart
}

type ResponsePart struct {
	Offsets struct {
		From int `json:"from"`
		To   int `json:"to"`
	} `json:"offsets"`
	Text string `json:"text"`
}
