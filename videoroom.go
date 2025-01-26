package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	janus "github.com/notedit/janus-go"
)

func NewVideoroomListener(jc *jc) (*videoroomListener, error) {
	handle, err := jc.VideoroomHandle()
	if err != nil {
		return nil, err
	}

	ret := &videoroomListener{
		handle: handle,
		rooms:  make(map[float64]*roomListener),
		jc:     jc,

		closer: NewCloser(),
		wg:     &sync.WaitGroup{},
	}

	ret.wg.Add(2)
	go ret.watchHandle()
	go ret.watchRooms()

	return ret, nil
}

type videoroomListener struct {
	handle *janus.Handle
	rooms  map[float64]*roomListener
	jc     *jc

	closer *closer
	wg     *sync.WaitGroup
}

func (l *videoroomListener) watchRooms() {
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
				l.rooms[roomID] = NewRoomListener(roomID, l.handle, l.jc)
			}
		}
	}
}

func (l *videoroomListener) watchHandle() {
	defer l.wg.Done()

	for {
		select {
		case <-l.closer.Closed():
			return
		case event := <-l.handle.Events:
			msg := ProcessEvent(l.handle.ID, event)
			if msg == nil {
				continue
			}
			jj, _ := json.MarshalIndent(msg, "", "\t")
			fmt.Printf("EventMsg %+v", msg.Plugindata.Data)
			fmt.Println(string(jj))
		}
	}
}

func (l *videoroomListener) Close() error {
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
