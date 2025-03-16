package listeners

import (
	"errors"
	"log"
	"slices"
	"strconv"
	"sync"
	"time"

	"janvlog/internal/janus"
	"janvlog/internal/libs/generics"
	"janvlog/internal/libs/xasync"
	"janvlog/internal/logs"
	"janvlog/internal/reporter"
)

func newRoom(
	roomID logs.RoomID,
	roomName string,
	handle *janus.Handle,
	janusClient *janus.Client,
	reporter *reporter.Generator,
) *room {
	ret := &room{
		reporter:     reporter,
		handle:       handle,
		roomID:       roomID,
		roomName:     roomName,
		participants: make(map[logs.ParticipantID]*participant),
		janusClient:  janusClient,
		closer:       xasync.NewCloser(),
		wg:           &sync.WaitGroup{},
		lw:           generics.Must(logs.NewStorage("logs/raw/room-" + strconv.Itoa(int(roomID)) + "/" + strconv.Itoa(int(time.Now().Unix())))),
	}

	ret.wg.Add(1)
	go ret.watchParticipants()

	return ret
}

type room struct {
	handle       *janus.Handle
	roomID       logs.RoomID
	roomName     string
	participants map[logs.ParticipantID]*participant
	janusClient  *janus.Client

	closer   xasync.Closer
	wg       *sync.WaitGroup
	lw       logs.Storage
	reporter *reporter.Generator
}

func (l *room) watchParticipants() {
	defer l.wg.Done()

	for {
		if l.closer.Wait(time.Second) {
			return
		}

		lst, err := l.handle.Request(map[string]interface{}{
			"request":   "listparticipants",
			"room":      l.roomID,
			"admin_key": "janusoverlord",
		})
		if err != nil {
			panic(err)
		}

		// log.Println(string(PairFirst(json.MarshalIndent(lst.PluginData.Data, "", "\t"))))
		participants, _ := lst.PluginData.Data["participants"].([]any)
		roomMemberIDs := make([]logs.ParticipantID, 0, len(participants))

		for _, participant := range participants {
			participantT := participant.(map[string]interface{})

			pid := logs.ParticipantID(participantT["id"].(float64))
			isActive := participantT["publisher"].(bool)
			displayName := participantT["display"].(string)
			metadata, _ := participantT["metadata"].(map[string]string)

			userData := logs.UserData{
				DisplayName: displayName,
				Metadata:    metadata,
			}

			roomMemberIDs = append(roomMemberIDs, pid)

			if !isActive {
				l.processNotActive(pid, userData)
			} else {
				l.processActive(pid, userData)
			}
		}

		wasSomeOne := len(l.participants) != 0

		for pid, participant := range l.participants {
			if !slices.Contains(roomMemberIDs, pid) {
				participant.Close()
				delete(l.participants, pid)

				l.lw.Add(logs.Item{
					RoomID:        l.roomID,
					RoomName:      l.roomName,
					ParticipantID: pid,
					DisplayName:   "",
					Message:       logs.MessageLeft,
					AudioFile:     participant.GetAudioFileName(),
					Metadata:      nil,
				})
			}
		}

		if wasSomeOne && len(l.participants) == 0 {
			log.Println("in room ", l.roomID, "no participants")
			l.generateReport()
		}
	}
}

func (l *room) Close() error {
	if l == nil {
		return nil
	}

	l.closer.Close()
	l.wg.Wait()

	errs := make([]error, 0, len(l.participants))
	for _, participant := range l.participants {
		errs = append(errs, participant.Close())
	}

	return errors.Join(errs...)
}

func (l *room) processActive(pid logs.ParticipantID, userData logs.UserData) {
	participant, wasConnected := l.participants[pid]
	if wasConnected && participant != nil {
		return
	}

	log.Println("in room ", l.roomID, "new participant: ", pid)

	participant, err := newParticipant(
		l.roomID, pid, userData.DisplayName,
		l.janusClient,
	)
	if err != nil {
		log.Println(err)
		return
	}

	l.participants[pid] = participant

	msg := logs.MessageJoined
	if wasConnected {
		msg = logs.MessageEnableCamera
	}

	l.lw.Add(logs.Item{
		RoomID:        l.roomID,
		RoomName:      l.roomName,
		ParticipantID: pid,
		Message:       msg,
		DisplayName:   userData.DisplayName,
		Metadata:      userData.Metadata,
	})
}

func (l *room) processNotActive(pid logs.ParticipantID, userData logs.UserData) {
	participant, ok := l.participants[pid]
	if !ok {
		l.participants[pid] = nil

		l.lw.Add(logs.Item{
			RoomID:        l.roomID,
			RoomName:      l.roomName,
			ParticipantID: pid,
			Message:       logs.MessageJoinedWithoutCam,
			DisplayName:   userData.DisplayName,
			Metadata:      userData.Metadata,
		})

		return
	}

	if participant == nil {
		return
	}

	err := participant.Close()
	if err != nil {
		log.Println(err)
	}

	l.participants[pid] = nil

	l.lw.Add(logs.Item{
		RoomID:        l.roomID,
		RoomName:      l.roomName,
		ParticipantID: pid,
		Message:       logs.MessageDisableCamera,
		AudioFile:     participant.trackRecorder.getName(),
		DisplayName:   userData.DisplayName,
		Metadata:      userData.Metadata,
	})
}

func (l *room) generateReport() {
	l.lw.Add(logs.Item{
		RoomID:   l.roomID,
		RoomName: l.roomName,
		Message:  logs.MessageEmptyRoom,
	})

	l.lw.Close()

	l.reporter.StartProcessing(l.lw.File())

	l.lw = generics.Must(logs.NewStorage("logs/raw/room-" + strconv.Itoa(int(l.roomID)) + "/" + strconv.Itoa(int(time.Now().Unix()))))
}
