package main

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/notedit/janus-go"
)

func NewRoomListener(roomID float64, handle *janus.Handle, jc *jc) *roomListener {
	ret := &roomListener{
		handle:       handle,
		roomID:       roomID,
		participants: make(map[uint64]*participantListener),
		jc:           jc,
		closer:       NewCloser(),
		wg:           &sync.WaitGroup{},
		lw:           must(NewLogWriter("room-" + strconv.Itoa(int(roomID)))),
	}

	ret.wg.Add(1)
	go ret.watchParticipants()

	return ret
}

type roomListener struct {
	handle       *janus.Handle
	roomID       float64
	participants map[uint64]*participantListener
	jc           *jc

	closer *closer
	wg     *sync.WaitGroup
	lw     *logWriter
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

		for pid, participant := range l.participants {
			if !slices.Contains(roomMemberIDs, pid) {
				participant.Close()
				delete(l.participants, pid)

				l.lw.Write(LogItem{
					RoomID:        l.roomID,
					ParticipantID: pid,
					DisplayName:   "",
					Message:       MessageLeft,
					AudioFile:     participant.tr.getName(),
				})
			}
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

	msg := MessageJoined
	if ok {
		msg = MessageEnableCamera
	}

	l.lw.Write(LogItem{
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

		l.lw.Write(LogItem{
			RoomID:        l.roomID,
			ParticipantID: pid,
			DisplayName:   displayName,
			Message:       MessageJoinedWithoutCam,
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

	l.lw.Write(LogItem{
		RoomID:        l.roomID,
		ParticipantID: pid,
		DisplayName:   displayName,
		Message:       MessageDisableCamera,
		AudioFile:     pl.tr.getName(),
	})
}
