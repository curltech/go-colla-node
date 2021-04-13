package sfu

import (
	"errors"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/p2p"
	"github.com/curltech/go-colla-node/p2p/chain/action/dht"
	log "github.com/pion/ion-log"
	"github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/webrtc/v3"
	"sync"
)

type PoolEvent struct {
	Name   string
	Source *SfuPeer
	Data   interface{}
}

type SfuPeerPool struct {
	*sfu.SFU
	sfuPeers map[string][]*SfuPeer
	lock     sync.RWMutex
	events   map[string]func(event *PoolEvent) (interface{}, error)
}

var sfuPeerPool *SfuPeerPool

func NewSfuPeerPool() *SfuPeerPool {
	sfuPeerPool = &SfuPeerPool{}
	conf := sfu.Config{}
	conf.SFU.Ballast = config.SfuParams.Ballast
	conf.Log = log.Config{Level: config.SfuParams.Level}
	simulcastConfig := sfu.SimulcastConfig{BestQualityFirst: config.SfuParams.Bestqualityfirst, EnableTemporalLayer: config.SfuParams.Enabletemporallayer}
	conf.Router = sfu.RouterConfig{MaxBandwidth: config.SfuParams.Maxbandwidth, MaxBufferTime: config.SfuParams.Maxbuffertime, Simulcast: simulcastConfig}
	portRange := make([]uint16, 2)
	portRange[0] = config.SfuParams.Minport
	portRange[1] = config.SfuParams.Maxport
	conf.WebRTC = sfu.WebRTCConfig{SDPSemantics: config.SfuParams.Sdpsemantics, ICEPortRange: portRange}
	serverConfigs := GetICEServerConfigs()
	conf.WebRTC.ICEServers = serverConfigs

	s := sfu.NewSFU(conf)
	sfuPeerPool.SFU = s
	sfuPeerPool.sfuPeers = make(map[string][]*SfuPeer)

	return sfuPeerPool
}

func GetICEServerConfigs() []sfu.ICEServerConfig {
	ICEServers := make([]sfu.ICEServerConfig, 0)
	for i, urls := range config.SfuParams.Urls {
		iceServer := sfu.ICEServerConfig{}
		iceServer.URLs = urls
		if i < len(config.SfuParams.Usernames) {
			username := config.SfuParams.Usernames[i]
			if username != "" {
				iceServer.Username = username
			}
		}
		if i < len(config.SfuParams.Credentials) {
			credential := config.SfuParams.Credentials[i]
			if credential != "" {
				iceServer.Credential = credential
			}
		}
		ICEServers = append(ICEServers, iceServer)
	}

	return ICEServers
}

func GetSfuPeerPool() *SfuPeerPool {

	return sfuPeerPool
}

func (this *SfuPeerPool) RegistEvent(name string, fn func(event *PoolEvent) (interface{}, error)) {
	if this.events == nil {
		this.events = make(map[string]func(event *PoolEvent) (interface{}, error), 0)
	}
	this.events[name] = fn
}

func (this *SfuPeerPool) UnregistEvent(name string) bool {
	if this.events == nil {
		return false
	}
	delete(this.events, name)
	return true
}

func (this *SfuPeerPool) EmitEvent(name string, event *PoolEvent) (interface{}, error) {
	if this.events == nil {
		return nil, errors.New("EventNotExist")
	}
	fn, ok := this.events[name]
	if ok {
		return fn(event)
	}
	return nil, errors.New("EventNotExist")
}

func (this *SfuPeerPool) signal(evt *PoolEvent) (interface{}, error) {
	targetPeerId := evt.Source.TargetPeerId
	sfuSignal, ok := evt.Data.(*SfuSignal)
	if !ok {
		return nil, errors.New("NotSfuSignal")
	}

	return Signal(sfuSignal, targetPeerId)
}

func (this *SfuPeerPool) Receive(netPeer *p2p.NetPeer, payload map[string]interface{}) (interface{}, error) {
	sfuSignal := Transform(payload)
	if sfuSignal == nil {
		return nil, errors.New("NotSfuSignal")
	}

	switch sfuSignal.SignalType {
	case "join":
		return this.join(netPeer, sfuSignal, true)
	case "offer":
		return this.offer(netPeer, sfuSignal, true)
	case "answer":
		return this.answer(netPeer, sfuSignal)
	case "trickle":
		return this.trickle(netPeer, sfuSignal)
	}

	return nil, errors.New("SignalTypeError")
}

func (this *SfuPeerPool) trickle(netPeer *p2p.NetPeer, sfuSignal *SfuSignal) (interface{}, error) {
	var sfuPeer = this.GetPeer(netPeer)
	if sfuPeer == nil {
		logger.Sugar.Errorf("sfuPeers:%v is not exist", netPeer)
		return nil, errors.New("NotExist")
	}
	err := sfuPeer.Trickle(*sfuSignal.Candidate, sfuSignal.Target)
	if err != nil {
		logger.Sugar.Errorf("%v", err.Error())
	}
	return nil, err
}

func (this *SfuPeerPool) answer(netPeer *p2p.NetPeer, sfuSignal *SfuSignal) (interface{}, error) {
	var sfuPeer = this.GetPeer(netPeer)
	if sfuPeer == nil {
		logger.Sugar.Errorf("sfuPeers:%v is not exist", netPeer)
		return nil, errors.New("NotExist")
	}
	err := sfuPeer.SetRemoteDescription(*sfuSignal.Sdp)
	if err != nil {
		logger.Sugar.Errorf("%v", err.Error())
	}
	return nil, err
}

func (this *SfuPeerPool) offer(netPeer *p2p.NetPeer, sfuSignal *SfuSignal, isSync bool) (interface{}, error) {
	var sfuPeer = this.GetPeer(netPeer)
	if sfuPeer == nil {
		logger.Sugar.Errorf("sfuPeers:%v is not exist, will recreate new sfuPeer", netPeer)
		return nil, errors.New("NotExist")
	}
	answer, err := sfuPeer.Answer(*sfuSignal.Sdp)
	if err != nil {
		return nil, err
	}
	answerSfuSignal := &SfuSignal{}
	answerSfuSignal.SignalType = "answer"
	answerSfuSignal.Sdp = answer
	if isSync {
		return answerSfuSignal, nil
	} else {
		_, err = Signal(answerSfuSignal, sfuPeer.GetPeer().TargetPeerId)
		if err != nil {
			logger.Sugar.Errorf("error sending answer %s", err.Error())
		}
	}

	return nil, nil
}

func (this *SfuPeerPool) join(netPeer *p2p.NetPeer, sfuSignal *SfuSignal, isSync bool) (interface{}, error) {
	var sfuPeer = this.GetPeer(netPeer)
	if sfuPeer != nil {
		logger.Sugar.Errorf("sfuPeers:%v is exist, will recreate new sfuPeer", netPeer)
		return nil, errors.New("Exist")
	}
	sfuPeer = NewSfuPeer(netPeer, nil)
	if sfuPeer == nil {
		return nil, errors.New("NewSfuPeerError")
	}

	sfuPeer.OnICEConnectionStateChange = func(state webrtc.ICEConnectionState) {
		sfuPeer.state = state
		event := &PoolEvent{Name: "onstatechange", Source: sfuPeer, Data: state}
		this.EmitEvent("onstatechange", event)
	}

	sfuPeer.OnOffer = func(offer *webrtc.SessionDescription) {
		offerSfuSignal := &SfuSignal{}
		offerSfuSignal.SignalType = "offer"
		offerSfuSignal.Sdp = offer
		_, err := Signal(offerSfuSignal, sfuPeer.GetPeer().TargetPeerId)
		if err != nil {
			logger.Sugar.Errorf("error sending offer %s", err.Error())
		}
	}

	sfuPeer.OnIceCandidate = func(candidate *webrtc.ICECandidateInit, target int) {
		trickleSfuSignal := &SfuSignal{}
		trickleSfuSignal.SignalType = "trickle"
		trickleSfuSignal.Candidate = candidate
		trickleSfuSignal.Target = target
		_, err := Signal(trickleSfuSignal, sfuPeer.GetPeer().TargetPeerId)
		if err != nil {
			logger.Sugar.Errorf("error sending offer %s", err.Error())
		}
	}

	answer, err := sfuPeer.Join(sfuSignal.Sid, *sfuSignal.Sdp)
	if err != nil {
		return nil, err
	}
	answerSfuSignal := &SfuSignal{}
	answerSfuSignal.SignalType = "answer"
	answerSfuSignal.Sdp = answer
	if isSync {
		return answerSfuSignal, nil
	} else {
		_, err = Signal(answerSfuSignal, sfuPeer.GetPeer().TargetPeerId)
		if err != nil {
			logger.Sugar.Errorf("error sending answer %s", err.Error())
		}
	}
	return nil, nil
}

func (this *SfuPeerPool) GetPeer(netPeer *p2p.NetPeer) *SfuPeer {
	this.lock.Lock()
	defer this.lock.Unlock()
	sfuPeers, ok := this.sfuPeers[netPeer.TargetPeerId]
	if !ok {
		logger.Sugar.Errorf("sfuPeers:%v is exist, will recreate new sfuPeer", netPeer)
		return nil
	}
	var sfuPeer *SfuPeer
	if sfuPeers != nil && len(sfuPeers) > 0 {
		for _, sfuPeer = range sfuPeers {
			// 如果连接没有完成
			if sfuPeer.ConnectPeerId == netPeer.ConnectPeerId && sfuPeer.ConnectSessionId == netPeer.ConnectSessionId {
				logger.Sugar.Infof("webrtcPeer:+  + ' exist, connected:%v", netPeer, sfuPeer.Connected())
				break
			}
		}
	}
	return sfuPeer
}

func (this *SfuPeerPool) leave(netPeer *p2p.NetPeer) error {
	var sfuPeer = this.GetPeer(netPeer)
	if sfuPeer == nil {
		logger.Sugar.Errorf("sfuPeers:%v is not exist, will recreate new sfuPeer", netPeer)
		return errors.New("NotExist")
	}
	return sfuPeer.Close()
}

func (this *SfuPeerPool) Remove(netPeer *p2p.NetPeer) error {
	this.lock.Lock()
	defer this.lock.Unlock()
	sfuPeers, ok := this.sfuPeers[netPeer.TargetPeerId]
	if !ok {
		logger.Sugar.Errorf("sfuPeers:%v is not exist", netPeer)
		return nil
	}
	if sfuPeers != nil && len(sfuPeers) > 0 {
		for i, sfuPeer := range sfuPeers {
			if sfuPeer.ConnectPeerId == netPeer.ConnectPeerId && sfuPeer.ConnectSessionId == netPeer.ConnectSessionId {
				logger.Sugar.Infof("sfuPeer:%v exist, connected:%v, will be left", netPeer, sfuPeer.Connected())
				sfuPeers = append(sfuPeers[:i], sfuPeers[i+1:]...)
				break
			}
		}
		if len(sfuPeers) == 0 {
			delete(this.sfuPeers, netPeer.TargetPeerId)
		}
	}

	return nil
}

func (this *SfuPeerPool) onStateChange(event *PoolEvent) (interface{}, error) {
	sfuPeer := event.Source
	state, ok := event.Data.(webrtc.ICEConnectionState)
	if ok {
		if state == webrtc.ICEConnectionStateFailed {
			sfuPeer.Close()
		} else if state == webrtc.ICEConnectionStateDisconnected || state == webrtc.ICEConnectionStateClosed {
			this.Remove(sfuPeer.NetPeer)
		} else if state == webrtc.ICEConnectionStateCompleted || state == webrtc.ICEConnectionStateConnected {
			logger.Sugar.Infof("sfuPeer:%v state changed, connected:%v", sfuPeer.TargetPeerId, sfuPeer.Connected())
		}
	}

	return nil, nil
}

func init() {
	sfuPeerPool = NewSfuPeerPool()
	sfuPeerPool.RegistEvent("signal", sfuPeerPool.signal)
	sfuPeerPool.RegistEvent("onstatechange", sfuPeerPool.onStateChange)
	dht.IonSignalAction.RegistReceiver(sfuPeerPool.Receive)
}
