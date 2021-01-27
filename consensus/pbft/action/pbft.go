package action

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

type pbftAction struct {
	action.BaseAction
}

var PbftAction pbftAction

func (this *pbftAction) Pbft(peerId string, data interface{}, targetPeerId string) (interface{}, error) {
	logger.Infof("Receive %v message", this.MsgType)
	chainMessage := msg.ChainMessage{}
	chainMessage.TargetPeerId = targetPeerId
	chainMessage.Payload = data
	chainMessage.ConnectPeerId = peerId
	chainMessage.PayloadType = handler.PayloadType_DataBlock
	chainMessage.MessageType = msgtype.CONSENSUS_PBFT
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

func init() {
	PbftAction = pbftAction{}
	PbftAction.MsgType = msgtype.CONSENSUS_PBFT
}
