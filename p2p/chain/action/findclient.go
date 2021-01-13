package action

import (
	"errors"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	service1 "github.com/curltech/go-colla-node/p2p/chain/service"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/kataras/golog"
)

type findClientAction struct {
	BaseAction
}

var FindClientAction findClientAction

/**
主动发送消息
*/
func (this *findClientAction) Send(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	golog.Infof("Send %v message", this.MsgType)
	response := &msg.ChainMessage{}

	return response, nil
}

/**
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (this *findClientAction) PCReceive(chainMessage *msg.PCChainMessage) (interface{}, error) {
	golog.Infof("Receive %v message", this.MsgType)
	conditionBean := chainMessage.MessagePayload.Payload.(map[string]interface{})
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
		golog.Errorf("InvalidPeerClientKey")
		return msgtype.ERROR, errors.New("InvalidPeerClientKey")
	}
	if config.Libp2pParams.FaultTolerantLevel == 0 {
		recvdVals, err := dht.PeerEndpointDHT.GetValues(key, config.Libp2pParams.Nvals)
		if err != nil {
			return msgtype.ERROR, err
		}
		for _, recvdVal := range recvdVals {
			pcs := make([]*entity.PeerClient, 0)
			err = message.TextUnmarshal(string(recvdVal.Val), &pcs)
			if err != nil {
				golog.Errorf("failed to TextUnmarshal PeerClient value: %v, err: %v", recvdVal.Val, err)
				return msgtype.ERROR, err
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
			locals, err = service1.GetLocalPCs(ns.PeerClient_KeyKind, peerId, "", "")
		} else if len(mobileNumber) > 0 {
			locals, err = service1.GetLocalPCs(ns.PeerClient_Mobile_KeyKind, "", mobileNumber, "")
		}
		if err != nil {
			return msgtype.ERROR, err
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
			return msgtype.ERROR, err
		}
		// 恢复local记录
		err = service1.PutLocalPCs(locals)
		if err != nil {
			return msgtype.ERROR, err
		}
		// 整合记录
		for _, recvdVal := range recvdVals {
			pcs := make([]*entity.PeerClient, 0)
			err = message.TextUnmarshal(string(recvdVal.Val), &pcs)
			if err != nil {
				golog.Errorf("failed to TextUnmarshal PeerClient value: %v, err: %v", recvdVal.Val, err)
				return msgtype.ERROR, err
			}
			for _, pc := range pcs {
				peerClients = append(peerClients, pc)
			}
		}
	}
	// 返回最优记录
	if len(peerClients) > 0 {
		var best *entity.PeerClient
		for _, peerClient := range peerClients {
			if best == nil {
				best = peerClient
			} else {
				if peerClient.LastUpdateTime.UTC().After(best.LastUpdateTime.UTC()) ||
					(peerClient.LastUpdateTime.UTC().Equal(best.LastUpdateTime.UTC()) &&
						peerClient.LastAccessTime.UTC().After(best.LastAccessTime.UTC())) {
					best = peerClient
				}
			}
		}
		return best, nil
	}

	return nil, nil
}

/**
处理返回消息
*/
func (this *findClientAction) PCResponse(chainMessage *msg.PCChainMessage) error {
	golog.Infof("Response %v message:%v", this.MsgType, chainMessage)

	return nil
}

func init() {
	FindClientAction = findClientAction{}
	FindClientAction.MsgType = msgtype.TRANSFINDCLIENT
	handler.RegistChainMessageHandler(msgtype.TRANSFINDCLIENT, FindClientAction.Send, FindClientAction.Receive, FindClientAction.Response)
	handler.RegistPCChainMessageHandler(msgtype.TRANSFINDCLIENT, FindClientAction.Send, FindClientAction.PCReceive, FindClientAction.PCResponse)
}
