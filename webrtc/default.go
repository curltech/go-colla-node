package webrtc

import (
	"github.com/curltech/go-colla-core/config"
	"github.com/pion/webrtc/v4"
)

type SignalType string

type NegotiateStatus string

const (
	NegotiateStatus_none        NegotiateStatus = "none"
	NegotiateStatus_negotiating NegotiateStatus = "negotiating" //协商过程中
	NegotiateStatus_negotiated  NegotiateStatus = "negotiated"
)

type PeerConnectionStatus string

const (
	PeerConnectionStatus_none         PeerConnectionStatus = "none"
	PeerConnectionStatus_created      PeerConnectionStatus = "created"
	PeerConnectionStatus_init         PeerConnectionStatus = "init"
	PeerConnectionStatus_reconnecting PeerConnectionStatus = "reconnecting"
	PeerConnectionStatus_failed       PeerConnectionStatus = "failed"
	PeerConnectionStatus_connected    PeerConnectionStatus = "connected" //连接是否完全建立，即协商过程结束
	PeerConnectionStatus_closed       PeerConnectionStatus = "closed"    //是否关闭连接完成
)

// /可以注册的事件
type WebrtcEventType string

const (
	WebrtcEventType_created            WebrtcEventType = "created"  //创建被叫连接
	WebrtcEventType_signal             WebrtcEventType = "signal"   //发送信号
	WebrtcEventType_onSignal           WebrtcEventType = "onSignal" //接收到信号
	WebrtcEventType_connected          WebrtcEventType = "connected"
	WebrtcEventType_closed             WebrtcEventType = "closed"
	WebrtcEventType_status             WebrtcEventType = "status"  //状态发生变化
	WebrtcEventType_message            WebrtcEventType = "message" //接收到消息
	WebrtcEventType_stream             WebrtcEventType = "stream"
	WebrtcEventType_removeStream       WebrtcEventType = "removeStream"
	WebrtcEventType_track              WebrtcEventType = "track"
	WebrtcEventType_addTrack           WebrtcEventType = "addTrack"
	WebrtcEventType_removeTrack        WebrtcEventType = "removeTrack"
	WebrtcEventType_error              WebrtcEventType = "error"
	WebrtcEventType_iceCandidate       WebrtcEventType = "iceCandidate"
	WebrtcEventType_connectionState    WebrtcEventType = "connectionState"
	WebrtcEventType_iceConnectionState WebrtcEventType = "iceConnectionState"
	WebrtcEventType_iceGatheringState  WebrtcEventType = "iceGatheringState"
	WebrtcEventType_signalingState     WebrtcEventType = "signalingState"
	WebrtcEventType_iceCompleted       WebrtcEventType = "iceCompleted"
	WebrtcEventType_dataChannelState   WebrtcEventType = "dataChannelState"
)

const (
	MAX_BUFFERED_AMOUNT     = 64 * 1024
	ICECOMPLETE_TIMEOUT     = 5 * 1000
	CHANNEL_CLOSING_TIMEOUT = 5 * 1000
)

const (
	WebrtcSignalType_TransceiverRequest = "transceiverRequest"
	WebrtcSignalType_Renegotiate        = "renegotiate"
	WebrtcSignalType_Sdp                = "sdp"
	WebrtcSignalType_Offer              = "offer"
	WebrtcSignalType_Answer             = "answer"
	WebrtcSignalType_Candidate          = "candidate"
	WebrtcSignalType_Join               = "join"
)

type WebrtcEvent struct {
	PeerId           string      `json:"peerId,omitempty"`
	ClientId         string      `json:"clientId,omitempty"`
	Name             string      `json:"name,omitempty"`
	ConnectPeerId    string      `json:"connectPeerId,omitempty"`
	ConnectSessionId string      `json:"connectSessionId,omitempty"`
	Data             interface{} `json:"data,omitempty"`
}

type SignalExtension struct {
	PeerId     string             `json:"peerId,omitempty"`
	ClientId   string             `json:"clientId,omitempty"`
	Name       string             `json:"name,omitempty"`
	Room       *MeetingRoom       `json:"room,omitempty"`
	IceServers []webrtc.ICEServer `json:"iceServers,omitempty"`
}

type MeetingRoom struct {
	Id       string `json:"id,omitempty"`
	Typ      string `json:"type,omitempty"`
	Action   string `json:"action,omitempty"`
	RoomId   string `json:"roomId,omitempty"`
	Identity string `json:"identity,omitempty"`
}

/*
*
用于交换的signal消息，当type为join的时候，表示需要服务器反拨，发起offer
如果为offer，表示服务器为被叫方，如果router字段不为空，表示有sfu的要求
*/
type WebrtcSignal struct {
	SignalType         string                     `json:"type,omitempty"`
	Renegotiate        bool                       `json:"renegotiate,omitempty"`
	TransceiverRequest map[string]interface{}     `json:"transceiverRequest,omitempty"`
	Candidates         []*webrtc.ICECandidateInit `json:"candidates,omitempty"`
	Sdp                *webrtc.SessionDescription `json:"sdp,omitempty"`
	SignalExtension    *SignalExtension           `json:"extension,omitempty"`
}

const unknownClientId = "unknownClientId"
const unknownName = "unknownName"

func GetICEServers() []webrtc.ICEServer {
	ICEServers := make([]webrtc.ICEServer, 0)
	for i, urls := range config.SfuParams.Urls {
		iceServer := webrtc.ICEServer{}
		iceServer.URLs = urls
		if i < len(config.SfuParams.Usernames) {
			username := config.SfuParams.Usernames[i]
			if username != "" {
				iceServer.Username = username
			}
		}
		if i < len(config.SfuParams.Credentials) {
			credential := config.SfuParams.Credentials[i]
			if credential != "" {
				iceServer.Credential = credential
			}
		}
		ICEServers = append(ICEServers, iceServer)
	}

	return ICEServers
}
