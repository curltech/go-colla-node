package dht

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/consensus/std"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/ns"
	handler1 "github.com/curltech/go-colla-node/libp2p/pipe/handler"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/chain/handler/sender"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

type p2pChatAction struct {
	action.BaseAction
}

var P2pChatAction p2pChatAction

/**
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (this *p2pChatAction) Receive(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Sugar.Infof("Receive %v message", this.MsgType)
	var response *msg.ChainMessage = nil

	targetPeerId := handler1.GetPeerId(chainMessage.TargetPeerId)
	key := ns.GetPeerClientKey(targetPeerId)
	peerClients, err := service.GetPeerClientService().GetLocals(key, "")
	if err != nil || len(peerClients) == 0 {
		peerClients, err = service.GetPeerClientService().GetValues(targetPeerId, "")
	}
	if err != nil {
		response = handler.Error(chainMessage.MessageType, err)
		return response, nil
	}
	if len(peerClients) == 0 {
		response = handler.Error(chainMessage.MessageType, errors.New("NUllPeerClients"))
		return response, nil
	}
	sent := false
	for _, peerClient := range peerClients {
		if peerClient.ActiveStatus == entity.ActiveStatus_Up {
			// 如果PeerClient的连接节点是自己，下一步就是最终目标，将目标会话放入消息中
			sent = true
			if global.IsMyself(peerClient.ConnectPeerId) {
				chainMessage.TargetConnectSessionId = peerClient.ConnectSessionId
				chainMessage.TargetConnectPeerId = peerClient.ConnectPeerId
				chainMessage.ConnectPeerId = chainMessage.TargetPeerId
			} else { // 否则下一步就是连接节点
				chainMessage.TargetConnectSessionId = peerClient.ConnectSessionId
				chainMessage.TargetConnectPeerId = peerClient.ConnectPeerId
				chainMessage.ConnectPeerId = peerClient.ConnectPeerId
			}
			go sender.SendCM(chainMessage)
		}
	}
	if sent == false {
		handler.Decrypt(chainMessage)
		response, _ = std.GetStdConsensus().ReceiveConsensus(chainMessage)
		return response, nil
	}

	return nil, nil
}

func init() {
	P2pChatAction = p2pChatAction{}
	P2pChatAction.MsgType = msgtype.P2PCHAT
	handler.RegistChainMessageHandler(msgtype.P2PCHAT, P2pChatAction.Send, P2pChatAction.Receive, P2pChatAction.Response)
}
