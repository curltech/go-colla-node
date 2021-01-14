package peer

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/p2p"
	webrtc2 "github.com/curltech/go-colla-node/webrtc"
	"github.com/curltech/go-colla-node/webrtc/peer/simplepeer"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"time"
)

type IWebrtcPeer interface {
	Id() string
	RegistEvent(name string, fn func(event *PoolEvent) (interface{}, error)) bool
	UnregistEvent(name string) bool
	EmitEvent(name string, event *PoolEvent) (interface{}, error)
	Create(targetPeerId string, iceServer []webrtc.ICEServer, initiator bool, options *webrtc2.WebrtcOption, router *simplepeer.Router)
	Router() *simplepeer.Router
	Signal(webrtcSignal *simplepeer.WebrtcSignal)
	Connected() bool
	GetPeer() *p2p.NetPeer
	SetPeer(connectPeerId string, connectSessionId string, peerType string)
	Send(data []byte) error
	SendText(data string) error
	Destroy(err error)
	ReadRTP(track *webrtc.TrackRemote) (*rtp.Packet, error)
	WriteRTP(sender *webrtc.RTPSender, packet *rtp.Packet) error
	CreateTrack(c *webrtc.RTPCodecCapability, trackId string, streamId string) (webrtc.TrackLocal, error)
	AddTrack(track webrtc.TrackLocal) (*webrtc.RTPSender, error)
	GetSenders(streamId string) []*webrtc.RTPSender
	GetTrackRemotes(streamId string) []*webrtc.TrackRemote
}

type WebrtcSimplePeer struct {
	*p2p.NetPeer
	simplePeer *simplepeer.SimplePeer
	iceServer  []webrtc.ICEServer
	options    *webrtc2.WebrtcOption
	start      time.Time
	end        time.Time
	events     map[string]func(event *PoolEvent) (interface{}, error)
	router     *simplepeer.Router
}

/**
 * 初始化一个SimplePeer的配置参数
{
	initiator: false,//是否是发起节点
	channelConfig: {},
	channelName: '<random string>',
	config: { iceServers: [{ urls: 'stun:stun.l.google.com:19302' }, { urls: 'stun:global.stun.twilio.com:3478?transport=udp' }] },
	offerOptions: {},
	answerOptions: {},
	sdpTransform: function (sdp) { return sdp },
	stream: false,
	streams: [],
	trickle: true,
	allowHalfTrickle: false,
	objectMode: false
}
*/

func (this *WebrtcSimplePeer) RegistEvent(name string, fn func(event *PoolEvent) (interface{}, error)) bool {
	if this.events == nil {
		this.events = make(map[string]func(event *PoolEvent) (interface{}, error), 0)
	}
	_, ok := this.events[name]
	this.events[name] = fn

	return !ok
}

func (this *WebrtcSimplePeer) UnregistEvent(name string) bool {
	if this.events == nil {
		return false
	}
	delete(this.events, name)
	return true
}

func (this *WebrtcSimplePeer) EmitEvent(name string, event *PoolEvent) (interface{}, error) {
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
创建新的Webrtc节点
*/
func (this *WebrtcSimplePeer) Create(targetPeerId string, iceServer []webrtc.ICEServer, initiator bool, options *webrtc2.WebrtcOption, router *simplepeer.Router) {
	this.TargetPeerId = targetPeerId
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
	this.options = options
	this.start = time.Now()
	this.simplePeer = &simplepeer.SimplePeer{}
	this.simplePeer.Create(this.options)
	this.router = router
	if this.router != nil && this.router.RoomId != "" {
		ok := this.RegistEvent(webrtc2.EVENT_CONNECT, func(event *PoolEvent) (interface{}, error) {
			return GetSfuPool().Join(this)
		})
		if !ok {
			logger.Errorf("RegistEvent %v fail", webrtc2.EVENT_CONNECT)
		}
		ok = this.RegistEvent(webrtc2.EVENT_CLOSE, func(event *PoolEvent) (interface{}, error) {
			err := GetSfuPool().Leave(this)

			return nil, err
		})
		if !ok {
			logger.Errorf("RegistEvent %v fail", webrtc2.EVENT_CLOSE)
		}
	}
	/**
	 * 可以发起信号
	 */
	this.simplePeer.RegistEvent(webrtc2.EVENT_SIGNAL, func(evt *webrtc2.PeerEvent) (interface{}, error) {
		logger.Infof("can signal to peer:%v;connectPeer:%v;session:", this.TargetPeerId, this.ConnectPeerId, this.ConnectSessionId)
		poolEvent := &PoolEvent{Name: webrtc2.EVENT_SIGNAL, Source: this, Data: evt.Data}
		this.EmitEvent(webrtc2.EVENT_SIGNAL, poolEvent)
		return webrtcPeerPool.EmitEvent(webrtc2.EVENT_SIGNAL, poolEvent)
	})

	/**
	 * 连接建立
	 */
	this.simplePeer.RegistEvent(webrtc2.EVENT_CONNECT, func(evt *webrtc2.PeerEvent) (interface{}, error) {
		logger.Infof("connected to peer:%v;connectPeer:%v;session:%v, can send message", this.TargetPeerId, this.ConnectPeerId, this.ConnectSessionId)
		this.end = time.Now()
		logger.Infof("connect time:%v", (this.end.Unix() - this.start.Unix()))
		this.SendText("hello,胡劲松")
		poolEvent := &PoolEvent{Name: webrtc2.EVENT_CONNECT, Source: this}
		this.EmitEvent(webrtc2.EVENT_CONNECT, poolEvent)
		return webrtcPeerPool.EmitEvent(webrtc2.EVENT_CONNECT, poolEvent)
	})

	this.simplePeer.RegistEvent(webrtc2.EVENT_CLOSE, func(evt *webrtc2.PeerEvent) (interface{}, error) {
		logger.Infof("connected to peer:%v;connectPeer:%v;session:%v, is closed", this.TargetPeerId, this.ConnectPeerId, this.ConnectSessionId)
		webrtcPeerPool.remove(this.NetPeer)
		poolEvent := &PoolEvent{Name: webrtc2.EVENT_CLOSE, Source: this}
		this.EmitEvent(webrtc2.EVENT_CLOSE, poolEvent)
		return webrtcPeerPool.EmitEvent(webrtc2.EVENT_CLOSE, poolEvent)
	})

	/**
	 * 收到数据
	 */
	this.simplePeer.RegistEvent(webrtc2.EVENT_DATA, func(evt *webrtc2.PeerEvent) (interface{}, error) {
		data := evt.Data.([]byte)
		logger.Infof("got a message from peer:%v", string(data))
		poolEvent := &PoolEvent{Name: webrtc2.EVENT_DATA, Source: this, Data: evt.Data}
		this.EmitEvent(webrtc2.EVENT_DATA, poolEvent)
		return webrtcPeerPool.EmitEvent(webrtc2.EVENT_DATA, poolEvent)
	})

	this.simplePeer.RegistEvent(webrtc2.EVENT_STREAM, func(evt *webrtc2.PeerEvent) (interface{}, error) {
		poolEvent := &PoolEvent{Name: webrtc2.EVENT_STREAM, Source: this, Data: evt.Data}
		this.EmitEvent(webrtc2.EVENT_STREAM, poolEvent)
		return webrtcPeerPool.EmitEvent(webrtc2.EVENT_STREAM, poolEvent)
	})

	this.simplePeer.RegistEvent(webrtc2.EVENT_TRACK, func(evt *webrtc2.PeerEvent) (interface{}, error) {
		logger.Infof("track")
		poolEvent := &PoolEvent{Name: webrtc2.EVENT_TRACK, Source: this, Data: evt.Data}
		this.EmitEvent(webrtc2.EVENT_TRACK, poolEvent)
		return webrtcPeerPool.EmitEvent(webrtc2.EVENT_TRACK, poolEvent)
	})

	this.simplePeer.RegistEvent(webrtc2.EVENT_ERROR, func(evt *webrtc2.PeerEvent) (interface{}, error) {
		logger.Errorf("error:%v", evt.Data)
		// 重试的次数需要限制，超过则从池中删除
		//this.init(this._targetPeerId, this._iceServer, null, this._options)
		poolEvent := &PoolEvent{Name: webrtc2.EVENT_ERROR, Source: this, Data: evt.Data}
		this.EmitEvent(webrtc2.EVENT_ERROR, poolEvent)
		return webrtcPeerPool.EmitEvent(webrtc2.EVENT_ERROR, poolEvent)
	})
}

func (this *WebrtcSimplePeer) Router() *simplepeer.Router {
	return this.router
}

/**
设置本节点的信号
*/
func (this *WebrtcSimplePeer) Signal(webrtcSignal *simplepeer.WebrtcSignal) {
	this.simplePeer.Signal(webrtcSignal)
}

/**
获取本节点的webrtc连接状态
*/
func (this *WebrtcSimplePeer) Connected() bool {
	return this.simplePeer.Connected()
}

func (this *WebrtcSimplePeer) Id() string {
	return this.TargetPeerId + ":" + this.ConnectPeerId + ":" + this.ConnectSessionId
}

func (this *WebrtcSimplePeer) GetPeer() *p2p.NetPeer {
	return this.NetPeer
}

func (this *WebrtcSimplePeer) SetPeer(connectPeerId string, connectSessionId string, peerType string) {
	this.ConnectPeerId = connectPeerId
	this.ConnectSessionId = connectSessionId
	this.PeerType = peerType
}

/**
利用缺省数据通道发送数据
*/
func (this *WebrtcSimplePeer) Send(data []byte) error {
	if this.simplePeer.Connected() {
		return this.simplePeer.Send(data)
	} else {
		logger.Errorf("peerId:%v;connectPeer:%v session:%v  webrtc connection state is not connected", this.TargetPeerId, this.ConnectPeerId, this.ConnectSessionId)
	}

	return nil
}

/**
利用缺省数据通道发送数据
*/
func (this *WebrtcSimplePeer) SendText(data string) error {
	if this.simplePeer.Connected() {
		return this.simplePeer.SendText(data)
	} else {
		logger.Errorf("peerId:%v;connectPeer:%v session:%v  webrtc connection state is not connected", this.TargetPeerId, this.ConnectPeerId, this.ConnectSessionId)
	}

	return nil
}

/**
关闭本节点的wenbrtc连接，并从池中删除
*/
func (this *WebrtcSimplePeer) Destroy(err error) {
	this.simplePeer.Destroy(err)
	webrtcPeerPool.remove(this.NetPeer)
}

func (this *WebrtcSimplePeer) ReadRTP(track *webrtc.TrackRemote) (*rtp.Packet, error) {
	return this.simplePeer.ReadRTP(track)
}

func (this *WebrtcSimplePeer) WriteRTP(sender *webrtc.RTPSender, packet *rtp.Packet) error {
	return this.simplePeer.WriteRTP(sender, packet)
}

func (this *WebrtcSimplePeer) CreateTrack(c *webrtc.RTPCodecCapability, trackId string, streamId string) (webrtc.TrackLocal, error) {
	return this.simplePeer.CreateTrack(c, trackId, streamId)
}

func (this *WebrtcSimplePeer) AddTrack(track webrtc.TrackLocal) (*webrtc.RTPSender, error) {
	return this.simplePeer.AddTrack(track)
}

func (this *WebrtcSimplePeer) GetSenders(streamId string) []*webrtc.RTPSender {
	return this.simplePeer.GetSenders(streamId)
}

func (this *WebrtcSimplePeer) GetTrackRemotes(streamId string) []*webrtc.TrackRemote {
	return this.simplePeer.GetTrackRemotes(streamId)
}
