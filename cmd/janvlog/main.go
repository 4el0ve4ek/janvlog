package main

import (
	"flag"
	"fmt"
	"janvlog/internal/janus"
	"janvlog/internal/listeners"
	"janvlog/internal/mail"
	"janvlog/internal/reporter"
	"janvlog/internal/stt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

type config struct {
	Mail struct {
		Host         string
		Port         int
		From         string
		Username     string
		PasswordFile string
	}

	WhisperTimestampdDomain string
	JanusDomain             string
}

func main() {
	exit := make(chan os.Signal, 1)
	signal.Notify(exit,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGTERM,
	)

	var cfgPath, regeneratePath string
	flag.StringVar(&cfgPath, "config", "config-dev.yaml", "config file")
	flag.StringVar(&regeneratePath, "regenerate", "", "what to regenerate")
	flag.Parse()

	slog.Info("using " + cfgPath + " for config")

	config, err := readConfig(cfgPath)
	if err != nil {
		panic(err)
	}

	slog.Info("using config: ", slog.Any("config", config))

	pwd, err := os.ReadFile(strings.ReplaceAll(config.Mail.PasswordFile, "$HOME", os.Getenv("HOME")))
	if err != nil {
		panic(err)
	}

	reporter := reporter.NewGenerator(
		stt.NewWhisperTimestampdClient(config.WhisperTimestampdDomain+"/transcribe"),
		mail.NewSender(mail.Config{
			Host:     config.Mail.Host,
			Port:     config.Mail.Port,
			From:     config.Mail.From,
			Username: config.Mail.Username,
			Password: strings.TrimSpace(string(pwd)),
		}),
	)

	defer reporter.Wait()

	if len(regeneratePath) > 0 {
		reporter.StartProcessing(regeneratePath)
		return
	}

	janusClient, err := janus.New(config.JanusDomain)
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

func readConfig(cfgPath string) (config, error) {
	file, err := os.ReadFile(cfgPath)
	if err != nil {
		return config{}, err
	}

	var parsed config
	err = yaml.Unmarshal(file, &parsed)
	if err != nil {
		return config{}, err
	}

	return parsed, nil
}
