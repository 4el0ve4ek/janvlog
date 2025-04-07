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

	report.WriteString("<!DOCTYPE html>")
	report.WriteString(`<html lang="en">`)
	report.WriteString("<body>")

	for roomID, roomEvents := range byRoomEvents {
		GenerateEventsHTML(&report, roomID.String(), roomEvents)
	}

	report.WriteString("</body>")
	report.WriteString("</html>")

	return report.Bytes()
}

func GenerateEventsHTML(res *bytes.Buffer, roomID string, items []logs.Item) []byte {
	res.WriteString("<h2>Room id is " + roomID + "</h2> <h2> Participants: </h2>")

	uniqueParticipants := make(map[string]struct{})
	res.WriteString("<ul>\n")
	for _, item := range items {
		if _, ok := uniqueParticipants[item.DisplayName]; ok {
			continue
		}
		if item.Message != logs.MessageJoined && item.Message != logs.MessageJoinedWithoutCam {
			continue
		}

		res.WriteString("<li>\n")
		res.WriteString(item.DisplayName)
		res.WriteString("(")
		res.WriteString("platform: " + item.Metadata["platform"].(string))
		res.WriteString(", useragent: " + item.Metadata["user-agent"].(string))
		res.WriteString(")")
		res.WriteString("\n</li>\n")

	}
	res.WriteString("</ul>\n")

	res.WriteString("<h2> List of Events: </h2>\n")

	res.WriteString("<ul>\n")
	for _, item := range items {
		res.WriteString("<li>\n")
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

		res.WriteString("\n</li>\n")
	}

	res.WriteString("</ul>\n")
	res.WriteString("</div>\n")

	return res.Bytes()
}

func groupby[T any, K comparable](items []T, key func(T) K) map[K][]T {
	result := make(map[K][]T)
	for _, item := range items {
		result[key(item)] = append(result[key(item)], item)
	}

	return result
}
