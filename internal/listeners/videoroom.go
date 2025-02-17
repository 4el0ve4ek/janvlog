package listeners

import (
	"encoding/json"
	"errors"
	"fmt"
	"janvlog/internal/janus"
	"janvlog/internal/libs/xasync"
	"janvlog/internal/reporter"
	"sync"
	"time"
)

func NewVideoroom(jc *janus.Client, reporter *reporter.Generator) (*videoroom, error) {
	handle, err := jc.VideoroomHandle()
	if err != nil {
		return nil, err
	}

	ret := &videoroom{
		handle:   handle,
		rooms:    make(map[float64]*room),
		jc:       jc,
		reporter: reporter,

		closer: xasync.NewCloser(),
		wg:     &sync.WaitGroup{},
	}

	ret.wg.Add(2)
	go ret.watchHandle()
	go ret.watchRooms()

	return ret, nil
}

type videoroom struct {
	handle   *janus.Handle
	rooms    map[float64]*room
	jc       *janus.Client
	reporter *reporter.Generator

	closer xasync.Closer
	wg     *sync.WaitGroup
}

func (l *videoroom) watchRooms() {
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
			roomID := room.(map[string]interface{})["room"].(float64)

			_, ok := l.rooms[roomID]
			if !ok {
				fmt.Println("New room", roomID)
				l.rooms[roomID] = NewRoom(roomID, l.handle, l.jc, l.reporter)
			}
		}
	}
}

func (l *videoroom) watchHandle() {
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

func (l *videoroom) Close() error {
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
