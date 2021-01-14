package simplepeer

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	webrtc2 "github.com/curltech/go-colla-node/webrtc"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"time"
)

/**
将本地轨道加入，并存放进流中
*/
func (this *SimplePeer) AddTrack(track webrtc.TrackLocal) (*webrtc.RTPSender, error) {
	// Add this newly created track to the PeerConnection
	sender, err := this.peerConnection.AddTrack(track)
	if err != nil {
		logger.Errorf(err.Error())

		return nil, err
	}
	this.needsNegotiation()

	return sender, nil
}

/**
读取发送者的数据
*/
func (this *SimplePeer) read(sender *webrtc.RTPSender) {
	// Read incoming RTCP packets
	// Before these packets are retuned they are processed by interceptors. For things
	// like NACK this needs to be called.
	rtcpBuf := make([]byte, 1500)
	for {
		if _, _, rtcpErr := sender.Read(rtcpBuf); rtcpErr != nil {
			return
		}
	}
}

/**
 * Replace a MediaStreamTrack by another in the connection.
 */
func (this *SimplePeer) replaceTrack(oldTrack webrtc.TrackLocal, newTrack webrtc.TrackLocal) {
	err := this.ReplaceTrack(oldTrack, newTrack)
	if err != nil {
		this.Destroy(errors.New(webrtc2.ERR_UNSUPPORTED_REPLACETRACK))
	}
}

/**
 * Remove a MediaStreamTrack from the connection.
 */
func (this *SimplePeer) removeTrack(track webrtc.TrackLocal) {
	err := this.RemoveTrack(track.ID())
	if err != nil {
		this.Destroy(errors.New(webrtc2.ERR_REMOVE_TRACK))
	}
	//this._sendersAwaitingStable.push(sender) // HACK: Firefox must wait until (signalingState === stable) https://bugzilla.mozilla.org/show_bug.cgi?id=1133874

	this.needsNegotiation()
}

/**
告诉对方peer加快发送
*/
func (this *SimplePeer) remb(track *webrtc.TrackRemote) {
	// Send a remb message with a very high bandwidth to trigger chrome to send also the high bitrate stream
	logger.Infof("Sending remb for stream with rid: %q, ssrc: %d\n", track.RID(), track.SSRC())
	if writeErr := this.peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.ReceiverEstimatedMaximumBitrate{Bitrate: 10000000, SenderSSRC: uint32(track.SSRC())}}); writeErr != nil {
		logger.Infof(writeErr.Error())
	}
}

/**
每三秒告诉对方关键帧
*/
func (this *SimplePeer) keyFrame(track *webrtc.TrackRemote) {
	ticker := time.NewTicker(time.Second * 3)
	for range ticker.C {
		errSend := this.peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})
		if errSend != nil {
			logger.Errorf(errSend.Error())
			ticker.Stop()
			break
		}
	}
}

/**
有新的媒体轨道连接上来
*/
func (this *SimplePeer) onTrack(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
	// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
	// This is a temporary fix until we implement incoming RTCP events, then we would push a PLI only when a viewer requests it
	go this.keyFrame(track)
	logger.Infof("Track has started, of type %d: %s \n", track.PayloadType(), track.Codec().MimeType)

	// track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver
	if this.destroyed {
		return
	}

	this.EmitEvent(webrtc2.EVENT_STREAM, &webrtc2.PeerEvent{Name: webrtc2.EVENT_STREAM, Data: track})
	this.EmitEvent(webrtc2.EVENT_TRACK, &webrtc2.PeerEvent{Name: webrtc2.EVENT_TRACK, Data: track})
}
