package dht

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

type findClientAction struct {
	action.BaseAction
}

var FindClientAction findClientAction

/**
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (this *findClientAction) Receive(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Sugar.Infof("Receive %v message", this.MsgType)
	var response *msg.ChainMessage = nil
	conditionBean, ok := chainMessage.Payload.(map[string]interface{})
	if !ok {
		response = handler.Error(chainMessage.MessageType, errors.New("ErrorCondition"))
		return response, nil
	}
	var peerId string = ""
	var mobileNumber string = ""
	if conditionBean["peerId"] != nil {
		peerId = conditionBean["peerId"].(string)
	}
	if conditionBean["mobileNumber"] != nil {
		mobileNumber = conditionBean["mobileNumber"].(string)
	}

	peerClients, err := service.GetPeerClientService().GetValues(peerId, mobileNumber)
	if err != nil {
		response = handler.Error(chainMessage.MessageType, err)
		return response, nil
	}
	response = handler.Response(chainMessage.MessageType, peerClients)
	response.PayloadType = handler.PayloadType_PeerClients

	return response, nil
}

func init() {
	FindClientAction = findClientAction{}
	FindClientAction.MsgType = msgtype.FINDCLIENT
	handler.RegistChainMessageHandler(msgtype.FINDCLIENT, FindClientAction.Send, FindClientAction.Receive, FindClientAction.Response)
}
