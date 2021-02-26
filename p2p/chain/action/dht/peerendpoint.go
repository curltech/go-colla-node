package dht

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/chain/handler/sender"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

type peerEndPointAction struct {
	action.BaseAction
}

var PeerEndPointAction peerEndPointAction

/**
在chain目录下的采用自定义protocol "/chain"的方式自己实现的功能
*/
func (this *peerEndPointAction) PeerEndPoint(targetPeerId string) (interface{}, error) {
	chainMessage := this.PrepareSend("", global.Global.MyselfPeer, targetPeerId)
	chainMessage.PayloadType = handler.PayloadType_PeerEndpoint

	response, err := sender.DirectSend(chainMessage)
	if err != nil {
		return nil, err
	}

	if response.Payload == msgtype.OK {
		return response.Payload, nil
	} else {
		return response.Payload, errors.New(response.Tip)
	}
}

func (this *peerEndPointAction) Receive(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Sugar.Infof("Receive %v message", this.MsgType)
	var response *msg.ChainMessage = nil
	if chainMessage.Payload != nil {
		srcPeerEndpoint := chainMessage.Payload.(*entity.PeerEndpoint)
		key := ns.GetPeerEndpointKey(srcPeerEndpoint.PeerId)
		byteSrcPeerEndpoint, err := message.Marshal(srcPeerEndpoint)
		if err != nil {
			logger.Sugar.Errorf("failed to Marshal SrcMyselfPeer, err: %v", err)
		} else {
			err = dht.PeerEndpointDHT.PutLocal(key, byteSrcPeerEndpoint)
		}
		if err != nil {
			response = handler.Error(chainMessage.MessageType, err)
		}
		if response == nil {
			response = handler.Ok(chainMessage.MessageType)
		}
	}
	return response, nil
}

func init() {
	PeerEndPointAction = peerEndPointAction{}
	PeerEndPointAction.MsgType = msgtype.PEERENDPOINT
	handler.RegistChainMessageHandler(msgtype.PEERENDPOINT, PeerEndPointAction.Send, PeerEndPointAction.Receive, PeerEndPointAction.Response)
}
