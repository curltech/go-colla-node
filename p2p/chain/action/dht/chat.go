package dht

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/chain/handler/sender"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"time"
)

type chatAction struct {
	action.BaseAction
}

var ChatAction chatAction

/**
在chain目录下的采用自定义protocol "/chain"的方式自己实现的功能
*/
func (this *chatAction) Chat(peerId string, payloadType string, data interface{}, targetPeerId string) (interface{}, error) {
	logger.Infof("Receive %v message", this.MsgType)
	chainMessage := msg.ChainMessage{}
	chainMessage.TargetPeerId = targetPeerId
	chainMessage.Payload = data
	chainMessage.ConnectPeerId = peerId
	chainMessage.PayloadType = payloadType
	chainMessage.MessageType = msgtype.CHAT
	chainMessage.MessageDirect = msgtype.MsgDirect_Request

	response, err := this.Send(&chainMessage)
	if err != nil {
		return nil, err
	}
	if response != nil {
		return response.Payload, nil
	}

	return nil, nil
}

func (this *chatAction) Receive(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	go sender.RelaySend(chainMessage)
	response := handler.Response(chainMessage.MessageType, time.Now())

	return response, nil
}

func init() {
	ChatAction = chatAction{}
	ChatAction.MsgType = msgtype.CHAT
	handler.RegistChainMessageHandler(msgtype.CHAT, ChatAction.Send, ChatAction.Receive, ChatAction.Response)
}
