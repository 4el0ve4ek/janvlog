package listeners

import (
	"encoding/json"
	"errors"
	"fmt"
	"janvlog/internal/janus"
	"janvlog/internal/libs/xasync"
	"janvlog/internal/libs/xerrors"
	"janvlog/internal/logs"
	"janvlog/internal/reporter"
	"sync"
	"time"
)

func NewVideoroom(janusClient *janus.Client, reporter *reporter.Generator) (*Videoroom, error) {
	handle, err := janusClient.VideoroomHandle()
	if err != nil {
		return nil, xerrors.Wrap(err, "janus.VideoroomHandle")
	}

	ret := &Videoroom{
		handle:      handle,
		rooms:       make(map[logs.RoomID]*room),
		janusClient: janusClient,
		reporter:    reporter,

		closer: xasync.NewCloser(),
		wg:     &sync.WaitGroup{},
	}

	ret.wg.Add(1)
	go ret.watchHandle()

	ret.wg.Add(1)
	go ret.watchRooms()

	return ret, nil
}

type Videoroom struct {
	handle      *janus.Handle
	rooms       map[logs.RoomID]*room
	janusClient *janus.Client
	reporter    *reporter.Generator

	closer xasync.Closer
	wg     *sync.WaitGroup
}

func (l *Videoroom) watchRooms() {
	defer l.wg.Done()

	for {
		if l.closer.Wait(time.Second) {
			return
		}

		lst, err := l.handle.Request(map[string]interface{}{
			"request":   "list",
			"admin_key": "janusoverlord",
		})
		if err != nil {
			panic(err)
		}

		rooms := lst.PluginData.Data["list"].([]any)
		for _, room := range rooms {
			roomID := logs.RoomID(room.(map[string]interface{})["room"].(float64))

			_, ok := l.rooms[roomID]
			if !ok {
				fmt.Println("New room", roomID)
				l.rooms[roomID] = newRoom(roomID, l.handle, l.janusClient, l.reporter)
			}
		}
	}
}

func (l *Videoroom) watchHandle() {
	defer l.wg.Done()

	for {
		select {
		case <-l.closer.Closed():
			return
		case event := <-l.handle.Events:
			msg := janus.ProcessEvent(l.handle.ID, event)
			if msg == nil {
				continue
			}

			jj, _ := json.MarshalIndent(msg, "", "\t")
			fmt.Printf("EventMsg %+v", msg.Plugindata.Data)
			fmt.Println(string(jj))
		}
	}
}

func (l *Videoroom) Close() error {
	l.closer.Close()
	l.wg.Wait()

	errs := make([]error, 0, len(l.rooms)+1)

	for _, room := range l.rooms {
		errs = append(errs, room.Close())
	}

	_, err := l.handle.Detach()
	errs = append(errs, err)

	return errors.Join(errs...)
}
