package main

import (
	"fmt"
	"janvlog/internal/janus"
	"janvlog/internal/listeners"
	"janvlog/internal/reporter"
	"janvlog/internal/stt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	exit := make(chan os.Signal, 1)
	signal.Notify(exit,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGTERM,
	)

	const (
		host           = "84.201.174.125"
		whispercpp     = "/Users/da-aksenov/self-thing/whisper.cpp"
		whispercli     = whispercpp + "/build/bin/whisper-cli"
		whisperweights = whispercpp + "/models/ggml-large-v3-turbo-q8_0.bin"
	)

	reporter := reporter.NewGenerator(stt.NewWhisperTimestampdClient("http://localhost:8080/transcribe"))
	defer reporter.Wait()

	if len(os.Args) == 3 && os.Args[1] == "regenerate" {
		reporter.StartProcessing(os.Args[2])
		return
	}

	jc, err := janus.New(host)
	if err != nil {
		panic(err)
	}
	defer jc.Close()

	videoroomListener, err := listeners.NewVideoroom(jc, reporter)
	if err != nil {
		panic(err)
	}
	defer func() {
		fmt.Println(videoroomListener.Close())
	}()

	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		if _, err := jc.KeepAlive(); err != nil {
			panic(err)
		}

		select {
		case <-tick.C:
		case <-exit:
			return
		}
	}
}
