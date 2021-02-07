package simplepeer

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/security"
	webrtc2 "github.com/curltech/go-colla-node/webrtc"
	"github.com/pion/webrtc/v3"
	"strings"
	"sync"
	"time"
)

type SimplePeer struct {
	id                    string
	channelName           string
	initiator             bool
	channelConfig         *webrtc.DataChannelInit
	config                *webrtc.Configuration
	offerOptions          *webrtc.OfferOptions
	answerOptions         *webrtc.AnswerOptions
	trickle               bool
	allowHalfTrickle      bool
	iceCompleteTimeout    uint64
	destroyed             bool
	destroying            bool
	connected             bool
	connecting            bool
	remoteAddress         string
	remoteFamily          string
	remotePort            string
	localAddress          string
	localFamily           string
	localPort             string
	pcReady               bool
	channelReady          bool
	iceComplete           bool        // ice candidate trickle done (got null candidate)
	iceCompleteTimer      *time.Timer // send an offer/answer anyway after some timeout
	dataChannel           *webrtc.DataChannel
	pendingCandidates     []webrtc.ICECandidateInit
	candidatesMux         sync.Mutex
	candidateFastAdd      bool // true采用pion模式
	isNegotiating         bool // is this peer waiting for negotiation to complete?
	firstNegotiation      bool
	batchedNegotiation    bool // batch synchronous negotiations
	queuedNegotiation     bool // is there a queued negotiation request?
	sendersAwaitingStable []webrtc.RTPSender
	//senderMap             map[string]interface{}
	closingInterval   uint64
	interval          uint64
	peerConnection    *webrtc.PeerConnection
	channelNegotiated *bool
	onFinishBound     func()
	events            map[string]func(event *webrtc2.PeerEvent) (interface{}, error)
	onces             map[string]func() //注册事件完成后需要调用的方法，调用完成后会清掉
	// datachannel的收发池
	sendChan chan []byte
}

/**
注册事件，发生时调用外部函数
*/
func (this *SimplePeer) RegistEvent(name string, fn func(event *webrtc2.PeerEvent) (interface{}, error)) {
	if this.events == nil {
		this.events = make(map[string]func(event *webrtc2.PeerEvent) (interface{}, error), 0)
	}
	this.events[name] = fn
}

func (this *SimplePeer) UnregistEvent(name string) bool {
	if this.events == nil {
		return false
	}
	delete(this.events, name)
	return true
}

/**
事件发生，调用外部函数
*/
func (this *SimplePeer) EmitEvent(name string, event *webrtc2.PeerEvent) (interface{}, error) {
	if this.events == nil {
		return nil, errors.New("EventNotExist")
	}
	if this.onces != nil {
		fn, ok := this.onces[name]
		if ok {
			delete(this.onces, name)
			fn()
		}
	}
	fn, ok := this.events[name]
	if ok {
		return fn(event)
	}
	return nil, errors.New("EventNotExist")
}

/**
创建媒体引擎和API对象，在需要自定义音视频编码的时候使用
*/
func (this *SimplePeer) createApi() (*webrtc.API, error) {
	// Create a MediaEngine object to configure the supported codec
	mediaEngine := webrtc.MediaEngine{}

	// Setup the codecs you want to use.
	// We'll use a VP8 and Opus but you can also define your own
	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "video/VP8", ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        96,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		return nil, err
	}

	// Create the API object with the MediaEngine
	api := webrtc.NewAPI(webrtc.WithMediaEngine(&mediaEngine))

	return api, nil
}

func (this *SimplePeer) Create(opts *webrtc2.WebrtcOption) error {
	this.id = security.UUID()
	logger.Sugar.Infof("new peer %v", opts)

	if opts.Initiator {
		this.channelName = opts.ChannelName
		if this.channelName == "" {
			this.channelName = security.UUID()
		}
	}
	this.initiator = opts.Initiator
	this.channelConfig = opts.ChannelConfig
	if this.channelConfig != nil {
		this.channelNegotiated = this.channelConfig.Negotiated
	}
	this.config = opts.Config
	this.offerOptions = opts.OfferOptions
	if this.offerOptions == nil {
		this.offerOptions = &webrtc.OfferOptions{ICERestart: true}
	}
	this.answerOptions = opts.AnswerOptions
	this.trickle = true
	this.allowHalfTrickle = opts.AllowHalfTrickle
	if opts.IceCompleteTimeout == 0 {
		this.iceCompleteTimeout = webrtc2.ICECOMPLETE_TIMEOUT
	} else {
		this.iceCompleteTimeout = opts.IceCompleteTimeout
	}
	this.pendingCandidates = make([]webrtc.ICECandidateInit, 0)
	this.candidateFastAdd = true
	this.sendersAwaitingStable = make([]webrtc.RTPSender, 0)
	this.firstNegotiation = true
	this.config = opts.Config

	// 1.Create a new RTCPeerConnection
	var err error
	// 使用缺省的音视频编码
	this.peerConnection, err = webrtc.NewPeerConnection(*this.config)
	if err != nil {
		logger.Sugar.Errorf("%v", err)
		return err
	}

	// 2. regist peerConnection event
	this.peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		this.onIceConnectionStateChange(state)
	})

	this.peerConnection.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
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
	if this.initiator || (this.channelNegotiated != nil && *this.channelNegotiated) {
		this.dataChannel, _ = this.peerConnection.CreateDataChannel(this.channelName, this.channelConfig)
		this.setupDataChannel(this.dataChannel)
	} else {
		this.peerConnection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
			this.setupDataChannel(dataChannel)
		})
	}

	// 4. add audio/video stream
	//if this.streams != nil && len(this.streams) > 0 {
	//	for _, stream := range this.streams {
	//		this.addStream(stream)
	//	}
	//}
	this.peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		this.onTrack(track, receiver)
	})

	// 5. Negotiation, if need, create Offer
	this.needsNegotiation()

	// 6. finish stream
	//this.onFinishBound = func() {
	//	this.onFinish()
	//}
	//this.once("finish", this.onFinishBound)

	return nil
}

func (this *SimplePeer) onIceConnectionStateChange(state webrtc.ICEConnectionState) {
	this.onIceStateChange()
}

func (this *SimplePeer) onIceGatheringStateChange(state webrtc.ICEGathererState) {
	this.onIceStateChange()
}

/**
ICE状态改变事件
*/
func (this *SimplePeer) onIceStateChange() {
	if this.destroyed {
		return
	}
	var iceConnectionState = this.peerConnection.ICEConnectionState()
	var iceGatheringState = this.peerConnection.ICEGatheringState()

	logger.Sugar.Infof(
		"iceStateChange (connection: %s) (gathering: %s)",
		iceConnectionState,
		iceGatheringState,
	)

	this.EmitEvent(webrtc2.EVENT_ICE_STATE_CHANGE, &webrtc2.PeerEvent{Data: iceConnectionState})
	this.EmitEvent(webrtc2.EVENT_ICE_STATE_CHANGE, &webrtc2.PeerEvent{Data: iceGatheringState})

	if iceConnectionState == webrtc.ICEConnectionStateConnected || iceConnectionState == webrtc.ICEConnectionStateCompleted {
		this.pcReady = true
		if !this.connected && !this.destroyed {
			this.connecting = true
		}
	}
	if iceConnectionState == webrtc.ICEConnectionStateFailed {
		this.Destroy(errors.New("ERR_ICE_CONNECTION_FAILURE"))
	}
	if iceConnectionState == webrtc.ICEConnectionStateClosed {
		this.Destroy(errors.New("ERR_ICE_CONNECTION_CLOSED"))
	}
}

func (this *SimplePeer) onConnectionStateChange(state webrtc.PeerConnectionState) {
	if this.destroyed {
		return
	}
	if this.peerConnection.ConnectionState() == webrtc.PeerConnectionStateFailed {
		this.Destroy(errors.New("ERR_CONNECTION_FAILURE"))
	}
}

func (this *SimplePeer) onSignalingStateChange(state webrtc.SignalingState) {
	if this.destroyed {
		return
	}

	if this.peerConnection.SignalingState() == webrtc.SignalingStateStable {
		this.isNegotiating = false

		// HACK: Firefox doesn't yet support removing tracks when signalingState !== 'stable'
		logger.Sugar.Infof("flushing sender queue", this.sendersAwaitingStable)
		for _, sender := range this.sendersAwaitingStable {
			this.peerConnection.RemoveTrack(&sender)
			this.queuedNegotiation = true
		}
		this.sendersAwaitingStable = make([]webrtc.RTPSender, 0)

		if this.queuedNegotiation {
			logger.Sugar.Infof("flushing negotiation queue")
			this.queuedNegotiation = false
			this.needsNegotiation() // negotiate again
		} else {
			logger.Sugar.Infof("negotiated")
			this.EmitEvent(webrtc2.EVENT_NEGOTIATED, nil)
		}
	}

	logger.Sugar.Infof("signalingStateChange %s", this.peerConnection.SignalingState())
	this.EmitEvent(webrtc2.EVENT_SIGNALING_STATE_CHANGE, &webrtc2.PeerEvent{Data: this.peerConnection.SignalingState()})
}

/**
对candidate的处理，pion建议的和simplepeer的实现有差异，体现在加入时间，发送时间，
pion的发送时间是在远程sdp被设置以后，之前一律候选，在收到sdp后统一发送，加入时间是在收到对方的candidate时马上加入，应该比较快
simplepeer的发送时间是onIceCandidate就发送，加入时间是在收到candidate或者sdp信号，sdp被设置后加入，包括候选一起加入，如果sdp未设置，一律候选
*/
func (this *SimplePeer) onIceCandidate(candidate *webrtc.ICECandidate) {
	if this.destroyed {
		return
	}

	this.candidatesMux.Lock()
	defer this.candidatesMux.Unlock()

	/**
	是否设置了远程sdp，如果还没有设置，将candidate放入候选列表，等待sdp被设置后再设置（simplepeer实现）
	另一个问题发送candidate的时机，一种是任何时候都发送（simplepeer），还有一种是设置了sdp才发送
	*/
	if this.candidateFastAdd {
		desc := this.peerConnection.RemoteDescription()
		if candidate != nil {
			can := candidate.ToJSON()
			if desc == nil {
				this.pendingCandidates = append(this.pendingCandidates, can)
			} else {
				this.signalCandidate(&can)
			}
		}
	} else {
		// 发送candidate
		if candidate != nil && this.trickle {
			can := candidate.ToJSON()
			this.signalCandidate(&can)
		} else if candidate == nil && this.iceComplete == false {
			/**
			传入的candidate为空，表示完成ice
			*/
			this.iceComplete = true
			this.EmitEvent(webrtc2.EVENT_ICE_COMPLETE, nil)
		}
		// as soon as we've received one valid candidate start timeout
		if candidate != nil {
			go this.startIceCompleteTimeout()
		}
	}
}

/**
对ice过程计时，要么完成，要么超时
*/
func (this *SimplePeer) startIceCompleteTimeout() {
	if this.destroyed {
		return
	}
	if this.iceCompleteTimer != nil {
		return
	}
	logger.Sugar.Infof("started iceComplete timeout")
	this.iceCompleteTimer = time.NewTimer(time.Duration(this.iceCompleteTimeout) * time.Second)
	defer this.iceCompleteTimer.Stop()
	/**
	超时发生
	*/
	if this.iceCompleteTimer.C != nil {
		// 没完成，设置成完成，触发完成和超时事件
		if !this.iceComplete {
			this.iceComplete = true
			logger.Sugar.Infof("iceComplete timeout completed")
			this.EmitEvent(webrtc2.EVENT_ICE_TIMEOUT, nil)
			this.EmitEvent(webrtc2.EVENT_ICE_COMPLETE, nil)
		}
	}
}

func (this *SimplePeer) address() (string, string, string) {
	return this.localPort, this.localFamily, this.localAddress
}

/**
需要协商
*/
func (this *SimplePeer) needsNegotiation() {
	if this.batchedNegotiation {
		return // batch synchronous renegotiations
	}
	this.batchedNegotiation = true
	//异步协商
	go func() {
		this.batchedNegotiation = false
		if this.initiator || !this.firstNegotiation {
			logger.Sugar.Infof("starting batched negotiation")
			this.negotiate()
		} else {
			logger.Sugar.Infof("non-initiator initial negotiation request discarded")
		}
		this.firstNegotiation = false
	}()
}

/**
开始协商
*/
func (this *SimplePeer) negotiate() {
	//主动方，如果正在协商，这只标志，否则创建offer
	if this.initiator {
		if this.isNegotiating {
			this.queuedNegotiation = true
			logger.Sugar.Infof("already negotiating, queueing")
		} else {
			logger.Sugar.Infof("start negotiation")
			this.createOffer() // HACK: Chrome crashes if we immediately call createOffer
		}
	} else { //如果被动方，如果正在协商，标志，否则发送协商信号
		if this.isNegotiating {
			this.queuedNegotiation = true
			logger.Sugar.Infof("already negotiating, queueing")
		} else {
			logger.Sugar.Infof("requesting negotiation from initiator")
			var webrtcSignal = &WebrtcSignal{}
			webrtcSignal.SignalType = WebrtcSignalType_Renegotiate
			webrtcSignal.Renegotiate = true
			this.EmitEvent(webrtc2.EVENT_SIGNAL, &webrtc2.PeerEvent{Data: webrtcSignal}) // request initiator to renegotiate
		}
	}
	//设置协商标志
	this.isNegotiating = true
}

/**
主动方创建offer
*/
func (this *SimplePeer) createOffer() {
	if this.destroyed {
		return
	}

	//&webrtc.OfferOptions{ICERestart: true}
	offer, err := this.peerConnection.CreateOffer(this.offerOptions)
	if err != nil {
		this.Destroy(errors.New(webrtc2.ERR_CREATE_OFFER))
	}
	if this.destroyed {
		return
	}
	var sendOffer = func() {
		if this.destroyed {
			return
		}
		var local *webrtc.SessionDescription = this.peerConnection.LocalDescription()
		if local == nil {
			local = &offer
		}
		webrtcSignal := &WebrtcSignal{SignalType: WebrtcSignalType_Offer, Sdp: local}
		_, err = this.EmitEvent("signal", &webrtc2.PeerEvent{Data: webrtcSignal})
		if err != nil {
			logger.Sugar.Errorf("%v", err)
			return
		}
	}
	err = this.peerConnection.SetLocalDescription(offer)
	if err != nil {
		this.Destroy(errors.New(webrtc2.ERR_SET_LOCAL_DESCRIPTION))
	} else {
		logger.Sugar.Infof("createOffer success")
		if this.destroyed {
			return
		}
		if this.trickle || this.iceComplete {
			sendOffer()
		} else {
			// 事件完成后调用方法，相当于注册方法
			this.once(webrtc2.EVENT_ICE_COMPLETE, sendOffer) // wait for candidates
		}
	}
}

func (this *SimplePeer) requestMissingTransceivers() {
	transceivers := this.peerConnection.GetTransceivers()
	if len(transceivers) > 0 {
		for _, transceiver := range transceivers {
			sender := transceiver.Sender()
			mid := transceiver.Mid()
			logger.Sugar.Errorf("transceiver mid:%v;sender:%v", mid, sender != nil)
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

func (this *SimplePeer) onFinish() {
	if this.destroyed {
		return
	}

	// Wait a bit before destroying so the socket flushes.
	// TODO: is there a more reliable way to accomplish this?
	var destroySoon = func() {
		timer := time.NewTimer(time.Duration(1) * time.Second)
		defer timer.Stop()
		if timer.C != nil {
			this.Destroy(nil)
		}
	}

	if this.connected {
		destroySoon()
	} else {
		this.once(webrtc2.EVENT_CONNECT, destroySoon)
	}
}

// 等待事件完成后调用一次方法
func (this *SimplePeer) once(event string, fn func()) {
	if this.onces == nil {
		this.onces = make(map[string]func(), 0)
	}
	this.onces[event] = fn
}

func (this *SimplePeer) createAnswer() {
	if this.destroyed {
		return
	}

	answer, err := this.peerConnection.CreateAnswer(this.answerOptions)
	if err != nil {
		this.Destroy(errors.New(webrtc2.ERR_CREATE_ANSWER))
		return
	}

	sendAnswer := func() {
		if this.destroyed {
			return
		}
		local := this.peerConnection.LocalDescription()
		if local == nil {
			local = &answer
		}
		webrtcSignal := &WebrtcSignal{SignalType: WebrtcSignalType_Answer, Sdp: local}
		_, err = this.EmitEvent("signal", &webrtc2.PeerEvent{Data: webrtcSignal})
		if err != nil {
			logger.Sugar.Errorf("%v", err)
			return
		}

		if !this.initiator {
			//this.requestMissingTransceivers()
		}
	}

	err = this.peerConnection.SetLocalDescription(answer)
	if err != nil {
		this.Destroy(errors.New(webrtc2.ERR_SET_LOCAL_DESCRIPTION))
		return
	} else {
		if this.destroyed {
			return
		}
		if this.trickle || this.iceComplete {
			sendAnswer()
		} else {
			this.once("_iceComplete", sendAnswer)
		}
	}
}

/**
关闭peer
*/
func (this *SimplePeer) Destroy(err error) {
	if this.destroyed || this.destroying {
		return
	}
	this.destroying = true
	if err != nil {
		logger.Sugar.Errorf("destroying (error: %s)", err.Error())
	} else {
		logger.Sugar.Errorf("destroying no error")
	}
	//this.readable = this.writable = false
	//
	//if (!this._readableState.ended) this.push(null)
	//if (!this._writableState.finished) this.end()

	this.connected = false
	this.pcReady = false
	this.channelReady = false

	//this.clearInterval(this.closingInterval)
	this.closingInterval = 0

	//clearInterval(this.interval)
	this.interval = 0

	if this.onFinishBound != nil {
		this.UnregistEvent("finish")
	}
	this.onFinishBound = nil

	if this.dataChannel != nil {
		this.dataChannel.Close()

		// allow events concurrent with destruction to be handled
		this.dataChannel.OnMessage(nil)
		this.dataChannel.OnOpen(nil)
		this.dataChannel.OnClose(nil)
		this.dataChannel.OnError(nil)
	}
	if this.peerConnection != nil {
		this.peerConnection.Close()

		// allow events concurrent with destruction to be handled
		this.peerConnection.OnICEConnectionStateChange(nil)
		this.peerConnection.OnICEGatheringStateChange(nil)
		this.peerConnection.OnSignalingStateChange(nil)
		this.peerConnection.OnICECandidate(nil)
		this.peerConnection.OnTrack(nil)
		this.peerConnection.OnDataChannel(nil)
	}
	//this.peerConnection = nil
	//this.dataChannel = nil
	if err != nil {
		this.EmitEvent(webrtc2.EVENT_ERROR, &webrtc2.PeerEvent{Data: err})
	}
	this.EmitEvent(webrtc2.EVENT_CLOSE, &webrtc2.PeerEvent{})

	this.destroying = false
	this.destroyed = true
	if err != nil {
		logger.Sugar.Errorf("destroy (error: %s)", err.Error())
	} else {
		logger.Sugar.Errorf("destroy no error")
	}
}

func (this *SimplePeer) addIceCandidate(candidate *webrtc.ICECandidateInit) error {
	err := this.peerConnection.AddICECandidate(*candidate)
	if err != nil {
		// 找出错误原因
		can := &webrtc.ICECandidate{}
		if can.Address != "" || strings.HasSuffix(can.Address, ".local") {
			logger.Sugar.Infof("Ignoring unsupported ICE candidate.")
		} else {
			this.Destroy(errors.New(webrtc2.ERR_ADD_ICE_CANDIDATE))
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
func (this *SimplePeer) addTransceiver(kind webrtc.RTPCodecType, init ...webrtc.RTPTransceiverInit) {
	if this.initiator {
		_, err := this.peerConnection.AddTransceiverFromKind(kind, init...)
		if err != nil {
			this.Destroy(errors.New(webrtc2.ERR_ADD_TRANSCEIVER))
		}
		this.needsNegotiation()
	} else {
		webrtcSignal := &WebrtcSignal{SignalType: WebrtcSignalType_TransceiverRequest}
		webrtcSignal.TransceiverRequest = make(map[string]interface{})
		webrtcSignal.TransceiverRequest["kind"] = kind
		webrtcSignal.TransceiverRequest["init"] = init

		this.EmitEvent(webrtc2.EVENT_SIGNAL, &webrtc2.PeerEvent{Name: webrtc2.EVENT_SIGNAL, Data: webrtcSignal})
	}
}

/**
获取统计信息，设置连接状态，发送缓存的待发送数据
*/
func (this *SimplePeer) setStatsReport() {
	statsReports := this.peerConnection.GetStats()
	candidatePair := &webrtc.ICECandidatePair{}
	candidatePairStats, _ := statsReports.GetICECandidatePairStats(candidatePair)
	logger.Sugar.Infof("%v", candidatePairStats)

	candidate := &webrtc.ICECandidate{}
	statsReports.GetICECandidateStats(candidate)

	logger.Sugar.Infof(
		"connect local: %s:%s remote: %s:%s",
		this.localAddress,
		this.localPort,
		this.remoteAddress,
		this.remotePort,
	)
}

func (this *SimplePeer) onNegotiationNeeded() {
	logger.Sugar.Infof("onNegotiationNeeded")
}
