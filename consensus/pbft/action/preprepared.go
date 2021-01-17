package action

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"time"
)

type prepreparedAction struct {
	action.BaseAction
}

var PrepreparedAction prepreparedAction

func (this *prepreparedAction) Preprepared(peerId string, data interface{}, targetPeerId string) (interface{}, error) {
	logger.Infof("Receive %v message", this.MsgType)
	chainMessage := msg.ChainMessage{}
	chainMessage.TargetPeerId = targetPeerId
	chainMessage.Payload = data
	chainMessage.ConnectPeerId = peerId
	chainMessage.PayloadType = handler.PayloadType_DataBlock
	chainMessage.MessageType = msgtype.CONSENSUS_PBFT_PREPREPARED
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

func (this *prepreparedAction) Receive(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	response := handler.Response(chainMessage.MessageType, time.Now())

	return response, nil
}

func init() {
	PrepreparedAction = prepreparedAction{}
	PrepreparedAction.MsgType = msgtype.CONSENSUS_PBFT_PREPREPARED
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_PBFT_PREPREPARED, PrepreparedAction.Send, PrepreparedAction.Receive, PrepreparedAction.Response)
}
