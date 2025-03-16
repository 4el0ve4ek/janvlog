package listeners

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"janvlog/internal/janus"
	"janvlog/internal/libs/xasync"
	"janvlog/internal/libs/xerrors"
	"janvlog/internal/logs"

	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"github.com/pion/webrtc/v4/pkg/media/oggwriter"
)

func newParticipant(
	roomID logs.RoomID, participantID logs.ParticipantID, displayName string,
	janusClient *janus.Client,
) (*participant, error) {
	handle, err := janusClient.VideoroomHandle()
	if err != nil {
		return nil, xerrors.Wrap(err, "janus.VideoroomHandle")
	}

	trackRecorder := &trackRecorder{
		room:        roomID.String(),
		displayName: displayName,
		filename:    "",
		mu:          sync.RWMutex{},
	}

	peerConnection, err := startPeerConnection(handle, roomID, participantID, trackRecorder)
	if err != nil {
		return nil, xerrors.Wrap(err, "startPeerConnection")
	}

	ret := &participant{
		handle:        handle,
		pc:            peerConnection,
		closer:        xasync.NewCloser(),
		trackRecorder: trackRecorder,
	}

	go ret.watchHandle()

	return ret, nil
}

type participant struct {
	handle        *janus.Handle
	pc            *webrtc.PeerConnection
	closer        xasync.Closer
	trackRecorder *trackRecorder
}

func (l *participant) watchHandle() {
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
			fmt.Println("EventMsg", string(jj))

			if msg.Plugindata.Data["videoroom"].(string) == "updated" {
				fmt.Println("we lost our guy, canceling everything")

				if err := l.Close(); err != nil {
					fmt.Println("error closing participant listener", err)
				}

				return
			}
		}
	}
}

func (l *participant) Close() error {
	if l == nil {
		return nil
	}

	if !l.closer.Close() {
		return nil
	}

	defer func() {
		_, _ = l.handle.Detach()
	}()

	err := l.pc.GracefulClose()
	if err != nil {
		return xerrors.Wrap(err, "graceful close peer connection")
	}

	_, err = l.handle.Message(map[string]interface{}{
		"request": "leave",
	}, nil)
	if err != nil {
		return xerrors.Wrap(err, "handle janus leave")
	}

	return nil
}

func (l *participant) GetAudioFileName() string {
	if l == nil {
		return ""
	}

	return l.trackRecorder.getName()
}

func startPeerConnection(handle *janus.Handle, roomID logs.RoomID, participantID logs.ParticipantID, trackRecorder *trackRecorder) (*webrtc.PeerConnection, error) {
	msg, err := handle.Message(map[string]interface{}{
		"request": "join",
		"ptype":   "subscriber",
		"room":    roomID,
		"streams": []any{
			map[string]any{
				"feed": participantID,
			},
		},
	}, nil)
	if err != nil {
		return nil, xerrors.Wrap(err, "handle janus join")
	}

	offer, err := createOffer(msg)
	if err != nil {
		return nil, xerrors.Wrap(err, "create offer")
	}

	peerConnection, err := createPeerConnection(trackRecorder)
	if err != nil {
		return nil, xerrors.Wrap(err, "create peer connection")
	}

	answer, err := processOffer(peerConnection, offer)
	if err != nil {
		return nil, xerrors.Wrap(err, "process offer")
	}

	// now we start
	_, err = handle.Message(map[string]interface{}{
		"request": "start",
		"room":    roomID,
	}, map[string]interface{}{
		"type": "answer",
		"sdp":  answer.SDP,
	})
	if err != nil {
		return nil, xerrors.Wrap(err, "hanlde janus start")
	}

	return peerConnection, nil
}

func createOffer(msg *janus.EventMsg) (webrtc.SessionDescription, error) {
	if msg.Jsep == nil {
		return webrtc.SessionDescription{}, errors.New("not found jsep")
	}

	sdpVal, ok := msg.Jsep["sdp"].(string)
	if !ok {
		return webrtc.SessionDescription{}, errors.New("failed to cast sdp")
	}

	return webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdpVal,
	}, nil
}

func createPeerConnection(trackRecorder *trackRecorder) (*webrtc.PeerConnection, error) {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
	})
	if err != nil {
		return nil, xerrors.Wrap(err, "new peer connection")
	}

	trancieverInit := webrtc.RTPTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionRecvonly,
	}
	// We must offer to send media for Janus to send anything
	if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, trancieverInit); err != nil {
		return nil, xerrors.Wrap(err, "add transceiver from kind")
	}

	if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, trancieverInit); err != nil {
		return nil, xerrors.Wrap(err, "add transceiver from kind")
	}

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
	})

	peerConnection.OnTrack(trackRecorder.Record())

	return peerConnection, nil
}

func processOffer(peerConnection *webrtc.PeerConnection, offer webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		return nil, xerrors.Wrap(err, "set remote description")
	}

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	answer, answerErr := peerConnection.CreateAnswer(nil)
	if answerErr != nil {
		return nil, xerrors.Wrap(answerErr, "create answer")
	}

	if err := peerConnection.SetLocalDescription(answer); err != nil {
		return nil, xerrors.Wrap(err, "set local description")
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	<-gatherComplete

	return peerConnection.LocalDescription(), nil
}

type trackRecorder struct {
	room        string
	displayName string
	filename    string
	mu          sync.RWMutex
}

func (r *trackRecorder) setName(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.filename = name
}

func (r *trackRecorder) getName() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.filename
}

func (r *trackRecorder) Record() func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
	return func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		codec := track.Codec()
		name := fmt.Sprintf("logs/audio/%s/%s/%s", r.room, r.displayName, strconv.FormatInt(time.Now().Unix(), 10))
		suffix := ""

		err := os.MkdirAll(filepath.Dir(name), 0777)
		if err != nil {
			slog.Error("os.MkdirAll", slog.Any("err", err))
		}

		defer func() {
			if suffix == "" {
				return
			}

			oldName := name + suffix
			newName := name + "_" + strconv.FormatInt(time.Now().Unix(), 10) + suffix

			if err := os.Rename(oldName, newName); err != nil {
				slog.Error("os.Rename", slog.Any("err", err))
			}

			r.setName(newName)
		}()

		if codec.MimeType == "audio/opus" {
			suffix = ".ogg"

			slog.Info("Got Opus track, saving to disk as " + name + suffix)

			i, oggNewErr := oggwriter.New(name+suffix, codec.ClockRate, codec.Channels)
			if oggNewErr != nil {
				panic(oggNewErr)
			}

			saveToDisk(i, track)
		} else if track.Kind() == webrtc.RTPCodecTypeAudio {
			slog.Info("Got audio track but not opus " + codec.MimeType)
		}
	}
}

func saveToDisk(writeFile media.Writer, track *webrtc.TrackRemote) {
	defer func() {
		if err := writeFile.Close(); err != nil {
			panic(err)
		}

		fmt.Println("stopped writing ", track.ID())
	}()

	for {
		packet, _, err := track.ReadRTP()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}

			panic(err)
		}

		if err := writeFile.WriteRTP(packet); err != nil {
			panic(err)
		}
	}
}
