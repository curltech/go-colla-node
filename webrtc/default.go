package webrtc

import (
	"github.com/curltech/go-colla-core/config"
	"github.com/pion/webrtc/v3"
)

const (
	MAX_BUFFERED_AMOUNT     = 64 * 1024
	ICECOMPLETE_TIMEOUT     = 5 * 1000
	CHANNEL_CLOSING_TIMEOUT = 5 * 1000

	EVENT_CONNECT                = "connect"
	EVENT_SIGNAL                 = "signal"
	EVENT_ERROR                  = "error"
	EVENT_CLOSE                  = "close"
	EVENT_DATA                   = "data"
	EVENT_TRACK                  = "track"
	EVENT_STREAM                 = "stream"
	EVENT_ICE_TIMEOUT            = "iceTimeout"
	EVENT_ICE_COMPLETE           = "iceComplete"
	EVENT_ICE_STATE_CHANGE       = "iceStateChange"
	EVENT_NEGOTIATED             = "negotiated"
	EVENT_SIGNALING_STATE_CHANGE = "signalingStateChange"
	EVENT_FINISH                 = "finish"

	ERR_SET_REMOTE_DESCRIPTION   = "ERR_SET_REMOTE_DESCRIPTION"
	ERR_DATA_CHANNEL             = "ERR_DATA_CHANNEL"
	ERR_SIGNALING                = "ERRSIGNALING"
	ERR_CREATE_OFFER             = "ERR_CREATE_OFFER"
	ERR_SET_LOCAL_DESCRIPTION    = "ERR_SET_LOCAL_DESCRIPTION"
	ERR_CREATE_ANSWER            = "ERR_CREATE_ANSWER"
	ERR_ADD_ICE_CANDIDATE        = "ERR_ADD_ICE_CANDIDATE"
	ERR_CONNECTION_FAILURE       = "ERR_CONNECTION_FAILURE"
	ERR_ICE_CONNECTION_CLOSED    = "ERR_ICE_CONNECTION_CLOSED"
	ERR_ADD_TRANSCEIVER          = "ERR_ADD_TRANSCEIVER"
	ERR_SENDER_ALREADY_ADDED     = "ERR_SENDER_ALREADY_ADDED"
	ERR_TRACK_ALREADY_ADDED      = "ERR_TRACK_ALREADY_ADDED"
	ERR_SENDER_REMOVED           = "ERR_SENDER_REMOVED"
	ERR_TRACK_NOT_ADDED          = "ERR_TRACK_NOT_ADDED"
	ERR_UNSUPPORTED_REPLACETRACK = "ERR_UNSUPPORTED_REPLACETRACK"
	ERR_REMOVE_TRACK             = "ERR_REMOVE_TRACK"
)

type PeerEvent struct {
	Name string
	Data interface{}
}

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

type WebrtcOption struct {
	Initiator          bool
	ChannelConfig      *webrtc.DataChannelInit
	ChannelName        string
	Config             *webrtc.Configuration
	OfferOptions       *webrtc.OfferOptions
	AnswerOptions      *webrtc.AnswerOptions
	SdpTransform       func(sdp string) string
	Trickle            bool
	AllowHalfTrickle   bool
	ObjectMode         bool
	AllowHalfOpen      bool
	IceCompleteTimeout uint64
}
