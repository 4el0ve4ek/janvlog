package main

import (
	"errors"
	"fmt"

	janus "github.com/notedit/janus-go"
)

func NewJanusClient(wshost string) *jc {
	// Connect to gateway
	gateway, err := janus.Connect("ws://" + wshost + ":8188/")
	if err != nil {
		panic(err)
	}

	// Create session
	session, err := gateway.Create()
	if err != nil {
		panic(err)
	}

	return &jc{
		gateway: gateway,
		session: session,
	}
}

type jc struct {
	gateway *janus.Gateway
	session *janus.Session
}

func (jc *jc) VideoroomHandle() (*janus.Handle, error) {
	return jc.session.Attach("janus.plugin.videoroom")
}

func (jc *jc) Close() error {
	return errors.Join(
		PairSecond(jc.session.Destroy()),
		jc.gateway.Close(),
	)
}

func ProcessEvent(handleID uint64, event any) *janus.EventMsg {
	switch msg := event.(type) {
	case *janus.SlowLinkMsg:
		fmt.Println("SlowLinkMsg type ", handleID)
	case *janus.MediaMsg:
		fmt.Println("MediaEvent type", msg.Type, " receiving ", msg.Receiving)
	case *janus.WebRTCUpMsg:
		fmt.Println("WebRTCUp type ", handleID)
	case *janus.HangupMsg:
		fmt.Println("HangupEvent type ", handleID)
	case *janus.EventMsg:
		return msg
	}

	return nil
}
