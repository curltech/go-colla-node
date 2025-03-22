package webrtc

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-core/util/security"
	"github.com/pion/webrtc/v4"
	"strings"
	"time"
)

type BasePeerConnection struct {
	id              string
	initiator       bool
	peerConnection  *webrtc.PeerConnection
	status          PeerConnectionStatus
	negotiateStatus NegotiateStatus
	//数据通道的状态是否打开
	dataChannelOpen bool
	//主动发送数据的通道
	dataChannel *webrtc.DataChannel
	//是否需要主动建立数据通道
	needDataChannel bool

	//本地媒体流渲染器数组
	localTracks map[string]map[string]webrtc.TrackLocal
	//媒体流的轨道，流和发送者之间的关系
	remoteTracks map[string]map[string]*webrtc.TrackRemote
	trackSenders map[string]map[string]*webrtc.RTPSender

	//外部使用时注册的回调方法，也就是注册事件
	//WebrtcEvent定义了事件的名称
	events map[WebrtcEventType]func(event *WebrtcEvent) (interface{}, error)

	//signal扩展属性，由外部传入，这个属性用于传递定制的属性
	//一般包括自己的iceServer，room，peerId，clientId，name
	extension *SignalExtension

	///从协商开始计时，连接成功结束，计算连接的时间
	///如果一直未结束，根据当前状态，可以进行重连操作
	///对主动方来说，发出candidate和offer后一直未得到answer回应，重发candidate和offer
	///对被动方来说，收到candidate但一直未收到offer，只能等待，或者发出answer一直未连接，重发answer
	start int
	end   int

	delayTimes     int
	reconnectTimes int
	// datachannel的收发池
	sendChan chan []byte

	messageSlice MessageSlice
}

func CreateBasePeerConnection(initiator bool) *BasePeerConnection {
	basePeerConnection := BasePeerConnection{
		initiator:       initiator,
		needDataChannel: true,
		dataChannelOpen: false,
		delayTimes:      20,
		reconnectTimes:  1,
		messageSlice:    MessageSlice{sliceBuffer: make(map[int][]byte)},
	}

	return &basePeerConnection
}

/*
*
注册事件，发生时调用外部函数
*/
func (this *BasePeerConnection) On(name WebrtcEventType, fn func(event *WebrtcEvent) (interface{}, error)) {
	if this.events == nil {
		this.events = make(map[WebrtcEventType]func(event *WebrtcEvent) (interface{}, error), 0)
	}
	this.events[name] = fn
}

/*
*
事件发生，调用外部函数
*/
func (this *BasePeerConnection) Emit(name WebrtcEventType, event *WebrtcEvent) (interface{}, error) {
	if this.events == nil {
		return nil, errors.New("EventNotExist")
	}
	fn, ok := this.events[name]
	if ok {
		return fn(event)
	}
	return nil, errors.New("EventNotExist")
}

/*
*
创建媒体引擎和API对象，在需要自定义音视频编码的时候使用
*/
func (this *BasePeerConnection) createApi() (*webrtc.API, error) {
	// Create a MediaEngine object to configure the supported codec
	mediaEngine := webrtc.MediaEngine{}

	// Setup the codecs you want to use.
	// We'll use a VP8 and Opus but you can also define your own
	params := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "video/VP8", ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        96,
	}
	err := mediaEngine.RegisterCodec(params, webrtc.RTPCodecTypeVideo)
	if err != nil {
		return nil, err
	}

	// Create the API object with the MediaEngine
	api := webrtc.NewAPI(webrtc.WithMediaEngine(&mediaEngine))

	return api, nil
}

func (this *BasePeerConnection) Init(extension *SignalExtension, localTracks []webrtc.TrackLocal) error {
	this.id = security.UUID()
	logger.Sugar.Infof("new peer %v", this.id)
	this.start = time.Time{}.Nanosecond()
	this.extension = extension

	// 1.Create a new RTCPeerConnection
	var err error
	// 使用缺省的音视频编码
	iceServers := extension.IceServers
	if iceServers == nil {
		iceServers = GetICEServers()
	}
	configuration := webrtc.Configuration{ICEServers: iceServers, PeerIdentity: this.id}
	this.peerConnection, err = webrtc.NewPeerConnection(configuration)
	if err != nil {
		logger.Sugar.Errorf("%v", err)
		return err
	}

	// 2. regist peerConnection event
	this.peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		this.onIceConnectionStateChange(state)
	})

	this.peerConnection.OnICEGatheringStateChange(func(state webrtc.ICEGatheringState) {
		this.onIceGatheringStateChange(state)
	})

	this.peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		this.onConnectionStateChange(state)
	})

	this.peerConnection.OnSignalingStateChange(func(state webrtc.SignalingState) {
		this.onSignalingStateChange(state)
	})

	this.peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		this.onIceCandidate(candidate)
	})

	this.peerConnection.OnNegotiationNeeded(func() {
		this.onNegotiationNeeded()
	})

	// 3. setup datachannel and regist datachannel event
	if this.initiator && this.needDataChannel {
		channelName := security.UUID()
		var id uint16 = 1
		var ordered bool = true
		var protocol string = "sctp"
		var negotiated bool = false
		var maxPacketLifeTime uint16 = 0
		var maxRetransmits uint16 = 0
		dataChannelInit := &webrtc.DataChannelInit{
			ID: &id, Ordered: &ordered, Negotiated: &negotiated, Protocol: &protocol,
			MaxRetransmits: &maxRetransmits, MaxPacketLifeTime: &maxPacketLifeTime,
		}
		this.dataChannel, _ = this.peerConnection.CreateDataChannel(channelName, dataChannelInit)
		this.setupDataChannel(this.dataChannel)
	} else {
		this.peerConnection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
			this.setupDataChannel(dataChannel)
		})
	}

	// 4. add local audio/video stream
	if localTracks != nil && len(localTracks) > 0 {
		for _, localTrack := range localTracks {
			this.AddTrack(localTrack)
		}
	}
	this.peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		this.onTrack(track, receiver)
	})

	this.status = PeerConnectionStatus_init

	return nil
}

func (this *BasePeerConnection) Connected() {
	if this.status == PeerConnectionStatus_connected {
		logger.Sugar.Errorf("PeerConnectionStatus has already connected")
		return
	}
	logger.Sugar.Infof("PeerConnectionStatus connected, webrtc connection is completed")
	this.end = time.Now().Nanosecond()
	this.status = PeerConnectionStatus_connected

	interval := this.end - this.start
	logger.Sugar.Infof("id:%v connected time:$interval", interval)
	this.Emit(WebrtcEventType_connected, &WebrtcEvent{})
}

func (this *BasePeerConnection) SetStatus(status PeerConnectionStatus) {
	data := make(map[string]PeerConnectionStatus)
	data["oldStatus"] = this.status
	data["newStatus"] = status
	this.Emit(WebrtcEventType_status, &WebrtcEvent{Data: data})
	this.status = status
}

func (this *BasePeerConnection) onIceConnectionStateChange(state webrtc.ICEConnectionState) {
	if this.status == PeerConnectionStatus_connected {
		logger.Sugar.Errorf("PeerConnectionStatus has already connected")
		return
	}

	this.Emit(WebrtcEventType_iceConnectionState, &WebrtcEvent{Data: state})
	if state == webrtc.ICEConnectionStateConnected ||
		state == webrtc.ICEConnectionStateCompleted {
		this.Connected()
	}
	if state == webrtc.ICEConnectionStateFailed ||
		state == webrtc.ICEConnectionStateClosed ||
		state == webrtc.ICEConnectionStateDisconnected {
		logger.Sugar.Errorf("Ice connection failed.")
		this.Close()
	}
}

func (this *BasePeerConnection) onIceGatheringStateChange(state webrtc.ICEGatheringState) {
	if this.status == PeerConnectionStatus_connected {
		logger.Sugar.Errorf("PeerConnectionStatus has already connected")
		return
	}
	this.Emit(WebrtcEventType_iceGatheringState, &WebrtcEvent{Data: state})
}

func (this *BasePeerConnection) onConnectionStateChange(state webrtc.PeerConnectionState) {

}

func (this *BasePeerConnection) onSignalingStateChange(state webrtc.SignalingState) {
	if this.status == PeerConnectionStatus_connected {
		logger.Sugar.Errorf("PeerConnectionStatus has already connected")
		return
	}
	if state == webrtc.SignalingStateStable {
		this.negotiateStatus = NegotiateStatus_negotiated
	}
	this.Emit(WebrtcEventType_signalingState, &WebrtcEvent{Data: state})
}

/*
*
对candidate的处理，pion建议的和simplepeer的实现有差异，体现在加入时间，发送时间，
pion的发送时间是在远程sdp被设置以后，之前一律候选，在收到sdp后统一发送，加入时间是在收到对方的candidate时马上加入，应该比较快
simplepeer的发送时间是onIceCandidate就发送，加入时间是在收到candidate或者sdp信号，sdp被设置后加入，包括候选一起加入，如果sdp未设置，一律候选
*/
func (this *BasePeerConnection) onIceCandidate(candidate *webrtc.ICECandidate) {
	if this.status == PeerConnectionStatus_closed {
		logger.Sugar.Errorf("PeerConnectionStatus closed")
		return
	}

	if candidate != nil {
		//发送candidate信号
		can := candidate.ToJSON()
		candidates := []*webrtc.ICECandidateInit{&can}
		webrtcSignal := WebrtcSignal{SignalType: WebrtcSignalType_Candidate,
			Candidates: candidates, SignalExtension: this.extension}
		this.Emit(
			WebrtcEventType_signal, &WebrtcEvent{Data: webrtcSignal})

	}
}

func (this *BasePeerConnection) onNegotiationNeeded() {
	logger.Sugar.Infof("onNegotiationNeeded")
}

// /被叫不能在第一次的时候主动发起协议过程，主叫或者被叫不在第一次的时候可以发起协商过程
func (this *BasePeerConnection) negotiate() {
	if this.initiator {
		this._negotiateOffer()
	} else {
		this._negotiateAnswer()
	}
	timer := time.NewTimer(time.Duration(this.delayTimes) * time.Second)
	defer timer.Stop()
	if timer.C != nil {
		if this.status != PeerConnectionStatus_connected {
			logger.Sugar.Warnf("delayed $delayTimes second cannot connected, will be closed")
			this.Close()
			if this.reconnectTimes > 0 {
				this.reconnectTimes--
				this.negotiate()
			}
		}
	}
}

// /作为主叫，发起协商过程createOffer
func (this *BasePeerConnection) _negotiateOffer() {
	if this.status == PeerConnectionStatus_closed {
		logger.Sugar.Errorf("PeerConnectionStatus closed")
		return
	}
	if this.negotiateStatus == NegotiateStatus_negotiating {
		logger.Sugar.Errorf("PeerConnectionStatus already negotiating")
		return
	}
	logger.Sugar.Warnf("Start negotiate")
	this.negotiateStatus = NegotiateStatus_negotiating
	this._createOffer()
}

// /作为主叫，创建offer，设置到本地会话描述，并发送offer
func (this *BasePeerConnection) _createOffer() {
	if this.status == PeerConnectionStatus_closed {
		logger.Sugar.Errorf("PeerConnectionStatus closed")
		return
	}
	var localDescription = this.peerConnection.LocalDescription()
	if localDescription != nil {
		logger.Sugar.Warnf("LocalDescription sdp offer is exist:%v", localDescription.Type)
	}
	offerOptions := &webrtc.OfferOptions{
		OfferAnswerOptions: webrtc.OfferAnswerOptions{VoiceActivityDetection: true},
		ICERestart:         true,
	}
	offer, err :=
		this.peerConnection.CreateOffer(offerOptions)
	if err == nil {
		this.peerConnection.SetLocalDescription(offer)
		logger.Sugar.Infof("createOffer and setLocalDescription offer successfully")
		this._sendOffer(offer)
	}
}

// /作为主叫，调用外部方法发送offer
func (this *BasePeerConnection) _sendOffer(offer webrtc.SessionDescription) {
	if this.status == PeerConnectionStatus_closed {
		logger.Sugar.Errorf("PeerConnectionStatus closed")
		return
	}

	var sdp = this.peerConnection.LocalDescription()
	webrtcSignal := WebrtcSignal{SignalType: WebrtcSignalType_Sdp,
		Sdp: sdp, SignalExtension: this.extension}
	this.Emit(
		WebrtcEventType_signal, &WebrtcEvent{Data: webrtcSignal})

	logger.Sugar.Infof("end sendOffer")
}

// /外部在收到信号的时候调用
func (this *BasePeerConnection) onSignal(webrtcSignal *WebrtcSignal) {
	if this.initiator {
		this._onOfferSignal(webrtcSignal)
	} else {
		this._onAnswerSignal(webrtcSignal)
	}
}

// /作为主叫，从信号服务器传回来远程的webrtcSignal信息，从signalAction回调
func (this *BasePeerConnection) _onOfferSignal(webrtcSignal *WebrtcSignal) {
	if this.status == PeerConnectionStatus_closed {
		logger.Sugar.Errorf("PeerConnectionStatus closed")
		return
	}
	// 错误的信号
	if webrtcSignal.Sdp == nil && webrtcSignal.Candidates == nil && !webrtcSignal.Renegotiate && webrtcSignal.TransceiverRequest == nil {
		this.Close()
		return
	}
	signalType := webrtcSignal.SignalType
	candidates := webrtcSignal.Candidates
	sdp := webrtcSignal.Sdp
	//被要求重新协商，则发起协商
	if signalType == WebrtcSignalType_Renegotiate &&
		webrtcSignal.Renegotiate {
		logger.Sugar.Infof("onSignal renegotiate")
		this.negotiate()
	} else if webrtcSignal.TransceiverRequest != nil { //被要求收发，则加收发器
		logger.Sugar.Infof("onSignal transceiver")
		logger.Sugar.Infof("got request for transceiver")
		webrtcSignal.TransceiverRequest = make(map[string]interface{})
		kind, ok := webrtcSignal.TransceiverRequest["kind"]
		if ok {
			codecType := webrtc.NewRTPCodecType(kind.(string))
			init, ok := webrtcSignal.TransceiverRequest["init"]
			if ok {
				buf, err := message.Marshal(init)
				if err != nil {
					return
				}
				inits := make([]webrtc.RTPTransceiverInit, 0)
				err = message.Unmarshal(buf, &inits)
				if err != nil {
					return
				}
				//webrtc.NewRTPTransceiverDirection(direction.(string))

				this.addTransceiver(codecType, inits...)
			}
		}
	} else if signalType == WebrtcSignalType_Candidate && candidates != nil {
		//如果是候选信息
		//logger.Sugar.Infof("onSignal candidate:${candidate.candidate}")
		for _, candidate := range candidates {
			err := this.addIceCandidate(candidate)
			if err != nil {
				logger.Sugar.Errorf("addIceCandidate err:%v", err.Error())
			}
		}
	} else if signalType == WebrtcSignalType_Sdp && sdp != nil {
		//如果sdp信息，则设置远程描述
		//对主叫节点来说，sdp应该是answer
		if sdp.Type.String() != "answer" {
			logger.Sugar.Errorf("onSignal sdp type is not answer:${sdp.type}")
		}
		remoteDescription :=
			this.peerConnection.RemoteDescription()
		if remoteDescription != nil {
			logger.Sugar.Warnf("remoteDescription is exist")
		}

		err := this.peerConnection.SetRemoteDescription(*sdp)
		if err == nil {
			if this.status == PeerConnectionStatus_closed {
				logger.Sugar.Errorf("PeerConnectionStatus closed")
				return
			}
		} else {
			this.Close()
		}
	} else { //如果什么都不是，报错
		logger.Sugar.Errorf("signal called with invalid signal type")
	}
}

// /作为被叫，协商时发送再协商信号给主叫，要求重新发起协商
func (this *BasePeerConnection) _negotiateAnswer() {
	if this.status == PeerConnectionStatus_closed {
		logger.Sugar.Errorf("PeerConnectionStatus closed")
		return
	}
	if this.negotiateStatus == NegotiateStatus_negotiating {
		logger.Sugar.Errorf("already negotiating")
		return
	}
	if this.status != PeerConnectionStatus_connected {
		logger.Sugar.Errorf("answer renegotiate only connected")
		return
	}
	//被叫发送重新协商的请求
	logger.Sugar.Warnf("send signal renegotiate")
	webrtcSignal := WebrtcSignal{SignalType: WebrtcSignalType_Renegotiate,
		Renegotiate: true, SignalExtension: this.extension}
	this.Emit(
		WebrtcEventType_signal, &WebrtcEvent{Data: webrtcSignal})
	this.negotiateStatus = NegotiateStatus_negotiating
}

// /作为被叫，创建answer，发生在被叫方，将answer回到主叫方
func (this *BasePeerConnection) _createAnswer() {
	if this.status == PeerConnectionStatus_closed {
		logger.Sugar.Errorf("PeerConnectionStatus closed")
		return
	}

	logger.Sugar.Infof("start createAnswer")
	sdp := this.peerConnection.LocalDescription()
	if sdp != nil {
		logger.Sugar.Warnf("getLocalDescription local sdp answer is exist:${answer.type}")
	}
	answerOptions := &webrtc.AnswerOptions{}
	answer, err := this.peerConnection.CreateAnswer(answerOptions)
	if err == nil {
		logger.Sugar.Infof("create local sdp answer:${answer.type}, and setLocalDescription")

		this.peerConnection.SetLocalDescription(answer)
		logger.Sugar.Infof(
			"setLocalDescription local sdp answer:${answer.type} successfully")
		this._sendAnswer(&answer)
	}
}

// 作为被叫，发送answer
func (this *BasePeerConnection) _sendAnswer(answer *webrtc.SessionDescription) {
	if this.status == PeerConnectionStatus_closed {
		logger.Sugar.Errorf("PeerConnectionStatus closed")
		return
	}
	logger.Sugar.Infof("send signal local sdp answer:${answer.type}")
	var sdp = this.peerConnection.LocalDescription()
	if sdp == nil {
		sdp = answer
	}
	webrtcSignal := WebrtcSignal{SignalType: WebrtcSignalType_Sdp,
		Sdp: sdp, SignalExtension: this.extension}
	this.Emit(
		WebrtcEventType_signal, &WebrtcEvent{Data: webrtcSignal})
	logger.Sugar.Infof("sendAnswer:${answer.type} successfully")
}

// /作为被叫，从信号服务器传回来远程的webrtcSignal信息，从signalAction回调
func (this *BasePeerConnection) _onAnswerSignal(webrtcSignal *WebrtcSignal) {
	if this.status == PeerConnectionStatus_closed {
		logger.Sugar.Errorf("PeerConnectionStatus closed")
		return
	}
	// 错误的信号
	if webrtcSignal.Sdp == nil && webrtcSignal.Candidates == nil && !webrtcSignal.Renegotiate && webrtcSignal.TransceiverRequest == nil {
		this.Close()
		return
	}
	this.negotiateStatus = NegotiateStatus_negotiating
	signalType := webrtcSignal.SignalType
	candidates := webrtcSignal.Candidates
	sdp := webrtcSignal.Sdp
	//如果是候选信息
	if signalType == WebrtcSignalType_Candidate && candidates != nil {
		for _, candidate := range candidates {
			err := this.addIceCandidate(candidate)
			if err != nil {
				logger.Sugar.Errorf("")
			}
		}
	} else if signalType == WebrtcSignalType_Sdp && sdp != nil { //如果sdp信息，则设置远程描述
		if sdp.Type.String() != "offer" {
			logger.Sugar.Errorf("onSignal sdp is not offer:${sdp.type}")
		}
		logger.Sugar.Infof("start setRemoteDescription sdp offer:${sdp.type}")
		remoteDescription :=
			this.peerConnection.RemoteDescription()
		if remoteDescription != nil {
			logger.Sugar.Warnf(
				"RemoteDescription sdp offer is exist:${remoteDescription.type}")
		}
		err := this.peerConnection.SetRemoteDescription(*sdp)
		if err != nil {
			logger.Sugar.Errorf("")
		}
		logger.Sugar.Infof("setRemoteDescription sdp offer:${sdp.type} successfully")
		//如果远程描述是offer请求，则创建answer
		remoteDescription = this.peerConnection.RemoteDescription()
		if remoteDescription != nil && remoteDescription.Type.String() == "offer" {
			this._createAnswer()
		} else {
			logger.Sugar.Errorf("RemoteDescription sdp is not offer:${remoteDescription!.type}")
		}
	} else {
		logger.Sugar.Errorf("signal called with invalid signal data") //如果什么都不是，报错
	}
}

func (this *BasePeerConnection) requestMissingTransceivers() {
	transceivers := this.peerConnection.GetTransceivers()
	if len(transceivers) > 0 {
		for _, transceiver := range transceivers {
			sender := transceiver.Sender()
			mid := transceiver.Mid()
			logger.Sugar.Errorf("transceiver mid:%vsender:%v", mid, sender != nil)
			if sender != nil {
				track := sender.Track()
				if mid == "" && track != nil {
					kind := track.Kind()
					logger.Sugar.Errorf("addTransceiver: %v", kind)
					if kind != 0 {
						this.addTransceiver(kind)
					}
				}
			}
		}
	}
}

/*
*
关闭peer
*/
func (this *BasePeerConnection) Close() {
	if this.status == PeerConnectionStatus_closed {
		logger.Sugar.Errorf("PeerConnectionStatus closed")
		return
	}
	this.localTracks = nil
	//this.trackSenders = nil
	if this.dataChannel != nil {
		this.dataChannel.Close()
		this.dataChannelOpen = false
		// allow events concurrent with destruction to be handled
		this.dataChannel.OnMessage(nil)
		this.dataChannel.OnClose(nil)
		this.dataChannel.OnOpen(nil)
		this.dataChannel.OnError(nil)
		this.dataChannel = nil
	}
	if this.peerConnection != nil {
		this.peerConnection.Close()
		this.peerConnection = nil

		// allow events concurrent with destruction to be handled
		this.peerConnection.OnConnectionStateChange(nil)
		this.peerConnection.OnDataChannel(nil)
		this.peerConnection.OnICECandidate(nil)
		this.peerConnection.OnICEConnectionStateChange(nil)
		this.peerConnection.OnICEGatheringStateChange(nil)
		this.peerConnection.OnSignalingStateChange(nil)
		this.peerConnection.OnTrack(nil)
		this.peerConnection.OnNegotiationNeeded(nil)
	}
	this.status = PeerConnectionStatus_closed
	this.negotiateStatus = NegotiateStatus_none
	logger.Sugar.Infof("PeerConnectionStatus closed")

	// if (this.reconnectTimes > 0) {
	//   this.reconnect()
	// }
	this.Emit(
		WebrtcEventType_closed, &WebrtcEvent{Data: ""})
}

func (this *BasePeerConnection) addIceCandidate(candidate *webrtc.ICECandidateInit) error {
	err := this.peerConnection.AddICECandidate(*candidate)
	if err != nil {
		// 找出错误原因
		can := &webrtc.ICECandidate{}
		if can.Address != "" || strings.HasSuffix(can.Address, ".local") {
			logger.Sugar.Infof("Ignoring unsupported ICE candidate.")
		} else {
			this.Close()
		}

		return err
	}

	return nil
}

/**
 * Add a Transceiver to the connection.
 * @param {String} kind
 * @param {Object} init
 */
func (this *BasePeerConnection) addTransceiver(kind webrtc.RTPCodecType, init ...webrtc.RTPTransceiverInit) {
	if this.initiator {
		_, err := this.peerConnection.AddTransceiverFromKind(kind, init...)
		if err != nil {
			this.Close()
		}
		this.negotiate()
	} else {
		webrtcSignal := &WebrtcSignal{SignalType: WebrtcSignalType_TransceiverRequest}
		webrtcSignal.TransceiverRequest = make(map[string]interface{})
		webrtcSignal.TransceiverRequest["kind"] = kind
		webrtcSignal.TransceiverRequest["init"] = init
		this.Emit(
			WebrtcEventType_signal, &WebrtcEvent{Name: string(WebrtcEventType_signal), Data: webrtcSignal})
	}
}

/*
*
获取统计信息，设置连接状态，发送缓存的待发送数据
*/
func (this *BasePeerConnection) setStatsReport() {
	statsReports := this.peerConnection.GetStats()
	candidatePair := &webrtc.ICECandidatePair{}
	candidatePairStats, _ := statsReports.GetICECandidatePairStats(candidatePair)
	logger.Sugar.Infof("%v", candidatePairStats)

	candidate := &webrtc.ICECandidate{}
	statsReports.GetICECandidateStats(candidate)

	logger.Sugar.Infof(
		"connect local: %s:%s remote: %s:%s",
	)
}
