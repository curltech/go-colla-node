package action

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/security"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	_ "github.com/curltech/go-colla-node/p2p/chain/handler/receiver"
	"github.com/curltech/go-colla-node/p2p/chain/handler/sender"
	"github.com/curltech/go-colla-node/p2p/msg/entity"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"time"
)

type BaseAction struct {
	MsgType string
}

func (this *BaseAction) PrepareSend(connectPeerId string, data interface{}, targetPeerId string) *entity.ChainMessage {
	chainMessage := entity.ChainMessage{}
	if connectPeerId == "" {
		connectPeerId = targetPeerId
	}
	chainMessage.ConnectPeerId = connectPeerId
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
func (this *BaseAction) Send(chainMessage *entity.ChainMessage) (*entity.ChainMessage, error) {
	logger.Sugar.Infof("Send %v message", this.MsgType)
	response, err := sender.Send(chainMessage)

	return response, err
}

/**
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (this *BaseAction) Receive(chainMessage *entity.ChainMessage) (*entity.ChainMessage, error) {
	logger.Sugar.Infof("Receive %v message", this.MsgType)
	go sender.RelaySend(chainMessage)
	response := handler.Response(chainMessage.MessageType, time.Now())

	return response, nil
}

/**
处理返回消息
*/
func (this *BaseAction) Response(chainMessage *entity.ChainMessage) error {
	logger.Sugar.Infof("Response %v message:%v", this.MsgType, chainMessage)

	return nil
}

func init() {
}
