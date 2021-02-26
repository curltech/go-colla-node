package sfu

import (
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/p2p/chain/action/dht"
	"github.com/pion/webrtc/v3"
)

const (
	SfuSignalType_Join    = "join"
	SfuSignalType_Offer   = "offer"
	SfuSignalType_Answer  = "answer"
	SfuSignalType_Trickle = "trickle"
)

type SfuSignal struct {
	//四种类型
	SignalType string `json:"type,omitempty"`
	// 房间号
	Sid string `json:"sid,omitempty"`
	// 房间的原始节点
	PrimaryPeerId string `json:"sid,omitempty"`
	// join,offer,answer
	Sdp *webrtc.SessionDescription `json:"sdp,omitempty"`
	// candidate(trickle) publisher:0 or subscriber:1
	Target int `json:"target,omitempty"`
	// candidate(trickle)
	Candidate *webrtc.ICECandidateInit `json:"candidate,omitempty"`
}

/**
调用注册发送信号函数发送信号
*/
func Signal(sfuSignal *SfuSignal, targetPeerId string) (interface{}, error) {
	sig := make(map[string]interface{})
	sig["type"] = sfuSignal.SignalType
	if sfuSignal.SignalType == SfuSignalType_Offer || sfuSignal.SignalType == SfuSignalType_Answer {
		if sfuSignal.Sdp != nil {
			sig["sdp"] = sfuSignal.Sdp.SDP
		}
	} else if sfuSignal.SignalType == SfuSignalType_Trickle {
		sig["candidate"] = sfuSignal.Candidate
	}

	return dht.IonSignalAction.Signal("", sig, targetPeerId)
}

func Transform(payload map[string]interface{}) *SfuSignal {
	signalType := ""
	_type, ok := payload["type"]
	if ok {
		signalType, ok = _type.(string)
		if ok && signalType == "" {
			return nil
		}
	}
	sfuSignal := &SfuSignal{SignalType: signalType}
	if signalType == SfuSignalType_Offer || signalType == SfuSignalType_Answer || signalType == SfuSignalType_Join {
		_sdp, ok := payload["sdp"]
		var SDP *webrtc.SessionDescription = nil
		if ok {
			sdp, ok := _sdp.(string)
			if ok && sdp != "" {
				SDP = &webrtc.SessionDescription{SDP: sdp}
				if signalType == SfuSignalType_Offer || signalType == SfuSignalType_Join {
					SDP.Type = webrtc.SDPTypeOffer
				} else if signalType == SfuSignalType_Answer {
					SDP.Type = webrtc.SDPTypeAnswer
				}
				sfuSignal.Sdp = SDP
			}
		}
		_sid, ok := payload["sid"]
		if ok {
			sid, ok := _sid.(string)
			if ok {
				sfuSignal.Sid = sid
			}
		}
		_primaryPeerId, ok := payload["primaryPeerId"]
		if ok {
			primaryPeerId, ok := _primaryPeerId.(string)
			if ok {
				sfuSignal.PrimaryPeerId = primaryPeerId
			}
		}
	} else if signalType == SfuSignalType_Trickle {
		_candidate, ok := payload["candidate"]
		if ok {
			can, ok := _candidate.(map[string]interface{})
			if ok && can != nil {
				candidate := &webrtc.ICECandidateInit{}
				json, err := message.TextMarshal(can)
				if err == nil {
					err = message.TextUnmarshal(json, candidate)
					if err == nil {
						sfuSignal.Candidate = candidate
					}
				}
			}
		}
		_target, ok := payload["target"]
		if ok {
			target, ok := _target.(int)
			if ok {
				sfuSignal.Target = target
			}
		}
	}

	return sfuSignal
}
