package sip

import (
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"

	"github.com/curltech/go-colla-node/webrtc/sip/softphone"
	"github.com/pion/sdp/v2"
)

var (
	username, _  = config.GetString("sip.username", "1000")
	password, _  = config.GetString("sip.password", "")
	extension, _ = config.GetString("sip.extension", "9198")
	host, _      = config.GetString("sip.host", "")
	port, _      = config.GetString("sip.port", "5066")
)

func start() {
	if host == "" || port == "" || password == "" {
		logger.Errorf("-host -port and -password are required")

		return
	}

	conn := softphone.NewSoftPhone(softphone.SIPInfoResponse{
		Username:        username,
		AuthorizationID: username,
		Password:        password,
		Domain:          host,
		Transport:       "ws",
		OutboundProxy:   host + ":" + port,
	})

	conn.OnOK(func(okBody string) {
		answer := okBody + "a=mid:0\r\n"
		logger.Infof("%v", answer)
	})
	offer := ""
	conn.Invite(extension, rewriteSDP(offer))

	select {}
}

// Apply the following transformations for FreeSWITCH
// * Add fake srflx candidate to each media section
// * Add msid to each media section
// * Make bundle first attribute at session level.
func rewriteSDP(in string) string {
	parsed := &sdp.SessionDescription{}
	if err := parsed.Unmarshal([]byte(in)); err != nil {
		panic(err)
	}

	// Reverse global attributes
	for i, j := 0, len(parsed.Attributes)-1; i < j; i, j = i+1, j-1 {
		parsed.Attributes[i], parsed.Attributes[j] = parsed.Attributes[j], parsed.Attributes[i]
	}

	parsed.MediaDescriptions[0].Attributes = append(parsed.MediaDescriptions[0].Attributes, sdp.Attribute{
		Key:   "candidate",
		Value: "79019993 1 udp 1686052607 1.1.1.1 9 typ srflx",
	})

	out, err := parsed.Marshal()
	if err != nil {
		panic(err)
	}

	return string(out)
}
