package dht

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

type findPeerAction struct {
	action.BaseAction
}

var FindPeerAction findPeerAction

/**
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (this *findPeerAction) Receive(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Sugar.Infof("Receive %v message", this.MsgType)
	var response *msg.ChainMessage = nil
	conditionBean, ok := chainMessage.Payload.(map[string]interface{})
	if !ok {
		response = handler.Error(chainMessage.MessageType, errors.New("ErrorCondition"))
		return response, nil
	}
	var peerId string = ""
	if conditionBean["peerId"] != nil {
		peerId = conditionBean["peerId"].(string)
	}
	if len(peerId) > 0 {
		addrInfo, err := service.GetPeerEndpointService().FindPeer(peerId)
		if err != nil {
			response = handler.Error(chainMessage.MessageType, err)
			return response, nil
		}
		response = handler.Response(chainMessage.MessageType, addrInfo)
	}

	return response, nil
}

func init() {
	FindPeerAction = findPeerAction{}
	FindPeerAction.MsgType = msgtype.FINDPEER
	handler.RegistChainMessageHandler(msgtype.FINDPEER, FindPeerAction.Send, FindPeerAction.Receive, FindPeerAction.Response)
}
