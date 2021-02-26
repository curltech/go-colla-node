package dht

import (
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

type chatAction struct {
	action.BaseAction
}

var ChatAction chatAction

/**
在chain目录下的采用自定义protocol "/chain"的方式自己实现的功能
*/
func (this *chatAction) Chat(peerId string, data interface{}, targetPeerId string, targetConnectSessionId string) (interface{}, error) {
	chainMessage := this.PrepareSend(peerId, data, targetPeerId)
	chainMessage.TargetConnectSessionId = targetConnectSessionId

	response, err := this.Send(chainMessage)
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
