package action

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/entity"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/chain/handler/sender"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

type consensusAction struct {
	action.BaseAction
}

var ConsensusAction consensusAction

func (this *consensusAction) ConsensusDataBlock(peerId string, msgType string, dataBlock *entity.DataBlock, targetPeerId string) (interface{}, error) {
	logger.Sugar.Infof("Receive %v message", this.MsgType)
	if targetPeerId == "" {
		targetPeerId = peerId
	}
	chainMessage := msg.ChainMessage{}
	chainMessage.TargetPeerId = targetPeerId
	chainMessage.Payload = dataBlock
	chainMessage.ConnectPeerId = peerId
	chainMessage.PayloadType = handler.PayloadType_DataBlock
	chainMessage.MessageType = msgtype.MsgType(msgType)
	chainMessage.MessageDirect = msgtype.MsgDirect_Request
	chainMessage.NeedCompress = true

	response, err := sender.DirectSend(&chainMessage) // 定位器之间
	if err != nil {
		return nil, err
	}
	if response != nil {
		return response.Payload, nil
	}

	return nil, nil
}

func (this *consensusAction) ConsensusLog(peerId string, msgType string, consensusLog *entity.ConsensusLog, targetPeerId string) (interface{}, error) {
	logger.Sugar.Infof("Receive %v message", this.MsgType)
	if targetPeerId == "" {
		targetPeerId = peerId
	}
	chainMessage := msg.ChainMessage{}
	chainMessage.TargetPeerId = targetPeerId
	chainMessage.Payload = consensusLog
	chainMessage.ConnectPeerId = peerId
	chainMessage.PayloadType = handler.PayloadType_ConsensusLog
	chainMessage.MessageType = msgtype.MsgType(msgType)
	chainMessage.MessageDirect = msgtype.MsgDirect_Request
	chainMessage.NeedCompress = true

	var response *msg.ChainMessage
	var err error
	if peerId == consensusLog.PrimaryPeerId { // 定位器之间
		response, err = sender.DirectSend(&chainMessage)
	} else {
		response, err = this.Send(&chainMessage)
	}
	if err != nil {
		return nil, err
	}
	if response != nil {
		return response.Payload, nil
	}

	return nil, nil
}

func init() {
	ConsensusAction = consensusAction{}
	ConsensusAction.MsgType = msgtype.CONSENSUS
}
