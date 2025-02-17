package reporter

import (
	"janvlog/internal/logs"
	"janvlog/internal/stt"
	"log"
	"path"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

func NewGenerator(stt stt.Client) *Generator {
	return &Generator{
		stt: stt,
	}
}

type Generator struct {
	wg  sync.WaitGroup
	stt stt.Client
}

func (g *Generator) StartProcessing(rawLog string) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		oldStorage, _ := logs.NewStorage(rawLog)
		items := oldStorage.Items()
		oldStorage.Close()

		resItems := g.process(items)
		resItems = g.fillNames(resItems)

		storage, _ := logs.NewStorage(filepath.Join("logs", "processed", path.Base(rawLog)))
		storage.Clear()
		storage.Add(resItems...)
		storage.Close()
	}()
}

func (g *Generator) Wait() {
	g.wg.Wait()
}

func (g *Generator) process(items []logs.Item) []logs.Item {
	talking := make(map[uint64]time.Time)
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

	var ret []logs.Item

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
	names := make(map[uint64]string)
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
