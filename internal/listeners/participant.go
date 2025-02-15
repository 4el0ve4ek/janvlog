package listeners

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"janvlog/internal/janus"
	"janvlog/internal/libs/xasync"

	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"github.com/pion/webrtc/v4/pkg/media/oggwriter"
)

func NewParticipantListener(
	roomID float64, participantID uint64, displayName string,
	janusClient *janus.Client,
) (*participantListener, error) {
	handle, err := janusClient.VideoroomHandle()
	if err != nil {
		return nil, err
	}

	tr := &trackRecorder{prefix: displayName}
	pc, err := startPeerConnection(handle, roomID, participantID, tr)
	if err != nil {
		return nil, err
	}

	ret := &participantListener{
		roomID:        roomID,
		participantID: participantID,
		displayName:   displayName,
		handle:        handle,
		pc:            pc,
		closer:        xasync.NewCloser(),
		tr:            tr,
	}

	go ret.watchHandle()

	return ret, nil
}

type participantListener struct {
	roomID        float64
	participantID uint64
	displayName   string
	handle        *janus.Handle
	pc            *webrtc.PeerConnection
	closer        xasync.Closer
	fname         string
	tr            *trackRecorder
}

func (l *participantListener) watchHandle() {
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

func (l *participantListener) Close() error {
	if l == nil {
		return nil
	}

	if !l.closer.Close() {
		return nil
	}

	defer l.handle.Detach()

	err := l.pc.GracefulClose()
	if err != nil {
		return err
	}

	_, err = l.handle.Message(map[string]interface{}{
		"request": "leave",
	}, nil)
	if err != nil {
		return err
	}

	return nil
}

func (l *participantListener) GetAudioFileName() string {
	if l == nil {
		return ""
	}

	return l.tr.getName()
}

func startPeerConnection(handle *janus.Handle, roomID float64, participantID uint64, trackRecorder *trackRecorder) (*webrtc.PeerConnection, error) {
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
		return nil, err
	}

	offer, err := createOffer(msg)
	if err != nil {
		return nil, err
	}

	peerConnection, err := createPeerConnection(trackRecorder)
	if err != nil {
		return nil, err
	}

	answer, err := processOffer(peerConnection, offer)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return peerConnection, nil
}

func createOffer(msg *janus.EventMsg) (webrtc.SessionDescription, error) {
	if msg.Jsep == nil {
		return webrtc.SessionDescription{}, fmt.Errorf("not found jsep")
	}

	sdpVal, ok := msg.Jsep["sdp"].(string)
	if !ok {
		return webrtc.SessionDescription{}, fmt.Errorf("failed to cast sdp")
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
		return nil, err
	}

	trancieverInit := webrtc.RTPTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionRecvonly,
	}
	// We must offer to send media for Janus to send anything
	if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, trancieverInit); err != nil {
		return nil, err
	}

	if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, trancieverInit); err != nil {
		return nil, err
	}

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
	})

	peerConnection.OnTrack(trackRecorder.Record())

	return peerConnection, nil
}

func processOffer(peerConnection *webrtc.PeerConnection, offer webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		return nil, err
	}

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	answer, answerErr := peerConnection.CreateAnswer(nil)
	if answerErr != nil {
		return nil, answerErr
	}

	if err := peerConnection.SetLocalDescription(answer); err != nil {
		return nil, err
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	<-gatherComplete

	return peerConnection.LocalDescription(), nil
}

type trackRecorder struct {
	prefix   string
	filename string
	mu       sync.RWMutex
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
		name := "output_" + r.prefix + "_" + strconv.FormatInt(time.Now().Unix(), 10)
		suffix := ""

		defer func() {
			if suffix == "" {
				return
			}

			oldName := name + suffix
			newName := name + "_" + strconv.FormatInt(time.Now().Unix(), 10) + suffix

			if err := os.Rename(oldName, newName); err != nil {
				fmt.Println(err)
			}

			r.setName(newName)
		}()

		if codec.MimeType == "audio/opus" {
			suffix = ".ogg"

			fmt.Println("Got Opus track, saving to disk as " + name + suffix)
			i, oggNewErr := oggwriter.New(name+suffix, codec.ClockRate, codec.Channels)
			if oggNewErr != nil {
				panic(oggNewErr)
			}

			saveToDisk(i, track)
		} else if track.Kind() == webrtc.RTPCodecTypeAudio {
			fmt.Println("Got audio track but not opus", codec.MimeType)
		}
	}
}

func saveToDisk(i media.Writer, track *webrtc.TrackRemote) {
	defer func() {
		if err := i.Close(); err != nil {
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

		if err := i.WriteRTP(packet); err != nil {
			panic(err)
		}
	}
}
