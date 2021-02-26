package action

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/security"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/chain/handler/sender"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"time"
)

type BaseAction struct {
	MsgType msgtype.MsgType
}

func (this *BaseAction) PrepareSend(peerId string, data interface{}, targetPeerId string) *msg.ChainMessage {
	chainMessage := msg.ChainMessage{}
	chainMessage.ConnectPeerId = peerId
	chainMessage.Payload = data
	chainMessage.TargetPeerId = targetPeerId
	chainMessage.PayloadType = handler.PayloadType_Map
	chainMessage.MessageType = this.MsgType
	chainMessage.MessageDirect = msgtype.MsgDirect_Request
	chainMessage.NeedCompress = true
	chainMessage.NeedEncrypt = false
	chainMessage.UUID = security.UUID()

	return &chainMessage
}

/**
主动发送消息
*/
func (this *BaseAction) Send(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Sugar.Infof("Send %v message", this.MsgType)
	response, err := sender.Send(chainMessage)

	return response, err
}

/**
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (this *BaseAction) Receive(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Sugar.Infof("Receive %v message", this.MsgType)
	go sender.RelaySend(chainMessage)
	response := handler.Response(chainMessage.MessageType, time.Now())

	return response, nil
}

/**
处理返回消息
*/
func (this *BaseAction) Response(chainMessage *msg.ChainMessage) error {
	logger.Sugar.Infof("Response %v message:%v", this.MsgType, chainMessage)

	return nil
}

func init() {
}
