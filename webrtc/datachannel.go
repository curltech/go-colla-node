package webrtc

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/pion/webrtc/v3"
)

func (this *BasePeerConnection) BufferedAmount() uint64 {
	return this.dataChannel.BufferedAmount()
}

func (this *BasePeerConnection) Status() PeerConnectionStatus {
	return this.status
}

func (this *BasePeerConnection) setupDataChannel(dataChannel *webrtc.DataChannel) {
	if dataChannel == nil {
		this.Close()
	}
	this.dataChannel = dataChannel
	this.dataChannel.SetBufferedAmountLowThreshold(MAX_BUFFERED_AMOUNT)

	this.dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		this.onMessage(msg)
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
		this.Close()
	})
}

func (this *BasePeerConnection) onMessage(msg webrtc.DataChannelMessage) {
	logger.Sugar.Infof("DataChannel message receive")
	if this.status == PeerConnectionStatus_closed {
		logger.Sugar.Errorf("PeerConnectionStatus closed")
		return
	}
	var data = msg.Data
	slices := this.messageSlice.merge(data)

	if slices != nil {
		logger.Sugar.Infof("webrtc binary onMessage length: ${slices.length}")

		this.Emit(WebrtcEventType_message, &WebrtcEvent{Data: slices})
	}
}

/**
低于阀值的时候触发
*/
func (this *BasePeerConnection) onChannelBufferedAmountLow() {
	if this.status == PeerConnectionStatus_closed {
		logger.Sugar.Errorf("PeerConnectionStatus closed")
		return
	}
	logger.Sugar.Debugf("ending backpressure: bufferedAmount %d", this.dataChannel.BufferedAmount())
}

func (this *BasePeerConnection) onChannelOpen() {
	if this.status == PeerConnectionStatus_closed {
		logger.Sugar.Errorf("PeerConnectionStatus closed")
		return
	}
	logger.Sugar.Infof("data channel open")
	//go this.loopSend()
	this.setStatsReport()
	this.dataChannelOpen = true
}

/**
数据通道关闭，关闭节点
*/
func (this *BasePeerConnection) onChannelClose() {
	logger.Sugar.Infof("DataChannel close")
	if this.status == PeerConnectionStatus_closed {
		logger.Sugar.Errorf("PeerConnectionStatus closed")
		return
	}
	logger.Sugar.Infof("data channel close")
	this.Close()
}

/**
把要发送的数据送入缓存
*/
func (this *BasePeerConnection) SendText(message string) error {
	return this.Send([]byte(message))
}

/**
把要发送的数据送入缓存
*/
func (this *BasePeerConnection) Send(data []byte) error {
	if this.status == PeerConnectionStatus_closed {
		logger.Sugar.Errorf("PeerConnectionStatus closed")
		return errors.New("PeerConnectionStatus closed")
	}
	slices := this.messageSlice.slice(data)
	for _, slice := range slices {
		//this.sendChan <- slice
		err := this.dataChannel.Send(slice)
		if err != nil {
			logger.Sugar.Errorf("error:%v", err.Error())
			return err
		}
	}

	return nil
}

/**
在独立的线程中循环发送缓存的数据
*/
func (this *BasePeerConnection) loopSend() {
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
