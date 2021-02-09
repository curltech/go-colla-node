package dht

import (
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

type chatAction struct {
	action.BaseAction
}

var ChatAction chatAction

/**
在chain目录下的采用自定义protocol "/chain"的方式自己实现的功能
*/
func (this *chatAction) Chat(peerId string, payloadType string, data interface{}, targetPeerId string, targetConnectSessionId string) (interface{}, error) {
	chainMessage := msg.ChainMessage{}
	chainMessage.TargetPeerId = targetPeerId
	chainMessage.TargetConnectSessionId = targetConnectSessionId
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

func init() {
	ChatAction = chatAction{}
	ChatAction.MsgType = msgtype.CHAT
	handler.RegistChainMessageHandler(msgtype.CHAT, ChatAction.Send, ChatAction.Receive, ChatAction.Response)
}
