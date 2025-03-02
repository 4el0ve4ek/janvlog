package janus

import (
	"errors"
	"fmt"
	"janvlog/internal/libs/generics"
	"janvlog/internal/libs/xerrors"

	janus "github.com/notedit/janus-go"
)

type Handle = janus.Handle
type EventMsg = janus.EventMsg

func New(wshost string) (*Client, error) {
	gateway, err := janus.Connect("ws://" + wshost + ":8188/")
	if err != nil {
		return nil, xerrors.Wrap(err, "janus.Connect")
	}

	// Create session
	session, err := gateway.Create()
	if err != nil {
		return nil, xerrors.Wrap(err, "gateway.Create")
	}

	return &Client{
		gateway: gateway,
		session: session,
	}, nil
}

type Client struct {
	gateway *janus.Gateway
	session *janus.Session
}

func (jc *Client) VideoroomHandle() (*janus.Handle, error) {
	return jc.session.Attach("janus.plugin.videoroom") //nolint:wrapcheck
}

func (jc *Client) Close() error {
	return errors.Join(
		generics.Second(jc.session.Destroy()),
		jc.gateway.Close(),
	)
}

func (jc *Client) KeepAlive() (*janus.AckMsg, error) {
	return jc.session.KeepAlive() //nolint:wrapcheck
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
