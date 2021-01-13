package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	sfu "github.com/curltech/go-colla-node/webrtc/ion/sfu"
	log "github.com/pion/ion-log"
	sdk "github.com/pion/ion-sdk-go"
	"github.com/pion/webrtc/v3"
)

type SfuClient struct {
	*sdk.WebRTCTransport
	Sid       string
	Connected bool
}

func (s *SfuClient) Join(peerId string, sid string) {
	log.Infof("Joining sfu session: %s", sid)

	offer, err := s.CreateOffer()
	if err != nil {
		log.Errorf("Error creating offer: %v", err)
		return
	}

	log.Debugf("Send offer:\n %s", offer.SDP)

	//自定义的发送join信号
	sfuSignal := &sfu.SfuSignal{SignalType: "join", Sid: sid, Sdp: &offer}
	_, err = sfu.Signal(sfuSignal, peerId)

	if err != nil {
		log.Errorf("Error sending publish request: %v", err)
		return
	}

	s.OnICECandidate(func(c *webrtc.ICECandidate, target int) {
		if c == nil {
			// Gathering done
			return
		}
		candidate := c.ToJSON()
		sfuSignal := &sfu.SfuSignal{SignalType: "trickle", Candidate: &candidate, Target: target}
		_, err := sfu.Signal(sfuSignal, "")

		if err != nil {
			log.Errorf("OnIceCandidate error %s", err)
		}
	})
}

type SfuClientPool struct {
	ctx        context.Context
	cancel     context.CancelFunc
	config     sdk.Config
	lock       sync.RWMutex
	onCloseFn  func()
	sfuClients map[string]*SfuClient
}

func NewSfuClientPool(config sdk.Config) (*SfuClientPool, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &SfuClientPool{
		ctx:        ctx,
		cancel:     cancel,
		config:     config,
		sfuClients: make(map[string]*SfuClient),
	}, nil
}

// GetTransport returns a webrtc transport for a session
func (s *SfuClientPool) Get(peerId string, sid string) *SfuClient {
	s.lock.Lock()
	defer s.lock.Unlock()
	key := sid + ":" + peerId
	sfuClient := s.sfuClients[key]

	return sfuClient
}

func (s *SfuClientPool) Create(peerId string, sid string) (*SfuClient, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	key := sid + ":" + peerId
	sfuClient := s.sfuClients[key]
	if sfuClient != nil {
		if !sfuClient.Connected {
			sfuClient.Close()
			delete(s.sfuClients, peerId)
		}

		return sfuClient, nil
	}
	transport := sdk.NewWebRTCTransport(peerId, s.config)
	if transport == nil {
		return nil, errors.New("CreateFail")
	}
	sfuClient = &SfuClient{WebRTCTransport: transport}
	sfuClient.OnClose(func() {
		s.lock.Lock()
		defer s.lock.Unlock()
		sfuClient.Connected = false
		delete(s.sfuClients, peerId)
		if len(s.sfuClients) == 0 && s.onCloseFn != nil {
			s.cancel()
			s.onCloseFn()
		}
	})
	s.sfuClients[key] = sfuClient

	return sfuClient, nil
}

func (this *SfuClientPool) Receive(peerId string, sid string, payload map[string]interface{}) (interface{}, error) {
	sfuSignal := sfu.Transform(payload)
	if sfuSignal == nil {
		return nil, errors.New("NotSfuSignal")
	}
	sfuClient := this.Get(peerId, sid)
	if sfuClient == nil {
		return nil, errors.New("SfuClient")
	}

	switch sfuSignal.SignalType {
	case "join":
		return this.join(sfuClient, sfuSignal)
	case "offer":
		return this.offer(sfuClient, sfuSignal)
	case "answer":
		return this.answer(sfuClient, sfuSignal)
	case "trickle":
		return this.trickle(sfuClient, sfuSignal)
	}

	return nil, errors.New("SignalTypeError")
}

func (s *SfuClientPool) join(sfuClient *SfuClient, sfuSignal *sfu.SfuSignal) (interface{}, error) {
	// Set the remote SessionDescription
	log.Debugf("got answer: %s", sfuSignal.Sdp)

	var sdp webrtc.SessionDescription = *sfuSignal.Sdp
	if err := sfuClient.SetRemoteDescription(sdp); err != nil {
		log.Errorf("join error %s", err)
		return nil, err
	}

	return nil, nil
}

func (s *SfuClientPool) offer(sfuClient *SfuClient, sfuSignal *sfu.SfuSignal) (interface{}, error) {
	var sdp webrtc.SessionDescription = *sfuSignal.Sdp
	if sdp.Type == webrtc.SDPTypeOffer {
		log.Debugf("got offer: %v", sdp)

		var answer webrtc.SessionDescription
		answer, err := sfuClient.Answer(sdp)
		if err != nil {
			log.Errorf("negotiate error %s", err)
		}

		sfuSignal := &sfu.SfuSignal{SignalType: "answer", Sdp: &answer}
		_, err = sfu.Signal(sfuSignal, "")
	}

	return nil, nil
}

func (s *SfuClientPool) answer(sfuClient *SfuClient, sfuSignal *sfu.SfuSignal) (interface{}, error) {
	var sdp webrtc.SessionDescription = *sfuSignal.Sdp
	log.Debugf("got answer: %v", sdp)
	err := sfuClient.SetRemoteDescription(sdp)

	if err != nil {
		log.Errorf("negotiate error %s", err)
	}

	return nil, nil
}

func (s *SfuClientPool) trickle(sfuClient *SfuClient, sfuSignal *sfu.SfuSignal) (interface{}, error) {
	var candidate webrtc.ICECandidateInit = *sfuSignal.Candidate
	err := sfuClient.AddICECandidate(candidate, sfuSignal.Target)
	if err != nil {
		log.Errorf("error adding ice candidate: %e", err)
	}

	return nil, nil
}

// Stats show all sfu client stats
func (s *SfuClientPool) Stats(cycle int) string {
	for {
		info := "\n-------stats-------\n"

		s.lock.RLock()
		if len(s.sfuClients) == 0 {
			s.lock.RUnlock()
			continue
		}
		info += fmt.Sprintf("Transport: %d\n", len(s.sfuClients))

		totalRecvBW, totalSendBW := 0, 0
		for _, sfuClient := range s.sfuClients {
			recvBW, sendBW := sfuClient.GetBandWidth(cycle)
			totalRecvBW += recvBW
			totalSendBW += sendBW
		}

		info += fmt.Sprintf("RecvBandWidth: %d KB/s\n", totalRecvBW)
		info += fmt.Sprintf("SendBandWidth: %d KB/s\n", totalSendBW)
		s.lock.RUnlock()
		log.Infof(info)
		time.Sleep(time.Duration(cycle) * time.Second)
	}
}
