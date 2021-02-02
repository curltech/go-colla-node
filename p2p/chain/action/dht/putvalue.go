package dht

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	entity1 "github.com/curltech/go-colla-node/p2p/chain/entity"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

type putValueAction struct {
	action.BaseAction
}

var PutValueAction putValueAction

/**
在chain目录下的采用自定义protocol "/chain"的方式自己实现的功能
*/
func (this *putValueAction) PutValue(peerId string, payloadType string, data interface{}) (interface{}, error) {
	chainMessage := msg.ChainMessage{}
	chainMessage.Payload = data
	chainMessage.ConnectPeerId = peerId
	chainMessage.PayloadType = payloadType
	chainMessage.MessageType = msgtype.PUTVALUE
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

func (this *putValueAction) Receive(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Infof("Receive %v message", this.MsgType)
	var response *msg.ChainMessage = nil
	v := chainMessage.Payload
	peerClient, ok := v.(*entity.PeerClient)
	var key string
	if ok {
		key = "/" + ns.PeerClient_Prefix + "/" + peerClient.PeerId
		peerClient.ConnectSessionId = chainMessage.ConnectSessionId
	} else {
		peerEndpoint, ok := v.(*entity.PeerEndpoint)
		if ok {
			key = "/" + ns.PeerEndpoint_Prefix + "/" + peerEndpoint.PeerId
			peerEndpoint.ConnectSessionId = chainMessage.ConnectSessionId
		} else {
			chainApp, ok := v.(*entity.ChainApp)
			if ok {
				key = "/" + ns.ChainApp_Prefix + "/" + chainApp.PeerId
				chainApp.ConnectSessionId = chainMessage.ConnectSessionId
			} else {
				dataBlock, ok := v.(*entity1.DataBlock)
				if ok {
					key = "/" + ns.DataBlock_Prefix + "/" + dataBlock.BlockId
				}
			}
		}
	}
	buf, err := message.Marshal(v)
	if err != nil {
		response = handler.Error(chainMessage.MessageType, err)
	}
	err = dht.PeerEndpointDHT.PutValue(key, buf)
	if err != nil {
		response = handler.Error(chainMessage.MessageType, err)
	}
	if response == nil {
		response = handler.Ok(chainMessage.MessageType)
	}

	return response, nil
}

func init() {
	PutValueAction = putValueAction{}
	PutValueAction.MsgType = msgtype.PUTVALUE
	handler.RegistChainMessageHandler(msgtype.PUTVALUE, PutValueAction.Send, PutValueAction.Receive, PutValueAction.Response)
}
