package reporter

import (
	"fmt"
	"janvlog/internal/logs"
	"janvlog/internal/mail"
	"janvlog/internal/stt"
	"janvlog/internal/templator"
	"log"
	"path"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

func NewGenerator(stt stt.Client, mail *mail.Sender) *Generator {
	return &Generator{
		stt:  stt,
		mail: mail,
	}
}

type Generator struct {
	wg   sync.WaitGroup
	stt  stt.Client
	mail *mail.Sender
}

func (g *Generator) StartProcessing(rawLog string) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()

		items, err := logs.ItemsFromStorage(rawLog)
		if err != nil {
			log.Println("error reading storage", err)
		}

		if len(items) == 0 {
			return
		}

		resItems := g.process(items)
		resItems = g.fillNames(resItems)

		storage, err := logs.NewStorage(filepath.Join("logs", "processed", path.Base(rawLog)))
		if err != nil {
			log.Println("error creating storage", err)
		}

		storage.Add(resItems...)
		storage.Close()

		message := templator.GenerateHTML(resItems)
		fmt.Println(string(message))

		err = g.mail.SendHTML("aksenoff.dany@yandex.ru", fmt.Sprintf("Generated remort for room - %v ", resItems[0].RoomID.String()), message)
		if err != nil {
			log.Println("error sending email", err)
		}
	}()
}

func (g *Generator) Wait() {
	g.wg.Wait()
}

func (g *Generator) process(items []logs.Item) []logs.Item {
	talking := make(map[logs.ParticipantID]time.Time)
	ret := slices.Concat(items)

	for _, item := range items {
		switch item.Message {
		case logs.MessageJoined, logs.MessageEnableCamera:
			talking[item.ParticipantID] = item.Time
		case logs.MessageLeft, logs.MessageDisableCamera:
			talkStartedAt, isTalking := talking[item.ParticipantID]
			if !isTalking {
				log.Println("no talk started for log", item)
				continue
			}

			if item.AudioFile == "" {
				log.Println("no audio file for log", item)
			} else {
				ret = append(ret, g.generateSpeech(talkStartedAt, item)...)
			}

			delete(talking, item.ParticipantID)
		default:
		}
	}

	slices.SortFunc(ret, func(a, b logs.Item) int {
		return a.Time.Compare(b.Time)
	})

	return ret
}

func (g *Generator) generateSpeech(talkStartedAt time.Time, item logs.Item) []logs.Item {
	speech, err := g.stt.Process(item.AudioFile)
	if err != nil {
		log.Println(err)
		return nil
	}

	ret := make([]logs.Item, 0, len(speech.Parts))

	for _, part := range speech.Parts {
		ret = append(ret, logs.Item{
			RoomID:        item.RoomID,
			ParticipantID: item.ParticipantID,
			DisplayName:   item.DisplayName,
			Time:          talkStartedAt.Add(time.Duration(part.Offsets.From) * time.Millisecond),
			Message:       logs.MessageSpeech,
			AudioFile:     item.AudioFile,
			Speech:        part.Text,
		})
	}

	return ret
}

func (g *Generator) fillNames(items []logs.Item) []logs.Item {
	names := make(map[logs.ParticipantID]string)

	for _, item := range items {
		if item.DisplayName == "" {
			continue
		}

		names[item.ParticipantID] = item.DisplayName
	}

	for i, item := range items {
		items[i].DisplayName = names[item.ParticipantID]
	}

	return items
}
