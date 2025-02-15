package listeners

import (
	"errors"
	"fmt"
	"janvlog/internal/janus"
	"janvlog/internal/libs/generics"
	"janvlog/internal/libs/xasync"
	"janvlog/internal/logs"
	"slices"
	"strconv"
	"sync"
	"time"
)

func NewRoomListener(roomID float64, handle *janus.Handle, jc *janus.Client) *roomListener {
	ret := &roomListener{
		handle:       handle,
		roomID:       roomID,
		participants: make(map[uint64]*participantListener),
		jc:           jc,
		closer:       xasync.NewCloser(),
		wg:           &sync.WaitGroup{},
		lw:           generics.Must(logs.NewWriter("room-" + strconv.Itoa(int(roomID)) + "_" + strconv.Itoa(int(time.Now().Unix())))),
	}

	ret.wg.Add(1)
	go ret.watchParticipants()

	return ret
}

type roomListener struct {
	handle       *janus.Handle
	roomID       float64
	participants map[uint64]*participantListener
	jc           *janus.Client

	closer xasync.Closer
	wg     *sync.WaitGroup
	lw     logs.Writer
}

func (l *roomListener) watchParticipants() {
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

		// fmt.Println(string(PairFirst(json.MarshalIndent(lst.PluginData.Data, "", "\t"))))
		participants := lst.PluginData.Data["participants"].([]any)

		roomMemberIDs := make([]uint64, 0, len(participants))
		for _, participant := range participants {
			participantT := participant.(map[string]interface{})

			pid := uint64(participantT["id"].(float64))
			isActive := participantT["publisher"].(bool)
			displayName := participantT["display"].(string)

			roomMemberIDs = append(roomMemberIDs, pid)

			if !isActive {
				l.processNotActive(pid, displayName)
			} else {
				l.processActive(pid, displayName)
			}
		}

		wasSomeOne := len(l.participants) != 0

		for pid, participant := range l.participants {
			if !slices.Contains(roomMemberIDs, pid) {
				participant.Close()
				delete(l.participants, pid)

				l.lw.Write(logs.LogItem{
					RoomID:        l.roomID,
					ParticipantID: pid,
					DisplayName:   "",
					Message:       logs.MessageLeft,
					AudioFile:     participant.GetAudioFileName(),
				})
			}
		}

		if wasSomeOne && len(l.participants) == 0 {
			fmt.Println("in room ", l.roomID, "no participants")
			l.generateReport()
		}
	}
}

func (l *roomListener) Close() error {
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

func (l *roomListener) processActive(pid uint64, displayName string) {
	pl, ok := l.participants[pid]
	if ok && pl != nil {
		return
	}

	fmt.Println("in room ", l.roomID, "new participant: ", pid)

	pl, err := NewParticipantListener(
		l.roomID, pid, displayName,
		l.jc,
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	l.participants[pid] = pl

	msg := logs.MessageJoined
	if ok {
		msg = logs.MessageEnableCamera
	}

	l.lw.Write(logs.LogItem{
		RoomID:        l.roomID,
		ParticipantID: pid,
		DisplayName:   displayName,
		Message:       msg,
	})
}

func (l *roomListener) processNotActive(pid uint64, displayName string) {
	pl, ok := l.participants[pid]
	if !ok {
		l.participants[pid] = nil

		l.lw.Write(logs.LogItem{
			RoomID:        l.roomID,
			ParticipantID: pid,
			DisplayName:   displayName,
			Message:       logs.MessageJoinedWithoutCam,
		})

		return
	}

	if pl == nil {
		return
	}

	err := pl.Close()
	if err != nil {
		fmt.Println(err)
	}
	l.participants[pid] = nil

	l.lw.Write(logs.LogItem{
		RoomID:        l.roomID,
		ParticipantID: pid,
		DisplayName:   displayName,
		Message:       logs.MessageDisableCamera,
		AudioFile:     pl.tr.getName(),
	})
}

func (l *roomListener) generateReport() {
	l.lw.Write(logs.LogItem{
		RoomID:  l.roomID,
		Message: logs.MessageEmptyRoom,
	})
	l.lw.Close()

	l.lw = generics.Must(logs.NewWriter("room-" + strconv.Itoa(int(l.roomID)) + "_" + strconv.Itoa(int(time.Now().Unix()))))
}
