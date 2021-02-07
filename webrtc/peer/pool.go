package peer

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/p2p"
	"github.com/curltech/go-colla-node/p2p/chain/action/dht"
	"github.com/curltech/go-colla-node/webrtc/peer/simplepeer"
)

type PoolEvent struct {
	Name   string
	Source IWebrtcPeer
	Data   interface{}
}

type WebrtcPeerPool struct {
	webrtcPeers map[string][]IWebrtcPeer
	events      map[string]func(event *PoolEvent) (interface{}, error)
}

var webrtcPeerPool = &WebrtcPeerPool{webrtcPeers: make(map[string][]IWebrtcPeer)}

func GetWebrtcPeerPool() *WebrtcPeerPool {
	return webrtcPeerPool
}

func (this *WebrtcPeerPool) RegistEvent(name string, fn func(event *PoolEvent) (interface{}, error)) {
	if this.events == nil {
		this.events = make(map[string]func(event *PoolEvent) (interface{}, error), 0)
	}
	this.events[name] = fn
}

func (this *WebrtcPeerPool) UnregistEvent(name string) bool {
	if this.events == nil {
		return false
	}
	delete(this.events, name)
	return true
}

func (this *WebrtcPeerPool) EmitEvent(name string, event *PoolEvent) (interface{}, error) {
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
 * 获取peerId的webrtc连接，可能是多个
 * 如果不存在，创建一个新的连接，发起连接尝试
 * 否则，根据connected状态判断连接是否已经建立
 * @param peerId
 */
func (this *WebrtcPeerPool) get(peerId string) []IWebrtcPeer {
	webrtcPeers, ok := this.webrtcPeers[peerId]
	if ok {
		return webrtcPeers
	}

	return nil
}

func (this *WebrtcPeerPool) getOne(peerId string, connectPeerId string, connectSessionId string) IWebrtcPeer {
	webrtcPeers, ok := this.webrtcPeers[peerId]
	if ok && webrtcPeers != nil && len(webrtcPeers) > 0 {
		for _, webrtcPeer := range webrtcPeers {
			p := webrtcPeer.GetPeer()
			if p.ConnectPeerId == connectPeerId && p.ConnectSessionId == connectSessionId {
				return webrtcPeer
			}
		}
	}

	return nil
}

func (this *WebrtcPeerPool) _create() IWebrtcPeer {
	p := p2p.NetPeer{}
	webrtcPeer := &WebrtcSimplePeer{}
	webrtcPeer.NetPeer = &p

	return webrtcPeer
}

func (this *WebrtcPeerPool) create(peerId string) IWebrtcPeer {
	webrtcPeers, ok := this.webrtcPeers[peerId]
	if ok {
		logger.Sugar.Errorf("webrtcPeer:%v is exist, will recreate new sender", peerId)
	}
	webrtcPeer := this._create()
	webrtcPeer.Create(peerId, nil, true, nil, nil)
	webrtcPeers = make([]IWebrtcPeer, 0)
	webrtcPeers = append(webrtcPeers, webrtcPeer)
	this.webrtcPeers[peerId] = webrtcPeers

	return webrtcPeer
}

func (this *WebrtcPeerPool) remove(netPeer *p2p.NetPeer) bool {
	webrtcPeers, ok := this.webrtcPeers[netPeer.TargetPeerId]
	if ok {
		if webrtcPeers != nil && len(webrtcPeers) > 0 {
			i := 0
			for _, webrtcPeer := range webrtcPeers {
				p := webrtcPeer.GetPeer()
				if p.ConnectPeerId == netPeer.ConnectPeerId && p.ConnectSessionId == netPeer.ConnectSessionId {
					webrtcPeers = append(webrtcPeers[:i], webrtcPeers[i+1:]...)
					break
				}
				i++
			}
			if webrtcPeers != nil && len(webrtcPeers) == 0 {
				delete(this.webrtcPeers, netPeer.TargetPeerId)
			}
		}

		return true
	} else {
		return false
	}
}

/**
 * 获取连接已经建立的连接，可能是多个
 * @param peerId
 */
func (this *WebrtcPeerPool) getConnectd(peerId string) []IWebrtcPeer {
	peers := make([]IWebrtcPeer, 0)
	webrtcPeers, ok := this.webrtcPeers[peerId]
	if ok {
		if webrtcPeers != nil && len(webrtcPeers) > 0 {
			for _, webrtcPeer := range webrtcPeers {
				if webrtcPeer.Connected() == true {
					peers = append(peers, webrtcPeer)
				}
			}
		}
	}
	if len(peers) > 0 {
		return peers
	}

	return nil
}

func (this *WebrtcPeerPool) getAll() []IWebrtcPeer {
	webrtcPeers := make([]IWebrtcPeer, 0)
	for _, peers := range this.webrtcPeers {
		for _, peer := range peers {
			webrtcPeers = append(webrtcPeers, peer)
		}
	}
	return webrtcPeers
}

/**
 * 接收到signal的场景，有如下多张场景
 * 1.自己是被动方，而且同peerId的连接从没有创建过
 * @param peerId
 * @param connectSessionId
 * @param data
 */
func (this *WebrtcPeerPool) Receive(netPeer *p2p.NetPeer, payload map[string]interface{}) {
	webrtcSignal := simplepeer.Transform(payload)
	signalType := webrtcSignal.SignalType
	if signalType != "" {
		logger.Sugar.Infof("webrtcPeer:%v receive signal type:%v", netPeer, signalType)
	}
	var webrtcPeer IWebrtcPeer = nil
	// 被动方创建WebrtcPeer，同peerId的连接从没有创建过，
	// 被动创建连接，设置peerId和connectPeerId，connectSessionId
	webrtcPeers, ok := this.webrtcPeers[netPeer.TargetPeerId]
	if !ok {
		logger.Sugar.Infof("webrtcPeer:%v not exist, will create receiver", netPeer)
		webrtcPeer = this._create()
		webrtcPeer.Create(netPeer.TargetPeerId, nil, false, nil, webrtcSignal.Router)
		webrtcPeer.SetPeer(netPeer.ConnectPeerId, netPeer.ConnectSessionId, "")
		webrtcPeers = make([]IWebrtcPeer, 0)
		webrtcPeers = append(webrtcPeers, webrtcPeer)
		this.webrtcPeers[netPeer.TargetPeerId] = webrtcPeers
	} else { // 同peerId的连接存在，那么
		if webrtcPeers != nil && len(webrtcPeers) > 0 {
			found := false
			for _, webrtcPeer = range webrtcPeers {
				// 如果连接没有完成
				p := webrtcPeer.GetPeer()
				if p.ConnectPeerId == "" {
					webrtcPeer.SetPeer(netPeer.ConnectPeerId, netPeer.ConnectSessionId, "")
					found = true
					break
				} else if p.ConnectPeerId == netPeer.ConnectPeerId && p.ConnectSessionId == netPeer.ConnectSessionId {
					found = true
					logger.Sugar.Infof("webrtcPeer:+  + ' exist, connected:%v", netPeer, webrtcPeer.Connected())
					break
				}
			}
			// 没有匹配的连接被发现，说明有多个客户端实例回应，这时创建新的主动连接请求，尝试建立新的连接
			if found == false {
				logger.Sugar.Infof("match webrtcPeer:%v not exist, will create sender", netPeer)
				webrtcPeer = this._create()
				webrtcPeer.Create(netPeer.TargetPeerId, nil, true, nil, webrtcSignal.Router)
				webrtcPeer.SetPeer(netPeer.ConnectPeerId, netPeer.ConnectSessionId, "")
				webrtcPeers = append(webrtcPeers, webrtcPeer)
				webrtcPeer = nil
			}
		}
	}
	if webrtcPeer != nil {
		logger.Sugar.Infof("webrtcPeer signal data:%v", webrtcSignal)
		webrtcPeer.Signal(webrtcSignal)
	}
}

/**
 * 向peer发送信息，如果是多个，遍历发送
 * @param peerId
 * @param data
 */
func (this *WebrtcPeerPool) send(peerId string, data []byte) {
	webrtcPeers, ok := this.webrtcPeers[peerId]
	if ok && webrtcPeers != nil && len(webrtcPeers) > 0 {
		for _, webrtcPeer := range webrtcPeers {
			webrtcPeer.Send(data)
		}
	}
}

func (this *WebrtcPeerPool) signal(evt *PoolEvent) (interface{}, error) {
	targetPeerId := evt.Source.GetPeer().TargetPeerId
	webrtcSignal, ok := evt.Data.(*simplepeer.WebrtcSignal)
	if !ok {
		return nil, errors.New("NotWebrtcSignal")
	}

	return simplepeer.Signal(webrtcSignal, targetPeerId)
}

func init() {
	webrtcPeerPool.RegistEvent("signal", webrtcPeerPool.signal)
	dht.SignalAction.RegistReceiver("webrtc", webrtcPeerPool.Receive)
}
