package dht

import (
	"errors"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

type findClientAction struct {
	action.BaseAction
}

var FindClientAction findClientAction

/**
主动发送消息
*/
func (this *findClientAction) FindClient(peerId string, targetPeerId string, mobileNumber string) (interface{}, error) {
	chainMessage := msg.ChainMessage{}
	conditionBean := make(map[string]interface{})
	conditionBean["peerId"] = targetPeerId
	conditionBean["mobileNumber"] = mobileNumber
	chainMessage.Payload = conditionBean
	chainMessage.ConnectPeerId = peerId
	chainMessage.PayloadType = handler.PayloadType_Map
	chainMessage.MessageType = msgtype.FINDCLIENT
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

	peerClients := make([]*entity.PeerClient, 0)
	var key string
	if len(peerId) > 0 {
		key = ns.GetPeerClientKey(peerId)
	} else if len(mobileNumber) > 0 {
		key = ns.GetPeerClientMobileKey(mobileNumber)
	} else {
		logger.Sugar.Errorf("InvalidPeerClientKey")
		response = handler.Error(chainMessage.MessageType, errors.New("InvalidPeerClientKey"))
		return response, nil
	}
	if config.Libp2pParams.FaultTolerantLevel == 0 {
		recvdVals, err := dht.PeerEndpointDHT.GetValues(key, config.Libp2pParams.Nvals)
		if err != nil {
			response = handler.Error(chainMessage.MessageType, err)
			return response, nil
		}
		for _, recvdVal := range recvdVals {
			pcs := make([]*entity.PeerClient, 0)
			err = message.TextUnmarshal(string(recvdVal.Val), &pcs)
			if err != nil {
				logger.Sugar.Errorf("failed to TextUnmarshal PeerClient value: %v, err: %v", recvdVal.Val, err)
				response = handler.Error(chainMessage.MessageType, err)
				return response, nil
			}
			for _, pc := range pcs {
				peerClients = append(peerClients, pc)
			}
		}
	} else if config.Libp2pParams.FaultTolerantLevel == 1 {
		// 查询删除local记录
		var locals []*entity.PeerClient
		var err error
		if len(peerId) > 0 {
			locals, err = service.GetPeerClientService().GetLocals(ns.PeerClient_KeyKind, peerId, "", "")
		} else if len(mobileNumber) > 0 {
			locals, err = service.GetPeerClientService().GetLocals(ns.PeerClient_Mobile_KeyKind, "", mobileNumber, "")
		}
		if err != nil {
			response = handler.Error(chainMessage.MessageType, err)
			return response, nil
		}
		if len(locals) > 0 {
			for _, local := range locals {
				peerClients = append(peerClients, local)
			}
			service.GetPeerClientService().Delete(locals, "")
		}
		// 查询non-local记录
		recvdVals, err := dht.PeerEndpointDHT.GetValues(key, config.Libp2pParams.Nvals)
		if err != nil {
			response = handler.Error(chainMessage.MessageType, err)
			return response, nil
		}
		// 恢复local记录
		err = service.GetPeerClientService().PutLocals(locals)
		if err != nil {
			response = handler.Error(chainMessage.MessageType, err)
			return response, nil
		}
		// 整合记录
		for _, recvdVal := range recvdVals {
			pcs := make([]*entity.PeerClient, 0)
			err = message.TextUnmarshal(string(recvdVal.Val), &pcs)
			if err != nil {
				logger.Sugar.Errorf("failed to TextUnmarshal PeerClient value: %v, err: %v", recvdVal.Val, err)
				response = handler.Error(chainMessage.MessageType, err)
				return response, nil
			}
			for _, pc := range pcs {
				peerClients = append(peerClients, pc)
			}
		}
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
