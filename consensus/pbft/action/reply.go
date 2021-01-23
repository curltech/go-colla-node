package action

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"time"
)

type replyAction struct {
	action.BaseAction
}

var ReplyAction replyAction

func (this *replyAction) Reply(peerId string, data interface{}, targetPeerId string) (interface{}, error) {
	logger.Infof("Receive %v message", this.MsgType)
	chainMessage := msg.ChainMessage{}
	chainMessage.TargetPeerId = targetPeerId
	chainMessage.Payload = data
	chainMessage.ConnectPeerId = peerId
	chainMessage.PayloadType = handler.PayloadType_PbftConsensusLog
	chainMessage.MessageType = msgtype.CONSENSUS_PBFT_REPLY
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

func (this *replyAction) Receive(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	response := handler.Response(chainMessage.MessageType, time.Now())

	return response, nil
}

func init() {
	ReplyAction = replyAction{}
	ReplyAction.MsgType = msgtype.CONSENSUS_PBFT_REPLY
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_PBFT_REPLY, ReplyAction.Send, ReplyAction.Receive, ReplyAction.Response)
}
