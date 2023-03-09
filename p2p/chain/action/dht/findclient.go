package dht

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	"github.com/curltech/go-colla-node/p2p/msg/entity"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

type findClientAction struct {
	action.BaseAction
}

var FindClientAction findClientAction

// Receive 根据peerid，mobile，name进行peerclient的查询，返回查询的结果
func (this *findClientAction) Receive(chainMessage *entity.ChainMessage) (*entity.ChainMessage, error) {
	logger.Sugar.Infof("Receive %v message", this.MsgType)
	var response *entity.ChainMessage = nil
	conditionBean, ok := chainMessage.Payload.(map[string]interface{})
	if !ok {
		response = handler.Error(chainMessage.MessageType, errors.New("ErrorCondition"))
		return response, nil
	}
	var peerId string = ""
	var mobile string = ""
	var email string = ""
	var name string = ""
	if conditionBean["peerId"] != nil {
		peerId = conditionBean["peerId"].(string)
	}
	if conditionBean["mobile"] != nil {
		mobile = conditionBean["mobile"].(string)
	}
	if conditionBean["email"] != nil {
		email = conditionBean["email"].(string)
	}
	if conditionBean["name"] != nil {
		name = conditionBean["name"].(string)
	}

	peerClients, err := service.GetPeerClientService().GetValues(peerId, mobile, email, name)
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
