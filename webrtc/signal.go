package webrtc

import (
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/p2p/chain/action/dht"
	"github.com/pion/webrtc/v3" //用到了webrtc的数据结构，但是更好的是不用，只将转换的方法做在webrtc里面
)

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
		sig["candidate"] = webrtcSignal.Candidates
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

///将信号消息转换成信号对象
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
		_candidates, ok := payload["candidates"]
		if ok {
			cans, ok := _candidates.([]map[string]interface{})
			if ok && cans != nil {
				for _, can := range cans {
					candidate := &webrtc.ICECandidateInit{}
					json, err := message.TextMarshal(can)
					if err == nil {
						err = message.TextUnmarshal(json, candidate)
						if err == nil {
							webrtcSignal.Candidates = append(webrtcSignal.Candidates, candidate)
						}
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
	r, ok := payload["extension"]
	if ok {
		extension := &SignalExtension{}
		json, err := message.TextMarshal(r)
		if err == nil {
			err = message.TextUnmarshal(json, extension)
			if err == nil {
				webrtcSignal.SignalExtension = extension
			}
		}
	}

	return webrtcSignal
}
