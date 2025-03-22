package webrtc

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
	"time"
)

/*
*
将本地轨道加入，并存放进流中
*/
func (this *BasePeerConnection) AddTrack(track webrtc.TrackLocal) (*webrtc.RTPSender, error) {
	// Add this newly created track to the PeerConnection
	streamId := track.StreamID()
	trackId := track.ID()
	sender, err := this.peerConnection.AddTrack(track)
	if err != nil {
		logger.Sugar.Errorf(err.Error())

		return nil, err
	}
	tracks, ok := this.localTracks[streamId]
	if !ok {
		tracks = make(map[string]webrtc.TrackLocal, 0)
		this.localTracks[streamId] = tracks
	}
	tracks[trackId] = track

	return sender, nil
}

/*
*
读取发送者的数据
*/
func (this *BasePeerConnection) read(sender *webrtc.RTPSender) {
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
func (this *BasePeerConnection) replaceTrack(oldTrack webrtc.TrackLocal, newTrack webrtc.TrackLocal) {
	err := this.ReplaceTrack(oldTrack, newTrack)
	if err != nil {
		this.Close()
	}
}

/**
 * Remove a MediaStreamTrack from the connection.
 */
func (this *BasePeerConnection) removeTrack(track webrtc.TrackLocal) {
	err := this.RemoveTrack(track.ID())
	if err != nil {
		this.Close()
	}
	//this._sendersAwaitingStable.push(sender) // HACK: Firefox must wait until (signalingState === stable) https://bugzilla.mozilla.org/show_bug.cgi?id=1133874

	this.negotiate()
}

/*
*
告诉对方peer加快发送
*/
func (this *BasePeerConnection) remb(track *webrtc.TrackRemote) {
	// Send a remb message with a very high bandwidth to trigger chrome to send also the high bitrate stream
	logger.Sugar.Infof("Sending remb for stream with rid: %q, ssrc: %d\n", track.RID(), track.SSRC())
	if writeErr := this.peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.ReceiverEstimatedMaximumBitrate{Bitrate: 10000000, SenderSSRC: uint32(track.SSRC())}}); writeErr != nil {
		logger.Sugar.Infof(writeErr.Error())
	}
}

/*
*
每三秒告诉对方关键帧
*/
func (this *BasePeerConnection) keyFrame(track *webrtc.TrackRemote) {
	ticker := time.NewTicker(time.Second * 3)
	for range ticker.C {
		errSend := this.peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})
		if errSend != nil {
			logger.Sugar.Errorf(errSend.Error())
			ticker.Stop()
			break
		}
	}
}

/*
*
有新的媒体轨道连接上来
*/
func (this *BasePeerConnection) onTrack(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
	if this.status == PeerConnectionStatus_closed {
		logger.Sugar.Errorf("PeerConnectionStatus closed")
		return
	}
	// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
	// This is a temporary fix until we implement incoming RTCP events, then we would push a PLI only when a viewer requests it
	go this.keyFrame(track)
	logger.Sugar.Infof("Track has started, of type %d: %s \n", track.PayloadType(), track.Codec().MimeType)

	// track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver

	this.Emit(WebrtcEventType_stream, &WebrtcEvent{Name: string(WebrtcEventType_stream), Data: track})
	this.Emit(WebrtcEventType_track, &WebrtcEvent{Name: string(WebrtcEventType_track), Data: track})
}
