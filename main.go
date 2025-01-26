package main

import (
	"fmt"
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

	jc := NewJanusClient(host)
	defer jc.Close()

	videoroomListener, err := NewVideoroomListener(jc)
	if err != nil {
		panic(err)
	}
	defer func() {
		fmt.Println(videoroomListener.Close())
	}()

	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		if _, err := jc.session.KeepAlive(); err != nil {
			panic(err)
		}

		select {
		case <-tick.C:
		case <-exit:
			return
		}
	}
}
