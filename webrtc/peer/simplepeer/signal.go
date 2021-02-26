package simplepeer

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/p2p/chain/action/dht"
	webrtc2 "github.com/curltech/go-colla-node/webrtc"
	"github.com/pion/webrtc/v3" //用到了webrtc的数据结构，但是更好的是不用，只将转换的方法做在webrtc里面
)

const (
	WebrtcSignalType_TransceiverRequest = "transceiverRequest"
	WebrtcSignalType_Renegotiate        = "renegotiate"
	WebrtcSignalType_Offer              = "offer"
	WebrtcSignalType_Answer             = "answer"
	WebrtcSignalType_Candidate          = "candidate"
	WebrtcSignalType_Join               = "join"
)

type Router struct {
	Id       string `json:"id,omitempty"`
	Typ      string `json:"type,omitempty"`
	Action   string `json:"action,omitempty"`
	RoomId   string `json:"roomId,omitempty"`
	Identity string `json:"identity,omitempty"`
}

/**
用于交换的signal消息，当type为join的时候，表示需要服务器反拨，发起offer
如果为offer，表示服务器为被叫方，如果router字段不为空，表示有sfu的要求
*/
type WebrtcSignal struct {
	SignalType         string                     `json:"type,omitempty"`
	Renegotiate        bool                       `json:"renegotiate,omitempty"`
	TransceiverRequest map[string]interface{}     `json:"transceiverRequest,omitempty"`
	Candidate          *webrtc.ICECandidateInit   `json:"candidate,omitempty"`
	Sdp                *webrtc.SessionDescription `json:"sdp,omitempty"`
	Router             *Router                    `json:"router,omitempty"`
}

/**
调用注册发送信号函数发送信号
*/
func Signal(webrtcSignal *WebrtcSignal, targetPeerId string) (interface{}, error) {
	sig := make(map[string]interface{})
	sig["type"] = webrtcSignal.SignalType
	if webrtcSignal.SignalType == WebrtcSignalType_Offer || webrtcSignal.SignalType == WebrtcSignalType_Answer {
		if webrtcSignal.Sdp != nil {
			sig["sdp"] = webrtcSignal.Sdp.SDP
		}
	} else if webrtcSignal.SignalType == WebrtcSignalType_Candidate {
		sig["candidate"] = webrtcSignal.Candidate
	} else if webrtcSignal.SignalType == WebrtcSignalType_Renegotiate {
		sig["renegotiate"] = webrtcSignal.Renegotiate
	} else if webrtcSignal.SignalType == WebrtcSignalType_TransceiverRequest {
		request := make(map[string]interface{})
		sig["transceiverRequest"] = request
		k, ok := webrtcSignal.TransceiverRequest["kind"]
		if ok {
			kind, ok := k.(webrtc.RTPCodecType)
			if ok {
				request["kind"] = kind.String()
			}
		}
		i, ok := webrtcSignal.TransceiverRequest["init"]
		if ok {
			init, ok := i.(string)
			if ok {
				request["kind"] = init
			}
		}
	}

	return dht.SignalAction.Signal("", sig, targetPeerId)
}

func Transform(payload map[string]interface{}) *WebrtcSignal {
	signalType := ""
	_type, ok := payload["type"]
	if ok {
		signalType, ok = _type.(string)
		if signalType == "" {
			return nil
		}
	}
	webrtcSignal := &WebrtcSignal{SignalType: signalType}
	if signalType == WebrtcSignalType_Offer || signalType == WebrtcSignalType_Answer {
		_sdp, ok := payload["sdp"]
		var webrtcSDP *webrtc.SessionDescription = nil
		if ok {
			sdp, ok := _sdp.(string)
			if ok && sdp != "" {
				webrtcSDP = &webrtc.SessionDescription{SDP: sdp}
				if signalType == WebrtcSignalType_Offer {
					webrtcSDP.Type = webrtc.SDPTypeOffer
				} else if signalType == WebrtcSignalType_Answer {
					webrtcSDP.Type = webrtc.SDPTypeAnswer
				}
				webrtcSignal.Sdp = webrtcSDP
			}
		}
	} else if signalType == WebrtcSignalType_Candidate {
		_candidate, ok := payload["candidate"]
		if ok {
			can, ok := _candidate.(map[string]interface{})
			if ok && can != nil {
				candidate := &webrtc.ICECandidateInit{}
				json, err := message.TextMarshal(can)
				if err == nil {
					err = message.TextUnmarshal(json, candidate)
					if err == nil {
						webrtcSignal.Candidate = candidate
					}
				}
			}
		}
	} else if signalType == WebrtcSignalType_Renegotiate {
		_renegotiate, ok := payload["renegotiate"]
		if ok {
			renegotiate, _ := _renegotiate.(bool)
			webrtcSignal.Renegotiate = renegotiate
		}
	} else if signalType == WebrtcSignalType_TransceiverRequest {
		_transceiverRequest, ok := payload["transceiverRequest"]
		if ok {
			transceiverRequest, ok := _transceiverRequest.(map[string]interface{})
			if ok {
				webrtcSignal.TransceiverRequest = transceiverRequest
			}
		}
	}
	r, ok := payload["router"]
	if ok {
		router := &Router{}
		json, err := message.TextMarshal(r)
		if err == nil {
			err = message.TextUnmarshal(json, router)
			if err == nil {
				webrtcSignal.Router = router
			}
		}
	}

	return webrtcSignal
}

/**
把外部接收的信号设置成设置本节点的信号
*/
func (this *SimplePeer) Signal(webrtcSignal *WebrtcSignal) error {
	if this.destroyed {
		logger.Sugar.Errorf("cannot signal after peer is destroyed'), 'ERRSIGNALING")
		return errors.New(webrtc2.ERR_SIGNALING)
	}
	/**
	如果是发起方并且需要重新协商
	*/
	if webrtcSignal.Renegotiate && this.initiator {
		logger.Sugar.Infof("got request to renegotiate")
		this.needsNegotiation()
	}
	/**
	如果是发起方并且需要增加收发器
	*/
	if webrtcSignal.TransceiverRequest != nil && this.initiator {
		logger.Sugar.Infof("got request for transceiver")
		webrtcSignal.TransceiverRequest = make(map[string]interface{})
		kind, ok := webrtcSignal.TransceiverRequest["kind"]
		if ok {
			codecType := webrtc.NewRTPCodecType(kind.(string))
			init, ok := webrtcSignal.TransceiverRequest["init"]
			if ok {
				buf, err := message.Marshal(init)
				if err != nil {
					return err
				}
				inits := make([]webrtc.RTPTransceiverInit, 0)
				err = message.Unmarshal(buf, &inits)
				if err != nil {
					return err
				}
				//webrtc.NewRTPTransceiverDirection(direction.(string))

				this.addTransceiver(codecType, inits...)
			}
		}
	}
	/**
	如果接收到candidate信号，是否设置了远程sdp，如果还没有设置，将candidate放入候选列表，等待sdp被设置后再设置（simplepeer实现）
	还有一种选择是直接加入，这种方式可能更快（pion实现）
	*/
	if webrtcSignal.Candidate != nil {
		if this.candidateFastAdd {
			err := this.addIceCandidate(webrtcSignal.Candidate)
			if err != nil {
				logger.Sugar.Errorf("addIceCandidate err:%v", err.Error())
			}
		} else {
			desc := this.peerConnection.RemoteDescription()
			if desc != nil && desc.Type != 0 {
				err := this.addIceCandidate(webrtcSignal.Candidate)
				if err != nil {
					logger.Sugar.Errorf("addIceCandidate err:%v", err.Error())
				}
			} else {
				this.candidatesMux.Lock()
				defer this.candidatesMux.Unlock()
				this.pendingCandidates = append(this.pendingCandidates, *webrtcSignal.Candidate)
			}
		}
	}
	/**
	接收到sdp信号，首先是设置远程sdp，如果成功设置，所以候选candidate都被加入，如果是被动方，创建answer
	*/
	if webrtcSignal.Sdp != nil {
		err := this.peerConnection.SetRemoteDescription(*webrtcSignal.Sdp)
		if err == nil {
			if this.destroyed {
				return nil
			}
			if !this.initiator && this.peerConnection.RemoteDescription().Type == webrtc.SDPTypeOffer {
				this.createAnswer()
			}
			this.candidatesMux.Lock()
			defer this.candidatesMux.Unlock()
			if this.candidateFastAdd {
				for _, candidate := range this.pendingCandidates {
					// 发送所有的候选candidate（pion的实现，设置远程sdp才一起发送，simplepeer是直接发送，在此不发送）
					this.signalCandidate(&candidate)
				}
				this.pendingCandidates = make([]webrtc.ICECandidateInit, 0)
			} else {
				for _, candidate := range this.pendingCandidates {
					// 加所有的候选candidate
					err := this.addIceCandidate(&candidate)
					if err != nil {
						logger.Sugar.Errorf("addIceCandidate err:%v", err.Error())
					}
				}
				this.pendingCandidates = make([]webrtc.ICECandidateInit, 0)
			}
		} else {
			this.Destroy(errors.New(webrtc2.ERR_SET_REMOTE_DESCRIPTION))
		}
		// 错误的信号
		if webrtcSignal.Sdp == nil && webrtcSignal.Candidate == nil && !webrtcSignal.Renegotiate && webrtcSignal.TransceiverRequest == nil {
			this.Destroy(errors.New(webrtc2.ERR_SIGNALING))
		}
	}

	return nil
}

/**
发送candidate信号
*/
func (this *SimplePeer) signalCandidate(candidate *webrtc.ICECandidateInit) (interface{}, error) {
	candidateSignal := &WebrtcSignal{SignalType: WebrtcSignalType_Candidate, Candidate: candidate}
	resp, err := this.EmitEvent(webrtc2.EVENT_SIGNAL, &webrtc2.PeerEvent{Data: candidateSignal})
	if err != nil {
		logger.Sugar.Errorf("%v", err.Error())
	}

	return resp, err
}
