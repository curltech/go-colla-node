package dht

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/libp2p/go-libp2p-core/peer"
)

type findPeerAction struct {
	action.BaseAction
}

var FindPeerAction findPeerAction

/**
在chain目录下的采用自定义protocol "/chain"的方式自己实现的功能
*/
func (this *findPeerAction) FindPeer(peerId string, payloadType string, data interface{}) (interface{}, error) {
	chainMessage := msg.ChainMessage{}
	chainMessage.Payload = data
	chainMessage.ConnectPeerId = peerId
	chainMessage.PayloadType = payloadType
	chainMessage.MessageType = msgtype.FINDPEER
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

func (this *findPeerAction) Receive(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Sugar.Infof("Receive %v message", this.MsgType)
	v := chainMessage.Payload
	id, ok := v.(string)
	if !ok {
		return nil, errors.New("ErrorId")
	}
	ID, err := peer.Decode(id)
	if err != nil {

	}
	addrInfo, err := dht.PeerEndpointDHT.FindPeer(ID)
	if err != nil {
		return nil, err
	}
	response := handler.Response(chainMessage.MessageType, addrInfo.String())

	return response, nil
}

func init() {
	FindPeerAction = findPeerAction{}
	FindPeerAction.MsgType = msgtype.FINDPEER
	handler.RegistChainMessageHandler(msgtype.FINDPEER, FindPeerAction.Send, FindPeerAction.Receive, FindPeerAction.Response)
}
