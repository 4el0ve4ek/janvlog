package main

import (
	"fmt"
	"janvlog/internal/janus"
	"janvlog/internal/listeners"
	"janvlog/internal/mail"
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

	const host = "84.201.174.125"

	reporter := reporter.NewGenerator(
		stt.NewWhisperTimestampdClient("http://localhost:8080/transcribe"),
		mail.NewSender(mail.Config{
			Host:     "smtp.yandex.ru",
			Port:     587,
			From:     "aksenoff.dany@yandex.ru",
			Username: "aksenoff.dany",
			Password: os.Getenv("YAPASSWORD"),
		}),
	)

	defer reporter.Wait()

	if len(os.Args) == 3 && os.Args[1] == "regenerate" {
		reporter.StartProcessing(os.Args[2])
		return
	}

	janusClient, err := janus.New(host)
	if err != nil {
		panic(err)
	}
	defer janusClient.Close()

	videoroomListener, err := listeners.NewVideoroom(janusClient, reporter)
	if err != nil {
		panic(err)
	}

	defer func() {
		fmt.Println(videoroomListener.Close())
	}()

	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		if _, err := janusClient.KeepAlive(); err != nil {
			panic(err)
		}

		select {
		case <-tick.C:
		case <-exit:
			return
		}
	}
}
