package webrtc

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/p2p/chain/action/dht"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/pion/webrtc/v3"
	"time"
)

type PeerConnectionPool struct {
	peerConnections map[string]map[string]*AdvancedPeerConnection
	peerId          string
	clientId        string
	peerPublicKey   libp2pcrypto.PubKey
	events          map[WebrtcEventType]func(event *WebrtcEvent) (interface{}, error)
}

var peerConnectionPool *PeerConnectionPool = &PeerConnectionPool{
	peerConnections: make(map[string]map[string]*AdvancedPeerConnection),
}

func GetPeerConnectionPool() *PeerConnectionPool {
	peerConnectionPool.peerId = global.Global.PeerId.String()
	peerConnectionPool.clientId = global.Global.Host.ID().String()
	peerConnectionPool.peerPublicKey = global.Global.PeerPublicKey

	return peerConnectionPool
}

func (this *PeerConnectionPool) On(name WebrtcEventType, fn func(event *WebrtcEvent) (interface{}, error)) {
	if this.events == nil {
		this.events = make(map[WebrtcEventType]func(event *WebrtcEvent) (interface{}, error), 0)
	}
	this.events[name] = fn
}

func (this *PeerConnectionPool) Emit(name WebrtcEventType, event *WebrtcEvent) (interface{}, error) {
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
func (this *PeerConnectionPool) get(peerId string) []*AdvancedPeerConnection {
	peerConnections, ok := this.peerConnections[peerId]
	pcs := make([]*AdvancedPeerConnection, 0)
	if ok {
		for _, peerConnection := range peerConnections {
			pcs = append(pcs, peerConnection)
		}
		return pcs
	}

	return nil
}

//
func (this *PeerConnectionPool) getOne(peerId string, clientId string) *AdvancedPeerConnection {
	peerConnections, ok := this.peerConnections[peerId]
	if ok && peerConnections != nil && len(peerConnections) > 0 {
		for _, peerConnection := range peerConnections {
			if clientId == peerConnection.clientId {
				return peerConnection
			}
		}
	}

	return nil
}

//
func (this *PeerConnectionPool) put(
	peerId string,
	advancedPeerConnection *AdvancedPeerConnection,
	clientId string,
) {
	peerConnections, ok := this.peerConnections[peerId]
	if !ok {
		peerConnections = make(map[string]*AdvancedPeerConnection)
		this.peerConnections[peerId] = peerConnections
	}
	_, ok = peerConnections[clientId]
	if ok {
		logger.Sugar.Warnf("old peerId:$peerId clientId:$clientId is exist!")
	}
	peerConnections[clientId] = advancedPeerConnection
}

//
func (this *PeerConnectionPool) create(peerId string, clientId string, name string, connectPeerId string, connectSessionId string, room *MeetingRoom,
	iceServers []webrtc.ICEServer) (*AdvancedPeerConnection, error) {
	//如果已经存在，先关闭删除
	var peerConnection *AdvancedPeerConnection = this.getOne(peerId, clientId)
	if peerConnection != nil {
		this.Close(peerId, clientId)
		logger.Sugar.Infof(
			"peerId:$peerId clientId:$clientId is closed and will be re-created!")
	}
	//创建新的主叫方
	peerConnection =
		CreateAdvancedPeerConnection(peerId, true, clientId, name, connectPeerId, connectSessionId, room)
	err := peerConnection.Init(iceServers)
	if err != nil {
		logger.Sugar.Errorf("webrtcPeer.init fail")
		return nil, err
	}
	peerConnection.negotiate()
	this.put(peerId, peerConnection, clientId)

	return peerConnection, nil
}

func (this *PeerConnectionPool) remove(peerId string, clientId string) map[string]*AdvancedPeerConnection {
	peerConnections, ok :=
		this.peerConnections[peerId]
	if !ok {
		return nil
	}
	removePeerConnections := make(map[string]*AdvancedPeerConnection)
	if len(peerConnections) > 0 {
		if clientId == "" {
			removePeerConnections = peerConnections
			delete(this.peerConnections, peerId)
		} else {
			peerConnection, ok := peerConnections[clientId]
			delete(peerConnections, clientId)
			if ok {
				removePeerConnections[clientId] =
					peerConnection
			}
			logger.Sugar.Infof("remove peerConnection peerId:$peerId,clientId:$clientId")
		}
		if len(peerConnections) == 0 {
			delete(this.peerConnections, peerId)
		}

		return removePeerConnections
	}
	return nil
}

///主动关闭，从池中移除连接
func (this *PeerConnectionPool) Close(peerId string, clientId string) bool {
	removePeerConnections :=
		this.remove(peerId, clientId)
	if removePeerConnections != nil && len(removePeerConnections) > 0 {
		for _, removePeerConnection := range removePeerConnections {
			if clientId == "" || clientId == removePeerConnection.clientId {
				removePeerConnection.Close()
				logger.Sugar.Infof("peerId:$peerId clientId:$clientId is closed!")
			}
		}
		return true
	}
	return false
}

/// 获取连接已经建立的连接，可能是多个
/// @param peerId
func (this *PeerConnectionPool) getConnected(peerId string) []*AdvancedPeerConnection {
	peerConnections_ := make([]*AdvancedPeerConnection, 0)
	peerConnections := this.get(peerId)
	if peerConnections != nil {
		for _, peerConnection := range peerConnections {
			if peerConnection.connected() {
				peerConnections_ = append(peerConnections_, peerConnection)
			}
		}
	}
	if len(peerConnections_) > 0 {
		return peerConnections_
	}

	return nil
}

func (this *PeerConnectionPool) getAll() []*AdvancedPeerConnection {
	peerConnections := make([]*AdvancedPeerConnection, 0)
	for _, peers := range this.peerConnections {
		for _, peer := range peers {
			peerConnections = append(peerConnections, peer)
		}
	}
	return peerConnections
}

///清除过一段时间仍没有连接上的连接
func (this *PeerConnectionPool) clear() {
	removedPeerIds := make([]string, 0)
	for peerId, _ := range this.peerConnections {
		peerConnections, _ :=
			this.peerConnections[peerId]
		if peerConnections != nil && len(peerConnections) > 0 {
			removedClientIds := make([]string, 0)
			for _, peerConnection := range peerConnections {
				if peerConnection.basePeerConnection.status !=
					PeerConnectionStatus_connected {
					var start = peerConnection.basePeerConnection.start
					var now = time.Now().Nanosecond()
					var gap = now - start
					var limit = 20 * 3600
					if gap > limit {
						removedClientIds = append(removedClientIds, peerConnection.clientId)
						logger.Sugar.Errorf(
							"peerConnection peerId:${peerConnection.peerId},clientId:${peerConnection.clientId} is overtime unconnected")
					}
				}
			}
			for _, removedClientId := range removedClientIds {
				delete(peerConnections, removedClientId)
			}
			if len(peerConnections) == 0 {
				removedPeerIds = append(removedPeerIds, peerId)
			}
		}
	}
	for _, removedPeerId := range removedPeerIds {
		delete(this.peerConnections, removedPeerId)
	}
}

func (this *PeerConnectionPool) createIfNotExist(peerId string,
	clientId string,
	name string,
	connectPeerId string, connectSessionId string,
	room *MeetingRoom,
	iceServers []webrtc.ICEServer) (*AdvancedPeerConnection, error) {
	advancedPeerConnection :=
		this.getOne(peerId, clientId)
	if advancedPeerConnection == nil {
		logger.Sugar.Infof("advancedPeerConnection is null,create new one")
		advancedPeerConnection = CreateAdvancedPeerConnection(peerId, false,
			clientId, name, connectPeerId, connectSessionId, room)
		this.put(peerId, advancedPeerConnection, clientId)
		err := advancedPeerConnection.Init(iceServers)
		if err != nil {
			logger.Sugar.Errorf("webrtcPeer.init fail")
			return nil, err
		}
		logger.Sugar.Infof(
			"advancedPeerConnection ${advancedPeerConnection.basePeerConnection.id} init completed")
	}

	return advancedPeerConnection, nil
}

///接收到signal
func (this *PeerConnectionPool) onSignal(peerId string, signal map[string]interface{},
	clientId string, connectPeerId string, connectSessionId string) {
	var webrtcSignal *WebrtcSignal = Transform(signal)
	this.onWebrtcSignal(peerId, webrtcSignal, clientId, connectPeerId, connectSessionId)
}

/// 接收到信号服务器发来的signal的处理,没有完成，要仔细考虑多终端的情况
/// 如果发来的是answer,寻找主叫的peerId,必须找到，否则报错，找到后检查clientId
/// 如果发来的是offer,检查peerId，没找到创建一个新的被叫，如果找到，检查clientId
/// @param peerId 源peerId
/// @param connectSessionId
/// @param data
func (this *PeerConnectionPool) onWebrtcSignal(peerId string, signal *WebrtcSignal,
	clientId string, connectPeerId string, connectSessionId string) error {
	var signalType = signal.SignalType
	logger.Sugar.Warnf("receive signal type: $signalType from webrtcPeer: $peerId")

	var name string
	var iceServers []webrtc.ICEServer
	var room *MeetingRoom
	var extension = signal.SignalExtension
	if extension != nil {
		if peerId != extension.PeerId {
			logger.Sugar.Errorf("peerId:$peerId extension peerId:${extension.peerId} is not same")
			peerId = extension.PeerId
		}
		if clientId != extension.ClientId {
			logger.Sugar.Errorf("peerId:$peerId extension peerId:${extension.peerId} is not same")
			clientId = extension.ClientId
		}
		name = extension.Name
		iceServers = extension.IceServers
		room = extension.Room
	}

	///收到信号，连接已经存在，但是clientId为unknownClientId，表明自己是主叫，建立的时候对方的clientId未知
	///设置clientId和name
	advancedPeerConnection :=
		this.getOne(peerId, unknownClientId)
	if advancedPeerConnection != nil {
		advancedPeerConnection.clientId = clientId
		advancedPeerConnection.name = name
		this.remove(peerId, unknownClientId)
		this.put(peerId, advancedPeerConnection, clientId)
	}
	advancedPeerConnection = this.getOne(peerId, clientId)
	// peerId的连接存在，而且已经连接，报错
	if advancedPeerConnection != nil {
		if advancedPeerConnection.connected() {
			logger.Sugar.Warnf("peerId:$peerId clientId:$clientId is connected, maybe renegotiate")
		}
	}

	//连接不存在，创建被叫连接，使用传来的iceServers，保证使用相同的turn服务器
	// logger.Sugar.Infof("webrtcPeer:$peerId $clientId not exist, will create receiver")
	// if (iceServers != null) {
	//   for (var iceServer in iceServers) {
	//     if (iceServer['username'] == null) {
	//       iceServer['username'] = this.peerId!
	//       iceServer['credential'] = peerPublicKey.toString()
	//     }
	//   }
	// }

	//作为主叫收到被叫的answer
	if signalType == WebrtcSignalType_Sdp && signal.Sdp.Type.String() == "answer" {
		//符合的主叫不存在，说明存在多个同peerid的被叫，其他的被叫的answer先来，将主叫占用了
		//需要再建新的主叫
		if advancedPeerConnection == nil {
			logger.Sugar.Warnf("peerId:$peerId, clientId:$clientId has no master to match")
		}
	}
	if signalType == WebrtcSignalType_Candidate ||
		(signalType == WebrtcSignalType_Sdp && signal.Sdp.Type.String() == "offer") {
		advancedPeerConnection, err := this.createIfNotExist(peerId,
			clientId, name, connectPeerId, connectSessionId, room, iceServers)
		if err != nil {
			logger.Sugar.Errorf("createIfNotExist fail")
			return err
		}

		///收到对方的offer，自己应该是被叫
		if signalType == WebrtcSignalType_Sdp && signal.Sdp.Type.String() == "offer" {
			if advancedPeerConnection.basePeerConnection.initiator {
				//如果自己是主叫，比较peerId，如果自己的较大，则自己继续作为主叫，忽略offer信号
				//否则自己将作为被叫，接收offer信号
			}
		}
	}
	if advancedPeerConnection != nil {
		advancedPeerConnection.onSignal(signal)
	}
	return nil
}

/// 向peer发送信息，如果是多个，遍历发送
/// @param peerId
/// @param data
func (this *PeerConnectionPool) send(peerId string, data []byte,
	clientId string) {
	peerConnections := this.get(peerId)
	if peerConnections != nil && len(peerConnections) > 0 {
		//logger.w('send signal:${peerConnections.length}")
		for _, peerConnection := range peerConnections {
			if clientId == "" || peerConnection.clientId == clientId {
				peerConnection.Send(data)

			}
		}
	} else {
		logger.Sugar.Errorf("PeerConnection:$peerId,clientId$clientId is not exist, cannot send")
	}
	return
}

///收到发来的ChainMessage消息，进行后续的action处理
///webrtc的数据通道发来的消息可以是ChainMessage，也可以是简单的非ChainMessage
func (this *PeerConnectionPool) onMessage(event WebrtcEvent) {
	logger.Sugar.Infof("peerId: ${event.peerId} clientId:${event.clientId} is onMessage")

}

func (this *PeerConnectionPool) onStatus(event WebrtcEvent) {

}

func (this *PeerConnectionPool) onConnected(event WebrtcEvent) {

}

func (this *PeerConnectionPool) onClosed(event WebrtcEvent) {
	logger.Sugar.Infof("peerId: ${event.peerId} clientId:${event.clientId} is closed")
	this.remove(event.PeerId, event.ClientId)
}

func (this *PeerConnectionPool) onError(event WebrtcEvent) {
	logger.Sugar.Infof("peerId: ${event.peerId} clientId:${event.clientId} is error")
}

func (this *PeerConnectionPool) onAddStream(event WebrtcEvent) {
	logger.Sugar.Infof("peerId: ${event.peerId} clientId:${event.clientId} is onAddStream")
}

func (this *PeerConnectionPool) onRemoveStream(event WebrtcEvent) {
	logger.Sugar.Infof(
		"peerId: ${event.peerId} clientId:${event.clientId} is onRemoveStream")
}

func (this *PeerConnectionPool) onTrack(event WebrtcEvent) {
	logger.Sugar.Infof("peerId: ${event.peerId} clientId:${event.clientId} is onTrack")
}

func (this *PeerConnectionPool) onAddTrack(event WebrtcEvent) {
	logger.Sugar.Infof(
		"peerId: ${event.peerId} clientId:${event.clientId} is onAddTrack")
	//peerConnectionsController.add(event.peerId, clientId: event.clientId)
}

func (this *PeerConnectionPool) onRemoveTrack(event WebrtcEvent) {
	logger.Sugar.Infof(
		"peerId: ${event.peerId} clientId:${event.clientId} is onRemoveTrack")
}

func (this *PeerConnectionPool) removeStream(peerId string, streamId string,
	clientId string) {
	advancedPeerConnection :=
		this.getOne(peerId, clientId)
	if advancedPeerConnection != nil {
		advancedPeerConnection.removeStream(streamId)
	}
}

func (this *PeerConnectionPool) removeTrack(peerId string, trackId string,
	clientId string) {
	advancedPeerConnection :=
		this.getOne(peerId, clientId)
	if advancedPeerConnection != nil {
		advancedPeerConnection.removeTrack(trackId)
	}
}

func (this *PeerConnectionPool) replaceTrack(peerId string, oldTrack webrtc.TrackLocal, newTrack webrtc.TrackLocal,
	clientId string) {
	advancedPeerConnection :=
		this.getOne(peerId, clientId)
	if advancedPeerConnection != nil {
		advancedPeerConnection.replaceTrack(oldTrack, newTrack)
	}
}

func (this *PeerConnectionPool) status(peerId string, clientId string) PeerConnectionStatus {
	advancedPeerConnection :=
		this.getOne(peerId, clientId)
	if advancedPeerConnection != nil {
		return advancedPeerConnection.status()
	}

	return PeerConnectionStatus_none
}

///调用signalAction发送signal到信号服务器
func (this *PeerConnectionPool) signal(evt *WebrtcEvent) (interface{}, error) {
	peerId := evt.PeerId
	clientId := evt.ClientId
	advancedPeerConnection :=
		this.getOne(peerId, clientId)
	if advancedPeerConnection != nil && advancedPeerConnection.connected() {

		logger.Sugar.Warnf(
			"sent signal chatMessage by webrtc peerId:$peerId, clientId:$clientId, signal:$jsonStr")
	} else {
		result, err := dht.SignalAction.Signal(evt.ConnectPeerId, evt.Data, peerId)
		if err != nil {
			logger.Sugar.Errorf("signal err:$result")
		}
		return result, err
	}

	return nil, nil
}

func init() {
	peerConnectionPool.On(WebrtcEventType_signal, peerConnectionPool.signal)
	///注册收到信号的处理器
	dht.SignalAction.RegistReceiver("webrtc", peerConnectionPool.onSignal)
}
