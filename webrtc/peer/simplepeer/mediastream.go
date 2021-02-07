package simplepeer

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/security"
	webrtc2 "github.com/curltech/go-colla-node/webrtc"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"io"
)

/**
 * 主动创建本地track能够发送和接收媒体信息
 */
func (this *SimplePeer) CreateTrack(c *webrtc.RTPCodecCapability, trackId string, streamId string) (webrtc.TrackLocal, error) {
	// Create Track that we send video back to browser on
	// webrtc.RTPCodecCapability{MimeType: "video/vp8"}
	var track webrtc.TrackLocal
	//  TrackLocalStaticSample
	// track, err := webrtc.NewTrackLocalStaticSample(*c, trackId, streamId)
	if streamId == "" {
		streamId = security.UUID()
	}
	if trackId == "" {
		trackId = security.UUID()
	}
	track, err := webrtc.NewTrackLocalStaticRTP(*c, trackId, streamId)
	if err != nil {
		logger.Sugar.Errorf(err.Error())

		return nil, err
	}

	return track, nil
}

func (this *SimplePeer) ReplaceTrack(oldTrack webrtc.TrackLocal, newTrack webrtc.TrackLocal) error {
	sender := this.GetSender(oldTrack.ID())
	if sender != nil {
		err := sender.ReplaceTrack(newTrack)
		return err
	} else {
		logger.Sugar.Errorf(webrtc2.ERR_TRACK_NOT_ADDED)
	}

	return nil
}

func (this *SimplePeer) RemoveTrack(trackId string) error {
	senders := this.peerConnection.GetSenders()
	if senders != nil && len(senders) > 0 {
		for _, sender := range senders {
			trackLocal := sender.Track()
			if trackLocal.ID() == trackId {
				return this.peerConnection.RemoveTrack(sender)
			}
		}
	}

	return errors.New("NotExist")
}

func (this *SimplePeer) RemoveStream(streamId string) error {
	senders := this.peerConnection.GetSenders()
	var err error
	if senders != nil && len(senders) > 0 {
		for _, sender := range senders {
			trackLocal := sender.Track()
			if trackLocal != nil {
				if streamId == "" || trackLocal.StreamID() == streamId {
					err1 := this.peerConnection.RemoveTrack(sender)
					if err1 != nil {
						err = err1
					}
				}
			}
		}
	}

	return err
}

func (this *SimplePeer) GetSender(trackId string) *webrtc.RTPSender {
	senders := this.peerConnection.GetSenders()
	if senders != nil && len(senders) > 0 {
		for _, sender := range senders {
			trackLocal := sender.Track()
			if trackLocal.ID() == trackId {
				return sender
			}
		}
	}

	return nil
}

func (this *SimplePeer) GetSenders(streamId string) []*webrtc.RTPSender {
	senders := make([]*webrtc.RTPSender, 0)
	all := this.peerConnection.GetSenders()
	if all != nil && len(all) > 0 {
		for _, sender := range all {
			trackLocal := sender.Track()
			if trackLocal != nil && streamId == trackLocal.StreamID() || streamId == "" {
				senders = append(senders, sender)
			}
		}
	}

	return senders
}

func (this *SimplePeer) GetSendersByKind(kind webrtc.RTPCodecType) []*webrtc.RTPSender {
	senders := make([]*webrtc.RTPSender, 0)
	all := this.peerConnection.GetSenders()
	if all != nil && len(all) > 0 {
		for _, sender := range all {
			trackLocal := sender.Track()
			if kind == trackLocal.Kind() || kind == 0 {
				senders = append(senders, sender)
			}
		}
	}

	return senders
}

func (this *SimplePeer) GetReceiver(trackId string) (*webrtc.TrackRemote, *webrtc.RTPReceiver) {
	all := this.peerConnection.GetReceivers()
	if all != nil && len(all) > 0 {
		for _, receiver := range all {
			trackRemotes := receiver.Tracks()
			if trackRemotes != nil && len(trackRemotes) > 0 {
				for _, trackRemote := range trackRemotes {
					if trackRemote.ID() == trackId {
						return trackRemote, receiver
					}
				}
			}
		}
	}

	return nil, nil
}

func (this *SimplePeer) GetTrackRemotes(streamId string) []*webrtc.TrackRemote {
	all := this.peerConnection.GetReceivers()
	if all != nil && len(all) > 0 {
		trackRemotes := make([]*webrtc.TrackRemote, 0)
		for _, receiver := range all {
			tracks := receiver.Tracks()
			if tracks != nil && len(tracks) > 0 {
				for _, track := range tracks {
					if track.StreamID() == streamId || streamId == "" {
						trackRemotes = append(trackRemotes, track)
					}
				}
			}
		}
		return trackRemotes
	}

	return nil
}

func (this *SimplePeer) GetTrackRemotesByKind(kind webrtc.RTPCodecType) []*webrtc.TrackRemote {
	all := this.peerConnection.GetReceivers()
	if all != nil && len(all) > 0 {
		trackRemotes := make([]*webrtc.TrackRemote, 0)
		for _, receiver := range all {
			tracks := receiver.Tracks()
			if tracks != nil && len(tracks) > 0 {
				for _, track := range tracks {
					if track.Kind() == kind || kind == 0 {
						trackRemotes = append(trackRemotes, track)
					}
				}
			}
		}
		return trackRemotes
	}

	return nil
}

func (this *SimplePeer) StopReceiver(trackId string) error {
	all := this.peerConnection.GetReceivers()
	if all != nil && len(all) > 0 {
		for _, receiver := range all {
			track := receiver.Track()
			if track.ID() == trackId {
				return receiver.Stop()
			}
		}
	}

	return errors.New("NotExist")
}

func (this *SimplePeer) StopStream(streamId string) error {
	all := this.peerConnection.GetReceivers()
	var err error
	if all != nil && len(all) > 0 {
		for _, receiver := range all {
			tracks := receiver.Tracks()
			if tracks != nil && len(tracks) > 0 {
				for _, track := range tracks {
					if track.StreamID() == streamId || streamId == "" {
						err1 := receiver.Stop()
						if err1 != nil {
							err = err1
						}
					}
				}
			}
		}
	}

	return err
}

/*
把remote track的数据发回
*/
func (this *SimplePeer) echo(track *webrtc.TrackRemote) error {
	c := track.Codec().RTPCodecCapability
	trackLocal, err := this.CreateTrack(&c, track.ID(), track.StreamID())
	if err != nil {
		logger.Sugar.Errorf(err.Error())
		return err
	}
	sender, err := this.AddTrack(trackLocal)
	if err != nil {
		logger.Sugar.Errorf(err.Error())
		return err
	}
	for {
		packet, err := this.ReadRTP(track)
		if err != nil {
			return err
		}
		go this.WriteRTP(sender, packet)
	}
}

func (this *SimplePeer) simulcast(track *webrtc.TrackRemote) error {
	for {
		// Read RTP packets being sent to Pion
		packet, readErr := this.ReadRTP(track)
		if readErr != nil {
			return readErr
		}

		for _, sender := range this.GetSenders("") {
			go this.WriteRTP(sender, packet)
		}
	}
}

/*
把remote track的数据发到stream中trackId的轨道中
*/
func (this *SimplePeer) forward(track *webrtc.TrackRemote, trackId string) error {
	rtpBuf := make([]byte, 1500)
	for {
		i, _, err := track.Read(rtpBuf)
		if err != nil {
			logger.Sugar.Errorf(err.Error())
			return err
		}
		sender := this.GetSender(trackId)
		if sender != nil {
			go this.Write(sender, rtpBuf[:i])
		} else {
			err := errors.New("trackId:%v is not exist" + trackId)

			return err
		}
	}
}

func (this *SimplePeer) Write(sender *webrtc.RTPSender, buf []byte) error {
	outputTrack, ok := sender.Track().(*webrtc.TrackLocalStaticRTP)
	if ok {
		// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
		_, err := outputTrack.Write(buf)
		if err != nil && !errors.Is(err, io.ErrClosedPipe) {
			logger.Sugar.Errorf(err.Error())

			return err
		}
	} else {
		err := errors.New("trackId:%v is not TrackLocalStaticRTP" + sender.Track().ID())

		return err
	}

	return nil
}

func (this *SimplePeer) ReadRTP(track *webrtc.TrackRemote) (*rtp.Packet, error) {
	packet, _, err := track.ReadRTP()
	if err != nil {
		logger.Sugar.Errorf(err.Error())

		return nil, err
	}

	return packet, nil
}

func (this *SimplePeer) WriteRTP(sender *webrtc.RTPSender, packet *rtp.Packet) error {
	outputTrack, ok := sender.Track().(*webrtc.TrackLocalStaticRTP)
	if ok {
		// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
		err := outputTrack.WriteRTP(packet)
		if err != nil && !errors.Is(err, io.ErrClosedPipe) {
			logger.Sugar.Errorf(err.Error())

			return err
		}
	} else {
		err := errors.New("trackId:%v is not TrackLocalStaticRTP" + sender.Track().ID())

		return err
	}

	return nil
}
