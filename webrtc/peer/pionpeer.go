package peer

import (
	"errors"
	"fmt"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/p2p"
	webrtc2 "github.com/curltech/go-colla-node/webrtc"
	"github.com/curltech/go-colla-node/webrtc/peer/simplepeer"
	"github.com/pion/webrtc/v3"
	"sync"
)

type PionPeer struct {
	*p2p.NetPeer
	/**
	是offer还是answer
	*/
	initiator         bool
	peerConnection    *webrtc.PeerConnection
	iceServer         []webrtc.ICEServer
	pendingCandidates []webrtc.ICECandidateInit
	candidatesMux     sync.Mutex
	dataChannel       *webrtc.DataChannel
	connected         bool
	events            map[string]func(event *PoolEvent) (interface{}, error)
}

func (this *PionPeer) RegistEvent(name string, fn func(event *PoolEvent) (interface{}, error)) {
	if this.events == nil {
		this.events = make(map[string]func(event *PoolEvent) (interface{}, error), 0)
	}
	this.events[name] = fn
}

func (this *PionPeer) UnregistEvent(name string) bool {
	if this.events == nil {
		return false
	}
	delete(this.events, name)
	return true
}

func (this *PionPeer) EmitEvent(name string, event *PoolEvent) (interface{}, error) {
	if this.events == nil {
		return nil, errors.New("EventNotExist")
	}
	fn, ok := this.events[name]
	if ok {
		return fn(event)
	}
	return nil, errors.New("EventNotExist")
}

/**
获取本节点的webrtc连接状态
*/
func (this *PionPeer) Connected() bool {
	return this.connected
}

func (this *PionPeer) Id() string {
	return this.TargetPeerId + ":" + this.ConnectPeerId + ":" + this.ConnectSessionId
}

/**
发送candidate signal
*/
func (this *PionPeer) signalCandidate(candidate *webrtc.ICECandidateInit) error {
	// payload := []byte(candidate.ToJSON().Candidate)
	// can := candidate.ToJSON()
	webrtcSignal := &simplepeer.WebrtcSignal{SignalType: simplepeer.WebrtcSignalType_Candidate, Candidate: candidate}
	_, err := simplepeer.Signal(webrtcSignal, this.TargetPeerId)
	if err != nil {
		logger.Sugar.Errorf("%v", err)
		return err
	}

	return nil
}

func (this *PionPeer) Create(targetPeerId string, initiator bool, options *webrtc2.WebrtcOption, iceServer []webrtc.ICEServer) {
	this.TargetPeerId = targetPeerId
	this.pendingCandidates = make([]webrtc.ICECandidateInit, 0)
	if iceServer == nil {
		iceServer = webrtc2.GetICEServers()
	}
	this.iceServer = iceServer
	if options == nil {
		options = &webrtc2.WebrtcOption{}
		options.Config = &webrtc.Configuration{
			ICEServers: iceServer,
		}
	}
	if initiator {
		options.Initiator = initiator
	}
	var err error
	this.peerConnection, err = webrtc.NewPeerConnection(*options.Config)
	if err != nil {
		logger.Sugar.Errorf("%v", err)
	}

	// When an ICE candidate is available send to the other Pion instance
	// the other Pion instance will add this candidate by calling AddICECandidate
	this.peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		this.candidatesMux.Lock()
		defer this.candidatesMux.Unlock()
		can := candidate.ToJSON()
		desc := this.peerConnection.RemoteDescription()
		if desc == nil {
			this.pendingCandidates = append(this.pendingCandidates, can)
		} else if onICECandidateErr := this.signalCandidate(&can); err != nil {
			logger.Sugar.Errorf("%v", onICECandidateErr)
		}
	})

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	this.peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	this.setDataChannel()
	if this.initiator {
		this.createOffer()
	}
}

/**
设置本节点的信号
*/
func (this *PionPeer) Signal(webrtcSignal *simplepeer.WebrtcSignal) {
	// A SignalAction receive handler that allows the other webrtc instance to send us ICE candidates
	// This allows us to add ICE candidates faster, we don't have to wait for STUN or TURN
	// candidates which may be slower
	candidate := webrtcSignal.Candidate
	if candidate != nil {
		candidateErr := this.peerConnection.AddICECandidate(*candidate)
		if candidateErr != nil {
			logger.Sugar.Errorf("%v", candidateErr)
			return
		}
	} else {
		// A SignalAction receive handler that processes a SessionDescription given to us from the other Pion process
		sdp := webrtcSignal.Sdp
		if sdp != nil {
			sdpErr := this.peerConnection.SetRemoteDescription(*sdp)
			if sdpErr != nil {
				logger.Sugar.Errorf("%v", sdpErr)
				return
			}
			if !this.initiator {
				this.createAnswer()
			}
			this.candidatesMux.Lock()
			defer this.candidatesMux.Unlock()

			for _, pendingCandidate := range this.pendingCandidates {
				onICECandidateErr := this.signalCandidate(&pendingCandidate)
				if onICECandidateErr != nil {
					logger.Sugar.Errorf("%v", onICECandidateErr)
					return
				}
			}
		}
	}

}

func (this *PionPeer) setDataChannel() {
	// Create a datachannel with label 'data'
	var err error
	if this.initiator {
		this.dataChannel, err = this.peerConnection.CreateDataChannel("data", nil)
		if err != nil {
			logger.Sugar.Errorf("%v", err)
			return
		}
		this.registDataChannelEvent()
	} else {
		this.peerConnection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
			fmt.Printf("New DataChannel %s %d\n", dataChannel.Label(), dataChannel.ID())
			this.dataChannel = dataChannel
			this.registDataChannelEvent()
		})
	}
}

func (this *PionPeer) registDataChannelEvent() {
	// Register channel opening handling
	this.dataChannel.OnOpen(func() {
		fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", this.dataChannel.Label(), this.dataChannel.ID())
		// Send the message as text
		sendTextErr := this.SendText("Hello,胡劲松")
		if sendTextErr != nil {
			logger.Sugar.Errorf("%v", sendTextErr)
			return
		}
		//for range time.NewTicker(5 * time.Second).C {
		//	message := security.UUID()
		//	fmt.Printf("Sending '%s'\n", message)
		//}
	})

	// Register text message handling
	this.dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("Message from DataChannel '%s': '%s'\n", this.dataChannel.Label(), string(msg.Data))
	})

	this.dataChannel.OnClose(func() {
		fmt.Printf("DataChannel close")
	})
	this.dataChannel.OnError(func(err error) {
		fmt.Printf("DataChannel error%v", err.Error())
	})
}

func (this *PionPeer) SendText(message string) error {
	return this.dataChannel.SendText(message)
}

func (this *PionPeer) Send(data []byte) error {
	return this.dataChannel.Send(data)
}

func (this *PionPeer) createOffer() {
	// Create an offer to send to the other process
	offer, err := this.peerConnection.CreateOffer(&webrtc.OfferOptions{ICERestart: true})
	if err != nil {
		logger.Sugar.Errorf("%v", err)
		return
	}

	// Sets the LocalDescription, and starts our UDP listeners
	// Note: this will start the gathering of ICE candidates
	if err = this.peerConnection.SetLocalDescription(offer); err != nil {
		logger.Sugar.Errorf("%v", err)
		return
	}

	// Send our offer to the server listening in the other process
	webrtcSignal := &simplepeer.WebrtcSignal{SignalType: simplepeer.WebrtcSignalType_Offer, Sdp: &offer}
	_, err = simplepeer.Signal(webrtcSignal, this.TargetPeerId)
	if err != nil {
		logger.Sugar.Errorf("%v", err)
		return
	}
}

/**
创建answer，在接收到sdp时调用
*/
func (this *PionPeer) createAnswer() {
	// Create an answer to send to the other process
	answer, err := this.peerConnection.CreateAnswer(nil)
	webrtcSignal := &simplepeer.WebrtcSignal{SignalType: simplepeer.WebrtcSignalType_Answer, Sdp: &answer}
	_, err = simplepeer.Signal(webrtcSignal, this.TargetPeerId)
	if err != nil {
		logger.Sugar.Errorf("%v", err)
		return
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = this.peerConnection.SetLocalDescription(answer)
	if err != nil {
		logger.Sugar.Errorf("%v", err)
		return
	}
}

func (this *PionPeer) Destroy(err error) {
	if this.dataChannel != nil {
		this.dataChannel.Close()
	}
	if this.peerConnection != nil {
		this.peerConnection.Close()
	}
	webrtcPeerPool.remove(this.NetPeer)
}
