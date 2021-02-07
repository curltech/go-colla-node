package simplepeer

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	webrtc2 "github.com/curltech/go-colla-node/webrtc"
	"github.com/pion/webrtc/v3"
)

func (this *SimplePeer) BufferedAmount() uint64 {
	return this.dataChannel.BufferedAmount()
}

// HACK: it's possible channel.readyState is "closing" before peer.destroy() fires
// https://bugs.chromium.org/p/chromium/issues/detail?id=882743
func (this *SimplePeer) Connected() bool {
	return (this.connected && this.dataChannel.ReadyState() == webrtc.DataChannelStateOpen)
}

func (this *SimplePeer) setupDataChannel(dataChannel *webrtc.DataChannel) {
	if dataChannel == nil {
		this.Destroy(errors.New(webrtc2.ERR_DATA_CHANNEL))
	}
	this.dataChannel = dataChannel
	this.dataChannel.SetBufferedAmountLowThreshold(webrtc2.MAX_BUFFERED_AMOUNT)
	this.channelName = this.dataChannel.Label()

	this.dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		this.onChannelMessage(msg)
	})

	/**
	当缓存的待发送数据低于阀值的时候触发
	*/
	this.dataChannel.OnBufferedAmountLow(func() {
		this.onChannelBufferedAmountLow()
	})

	this.dataChannel.OnOpen(func() {
		this.onChannelOpen()
	})
	this.dataChannel.OnClose(func() {
		this.onChannelClose()
	})
	this.dataChannel.OnError(func(err error) {
		this.Destroy(errors.New(webrtc2.ERR_DATA_CHANNEL))
	})
}

func (this *SimplePeer) onChannelMessage(msg webrtc.DataChannelMessage) {
	logger.Sugar.Infof("DataChannel message receive")
	if this.destroyed {
		return
	}
	this.EmitEvent(webrtc2.EVENT_DATA, &webrtc2.PeerEvent{Data: msg.Data})
}

/**
低于阀值的时候触发
*/
func (this *SimplePeer) onChannelBufferedAmountLow() {
	if this.destroyed {
		return
	}
	logger.Sugar.Debugf("ending backpressure: bufferedAmount %d", this.dataChannel.BufferedAmount())
}

func (this *SimplePeer) onChannelOpen() {
	this.channelReady = true
	if this.connected || !this.pcReady || this.destroyed {
		return
	}
	this.connecting = false
	this.connected = true
	logger.Sugar.Infof("connected")
	/**
	初始化数据的缓冲池
	*/
	this.sendChan = make(chan []byte, 1)
	go this.loopSend()
	this.setStatsReport()
	this.EmitEvent(webrtc2.EVENT_CONNECT, nil)
}

/**
数据通道关闭，关闭节点
*/
func (this *SimplePeer) onChannelClose() {
	logger.Sugar.Infof("DataChannel close")
	if this.destroyed {
		return
	}

	this.Destroy(nil)
}

/**
把要发送的数据送入缓存
*/
func (this *SimplePeer) SendText(message string) error {
	return this.Send([]byte(message))
}

/**
把要发送的数据送入缓存
*/
func (this *SimplePeer) Send(data []byte) error {
	if this.destroyed {
		return errors.New(webrtc2.ERR_DATA_CHANNEL)
	}
	this.sendChan <- data

	return nil
}

/**
在独立的线程中循环发送缓存的数据
*/
func (this *SimplePeer) loopSend() {
	var (
		data []byte
		err  error
	)

	for {
		select {
		case data = <-this.sendChan:
			//case <-this.closeChan:
			//	this.Close()
		}
		if err = this.dataChannel.Send(data); err != nil {
			//this.Close()
		}
	}
}

func (this *SimplePeer) checkClosing() {
	// HACK: Chrome will sometimes get stuck in readyState "closing", let's check for this condition
	// https://bugs.chromium.org/p/chromium/issues/detail?id=882743
	//var isClosing = false
	//t := time.NewTicker(CHANNEL_CLOSING_TIMEOUT * time.Millisecond)
	//for {
	//	select {
	//	case <-t.C:
	//		if this.dataChannel != nil && this.dataChannel.ReadyState() == webrtc.DataChannelStateClosing {
	//			if isClosing {
	//				this.onChannelClose() // closing timed out: equivalent to onclose firing
	//			}
	//			isClosing = true
	//		} else {
	//			isClosing = false
	//		}
	//	}
	//	t.Stop()
	//}
}
