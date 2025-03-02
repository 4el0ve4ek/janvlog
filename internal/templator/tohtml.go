package templator

import (
	"bytes"
	"janvlog/internal/logs"
	"time"
)

func GenerateHTML(items []logs.Item) []byte {
	byRoomEvents := groupby(items, func(item logs.Item) logs.RoomID {
		return item.RoomID
	})

	var report bytes.Buffer

	report.WriteString(`<!DOCTYPE html>`)
	report.WriteString(`<html lang="en">`)
	report.WriteString(`<body>`)

	for roomID, roomEvents := range byRoomEvents {
		GenerateEventsHTML(&report, roomID.String(), roomEvents)
	}

	report.WriteString(`</body>`)
	report.WriteString(`</html>`)

	return report.Bytes()
}

func GenerateEventsHTML(res *bytes.Buffer, roomID string, items []logs.Item) []byte {
	res.WriteString(`<h2>Room id is ` + roomID + `. List of Events: </h2>`)
	res.WriteString(`<ul>`)

	for _, item := range items {
		res.WriteString(`<li>`)
		res.WriteString(item.Time.Format(time.TimeOnly))
		res.WriteRune(' ')

		if item.DisplayName != "" {
			res.WriteString(item.DisplayName)
			res.WriteString(": ")
		}

		if item.Message == logs.MessageSpeech {
			res.WriteString(item.Speech)
		} else {
			res.WriteString(string(item.Message))
		}

		res.WriteString(`</li>`)
	}

	res.WriteString(`</ul>`)
	res.WriteString(`</div>`)

	return res.Bytes()
}

func groupby[T any, K comparable](items []T, key func(T) K) map[K][]T {
	result := make(map[K][]T)
	for _, item := range items {
		result[key(item)] = append(result[key(item)], item)
	}

	return result
}

// <div id="log-viewer">
// <h2>Room id is 1234. List of Events: </h2>
// <ul>
// <li>17:53:51.447 danya1: joined without camera</li>
// <li>17:53:53.612 danya1: enable camera</li>
// <li>17:53:55.152 danya1:  Привет! Раз, два, три, четыре, пять. Я иду искать.</li>
// <li>17:54:19.029 notDanya: joined without camera</li>
// <li>17:54:20.192 notDanya: enable camera</li>
// <li>17:54:21.052 notDanya:  5 раз 4-5 снова</li>
// <li>17:54:27.392 notDanya:  Я тебя побиваю, не говори вместе со мной</li>
// <li>17:54:28.432 danya1:  Я тебя побиваю, не говори ко мне со мной. Я ухожу, мне может сбегать за чем-то.</li>
// <li>17:54:31.292 notDanya:  Я ухожу, мне нужно сбегать за чем-то</li>
// <li>17:54:36.449 danya1: disable camera</li>
// <li>17:54:36.712 notDanya:  А он мне это нашел, пока</li>
// <li>17:54:42.551 notDanya: left</li>
// <li>17:54:46.755 danya1: enable camera</li>
// <li>17:54:46.755 danya1:  а тут больше нет и его я ухожу</li>
// <li>17:54:51.840 danya1: left</li>
// <li>17:54:51.840 : every one left</li>
// </ul>
// </div>
