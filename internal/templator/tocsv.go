package templator

import (
	"bytes"
	"encoding/csv"
	"janvlog/internal/logs"
	"log/slog"
)

func GenerateCSV(items []logs.Item) []byte {
	var b bytes.Buffer

	w := csv.NewWriter(&b)
	err := w.Write([]string{
		"RoomID",
		"RoomName",
		"ParticipantID",
		"DisplayName",
		"Time",
		"Message",
		"Speech",
		"UserAgent",
		"Mail",
	})
	if err != nil {
		slog.Info("error writing csv header", slog.Any("err", err))
		return nil
	}

	for _, item := range items {
		ua, _ := item.Metadata["user-agent"].(string)
		mail, _ := item.Metadata["mail"].(string)

		err = w.Write([]string{
			item.RoomID.String(),
			item.RoomName,
			item.ParticipantID.String(),
			item.DisplayName,
			item.Time.String(),
			string(item.Message),
			item.Speech,
			ua,
			mail,
		})
		if err != nil {
			slog.Info("error writing csv row", slog.Any("err", err))
			return nil
		}
	}

	w.Flush()
	return b.Bytes()
}
