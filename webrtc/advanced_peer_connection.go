package webrtc

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type AdvancedPeerConnection struct {
	basePeerConnection *BasePeerConnection
	peerId             string
	clientId           string
	name               string
	connectPeerId      string
	connectSessionId   string
	Room               *MeetingRoom
}

func CreateAdvancedPeerConnection(peerId string, initiator bool, clientId string, name string,
	connectPeerId string, connectSessionId string, room *MeetingRoom) *AdvancedPeerConnection {
	basePeerConnection := CreateBasePeerConnection(initiator)
	advancedPeerConnection := AdvancedPeerConnection{
		basePeerConnection: basePeerConnection,
		peerId:             peerId,
		clientId:           clientId,
		name:               name,
		connectPeerId:      connectPeerId,
		connectSessionId:   connectSessionId,
		Room:               room,
	}

	return &advancedPeerConnection
}

/**
创建新的Webrtc节点
*/
func (this *AdvancedPeerConnection) Init(iceServers []webrtc.ICEServer) error {
	if iceServers == nil {
		iceServers = GetICEServers()
	}
	extension := &SignalExtension{Room: this.Room}
	this.basePeerConnection.Init(extension, nil)
	//if this.room != nil && this.room.RoomId != "" {
	//	ok := this.On(EVENT_CONNECT, func(event *PoolEvent) (interface{}, error) {
	//		return GetSfuPool().Join(this)
	//	})
	//	if !ok {
	//		logger.Sugar.Errorf("RegistEvent %v fail")
	//	}
	//	ok = this.On(EVENT_CLOSE, func(event *PoolEvent) (interface{}, error) {
	//		err := GetSfuPool().Leave(this)
	//
	//		return nil, err
	//	})
	//	if !ok {
	//		logger.Sugar.Errorf("RegistEvent %v fail")
	//	}
	//}
	/**
	 * 可以发起信号
	 */
	this.basePeerConnection.On(WebrtcEventType_signal, func(evt *WebrtcEvent) (interface{}, error) {
		logger.Sugar.Infof("can signal to peer:%v;connectPeer:%v;session:%v", this.peerId, this.connectPeerId, this.connectSessionId)
		webrtcEvent := &WebrtcEvent{
			Name:   string(WebrtcEventType_signal),
			PeerId: this.peerId, ClientId: this.clientId,
			ConnectPeerId: this.connectPeerId, ConnectSessionId: this.connectSessionId,
			Data: evt.Data,
		}

		return GetPeerConnectionPool().Emit(WebrtcEventType_signal, webrtcEvent)
	})
	//
	///**
	// * 连接建立
	// */
	this.basePeerConnection.On(WebrtcEventType_connected, func(evt *WebrtcEvent) (interface{}, error) {
		logger.Sugar.Infof("connected to peer:%v;connectPeer:%v;session:%v, can send message", this.peerId, this.connectPeerId, this.connectSessionId)
		this.SendText("hello,胡劲松")
		webrtcEvent := &WebrtcEvent{Name: string(WebrtcEventType_connected), PeerId: this.peerId, ClientId: this.clientId,
			ConnectPeerId: this.connectPeerId, ConnectSessionId: this.connectSessionId}
		return GetPeerConnectionPool().Emit(WebrtcEventType_connected, webrtcEvent)
	})
	//
	this.basePeerConnection.On(WebrtcEventType_closed, func(evt *WebrtcEvent) (interface{}, error) {
		logger.Sugar.Infof("connected to peer:%v;connectPeer:%v;session:%v, is closed", this.peerId, this.connectPeerId, this.connectSessionId)
		GetPeerConnectionPool().remove(this.peerId, this.clientId)
		webrtcEvent := &WebrtcEvent{Name: string(WebrtcEventType_closed), PeerId: this.peerId, ClientId: this.clientId,
			ConnectPeerId: this.connectPeerId, ConnectSessionId: this.connectSessionId}
		return GetPeerConnectionPool().Emit(WebrtcEventType_closed, webrtcEvent)
	})
	//
	///**
	// * 收到数据
	// */
	this.basePeerConnection.On(WebrtcEventType_message, func(evt *WebrtcEvent) (interface{}, error) {
		data := evt.Data.([]byte)
		logger.Sugar.Infof("got a message from peer:%v", string(data))
		webrtcEvent := &WebrtcEvent{Name: string(WebrtcEventType_message), PeerId: this.peerId, ClientId: this.clientId,
			ConnectPeerId: this.connectPeerId, ConnectSessionId: this.connectSessionId, Data: evt.Data}
		return GetPeerConnectionPool().Emit(WebrtcEventType_message, webrtcEvent)
	})
	//
	this.basePeerConnection.On(WebrtcEventType_stream, func(evt *WebrtcEvent) (interface{}, error) {
		webrtcEvent := &WebrtcEvent{Name: string(WebrtcEventType_stream), PeerId: this.peerId, ClientId: this.clientId,
			ConnectPeerId: this.connectPeerId, ConnectSessionId: this.connectSessionId, Data: evt.Data}
		return GetPeerConnectionPool().Emit(WebrtcEventType_stream, webrtcEvent)
	})
	//
	this.basePeerConnection.On(WebrtcEventType_track, func(evt *WebrtcEvent) (interface{}, error) {
		logger.Sugar.Infof("track")
		webrtcEvent := &WebrtcEvent{Name: string(WebrtcEventType_track), PeerId: this.peerId, ClientId: this.clientId,
			ConnectPeerId: this.connectPeerId, ConnectSessionId: this.connectSessionId, Data: evt.Data}
		return GetPeerConnectionPool().Emit(WebrtcEventType_track, webrtcEvent)
	})
	//
	this.basePeerConnection.On(WebrtcEventType_error, func(evt *WebrtcEvent) (interface{}, error) {
		logger.Sugar.Errorf("error:%v", evt.Data)
		webrtcEvent := &WebrtcEvent{Name: string(WebrtcEventType_error), PeerId: this.peerId, ClientId: this.clientId,
			ConnectPeerId: this.connectPeerId, ConnectSessionId: this.connectSessionId, Data: evt.Data}
		return GetPeerConnectionPool().Emit(WebrtcEventType_error, webrtcEvent)
	})
	return nil
}

func (this *AdvancedPeerConnection) Id() string {
	return this.basePeerConnection.id
}

/**
设置本节点的信号
*/
func (this *AdvancedPeerConnection) onSignal(webrtcSignal *WebrtcSignal) {
	this.basePeerConnection.onSignal(webrtcSignal)
}

/**
获取本节点的webrtc连接状态
*/
func (this *AdvancedPeerConnection) status() PeerConnectionStatus {
	return this.basePeerConnection.status
}

/**
利用缺省数据通道发送数据
*/
func (this *AdvancedPeerConnection) Send(data []byte) error {
	if this.connected() {
		return this.basePeerConnection.Send(data)
	} else {
		logger.Sugar.Errorf("peerId:%v;connectPeer:%v session:%v  webrtc connection state is not connected", this.peerId, this.connectPeerId, this.connectSessionId)
	}

	return nil
}

/**
利用缺省数据通道发送数据
*/
func (this *AdvancedPeerConnection) SendText(data string) error {
	if this.connected() {
		return this.basePeerConnection.SendText(data)
	} else {
		logger.Sugar.Errorf("peerId:%v;connectPeer:%v session:%v  webrtc connection state is not connected", this.peerId, this.connectPeerId, this.connectSessionId)
	}

	return nil
}

func (this *AdvancedPeerConnection) connected() bool {
	return this.basePeerConnection.status == PeerConnectionStatus_connected
}

/**
关闭本节点的wenbrtc连接，并从池中删除
*/
func (this *AdvancedPeerConnection) Close() {
	this.basePeerConnection.Close()
	//peer.webrtcPeerPool.remove(this.NetPeer)
}

func (this *AdvancedPeerConnection) negotiate() {
	this.basePeerConnection.negotiate()
}

func (this *AdvancedPeerConnection) ReadRTP(track *webrtc.TrackRemote) (*rtp.Packet, error) {
	return this.basePeerConnection.ReadRTP(track)
}

func (this *AdvancedPeerConnection) WriteRTP(sender *webrtc.RTPSender, packet *rtp.Packet) error {
	return this.basePeerConnection.WriteRTP(sender, packet)
}

func (this *AdvancedPeerConnection) CreateTrack(c *webrtc.RTPCodecCapability, trackId string, streamId string) (webrtc.TrackLocal, error) {
	return this.basePeerConnection.CreateTrack(c, trackId, streamId)
}

func (this *AdvancedPeerConnection) AddTrack(track webrtc.TrackLocal) (*webrtc.RTPSender, error) {
	return this.basePeerConnection.AddTrack(track)
}

func (this *AdvancedPeerConnection) removeStream(streamId string,
) {
	this.basePeerConnection.RemoveStream(streamId)
}

func (this *AdvancedPeerConnection) removeTrack(trackId string) {

	this.basePeerConnection.RemoveTrack(trackId)
}

func (this *AdvancedPeerConnection) replaceTrack(oldTrack webrtc.TrackLocal, newTrack webrtc.TrackLocal) {
	this.basePeerConnection.ReplaceTrack(oldTrack, newTrack)
}

func (this *AdvancedPeerConnection) GetSenders(streamId string) []*webrtc.RTPSender {
	return this.basePeerConnection.GetSenders(streamId)
}

func (this *AdvancedPeerConnection) GetTrackRemotes(streamId string) []*webrtc.TrackRemote {
	return this.basePeerConnection.GetTrackRemotes(streamId)
}
