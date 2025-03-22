package sfu

//
//import (
//	"errors"
//	"github.com/curltech/go-colla-node/p2p"
//	webrtc2 "github.com/curltech/go-colla-node/webrtc"
//	"github.com/pion/ion-sfu/pkg/sfu"
//	"github.com/pion/webrtc/v4"
//	"time"
//)
//
///**
//代表两个peer之间的webrtc连接
//*/
//type SfuPeer struct {
//	peer *sfu.PeerLocal
//	*p2p.NetPeer
//	iceServer []webrtc.ICEServer
//	start     time.Time
//	end       time.Time
//	events    map[string]func(event *PoolEvent) (interface{}, error)
//	state     webrtc.ICEConnectionState
//}
//
//func NewSfuPeer(netPeer *p2p.NetPeer, iceServer []webrtc.ICEServer) *SfuPeer {
//	sfuPeerPool := GetSfuPeerPool()
//	peer := sfu.NewPeer(sfuPeerPool)
//	sfuPeer := &SfuPeer{}
//	sfuPeer.peer = peer
//	p := p2p.NetPeer{}
//	sfuPeer.NetPeer = &p
//	sfuPeer.TargetPeerId = netPeer.TargetPeerId
//	sfuPeer.ConnectPeerId = netPeer.ConnectPeerId
//	sfuPeer.ConnectSessionId = netPeer.ConnectSessionId
//	if iceServer == nil {
//		iceServer = webrtc2.GetICEServers()
//	}
//	sfuPeer.iceServer = iceServer
//	sfuPeer.start = time.Now()
//
//	sfuPeers, ok := sfuPeerPool.sfuPeers[netPeer.TargetPeerId]
//	if !ok {
//		sfuPeers = make([]*SfuPeer, 0)
//	}
//	sfuPeers = append(sfuPeers, sfuPeer)
//	sfuPeerPool.sfuPeers[netPeer.TargetPeerId] = sfuPeers
//
//	return sfuPeer
//}
//
//func (this *SfuPeer) RegistEvent(name string, fn func(event *PoolEvent) (interface{}, error)) bool {
//	if this.events == nil {
//		this.events = make(map[string]func(event *PoolEvent) (interface{}, error), 0)
//	}
//	_, ok := this.events[name]
//	this.events[name] = fn
//
//	return !ok
//}
//
//func (this *SfuPeer) UnregistEvent(name string) bool {
//	if this.events == nil {
//		return false
//	}
//	delete(this.events, name)
//	return true
//}
//
//func (this *SfuPeer) EmitEvent(name string, event *PoolEvent) (interface{}, error) {
//	if this.events == nil {
//		return nil, errors.New("EventNotExist")
//	}
//	fn, ok := this.events[name]
//	if ok {
//		return fn(event)
//	}
//	return nil, errors.New("EventNotExist")
//}
//
//func (this *SfuPeer) Id() string {
//	return this.TargetPeerId + ":" + this.ConnectPeerId + ":" + this.ConnectSessionId
//}
//
//func (this *SfuPeer) GetPeer() *p2p.NetPeer {
//	return this.NetPeer
//}
//
//func (this *SfuPeer) SetPeer(connectPeerId string, connectSessionId string, peerType string) {
//	this.NetPeer.ConnectPeerId = connectPeerId
//	this.NetPeer.ConnectSessionId = connectSessionId
//	this.NetPeer.PeerType = peerType
//}
//
//func (this *SfuPeer) Connected() bool {
//	if this.state == webrtc.ICEConnectionStateConnected || this.state == webrtc.ICEConnectionStateCompleted {
//		return true
//	}
//
//	return false
//}
