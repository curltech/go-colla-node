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
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

type pingAction struct {
	action.BaseAction
}

var PingAction pingAction

/**
在chain目录下的采用自定义protocol "/chain"的方式自己实现的功能
Ping只是一个演示，适合点对点的通信，这种方式灵活度高，但是需要自己实现全网遍历的功能
chat就可以采用这种方式
*/
func (this *pingAction) Ping(peerId string, targetPeerId string) (interface{}, error) {
	chainMessage := msg.ChainMessage{}
	chainMessage.TargetPeerId = targetPeerId
	chainMessage.Payload = global.Global.MyselfPeer
	chainMessage.ConnectPeerId = peerId
	chainMessage.PayloadType = handler.PayloadType_PeerEndpoint
	chainMessage.MessageType = msgtype.PING
	chainMessage.MessageDirect = msgtype.MsgDirect_Request

	response, err := this.Send(&chainMessage)
	if err != nil {
		return nil, err
	}

	if response.Payload == msgtype.OK {
		return response.Payload, nil
	} else {
		return response.Payload, errors.New(response.Tip)
	}
}

func (this *pingAction) Receive(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
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
	PingAction = pingAction{}
	PingAction.MsgType = msgtype.PING
	handler.RegistChainMessageHandler(msgtype.PING, PingAction.Send, PingAction.Receive, PingAction.Response)
}
