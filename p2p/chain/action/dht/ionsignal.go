package dht

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/p2p"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/chain/handler/sender"
	"github.com/curltech/go-colla-node/p2p/msg/entity"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

type ionSignalAction struct {
	action.BaseAction
	receiver func(netPeer *p2p.NetPeer, payload map[string]interface{}) (interface{}, error)
}

var IonSignalAction ionSignalAction

func (this *ionSignalAction) RegistReceiver(receiver func(netPeer *p2p.NetPeer, payload map[string]interface{}) (interface{}, error)) error {
	if this.receiver == nil {
		this.receiver = receiver
	} else {
		return errors.New("Exist")
	}

	return nil
}

/**
peerId如果为空，发送的对象是自己，需要检查如果是自己，则检查最终目标，考虑转发
*/
func (this *ionSignalAction) Signal(peerId string, data interface{}, targetPeerId string) (interface{}, error) {
	chainMessage := this.PrepareSend(peerId, data, targetPeerId)

	response, err := this.Send(chainMessage)
	if err != nil {
		return nil, err
	}
	if response != nil {
		return response.Payload, nil
	}

	return nil, nil
}

func (this *ionSignalAction) Receive(chainMessage *entity.ChainMessage) (*entity.ChainMessage, error) {
	logger.Sugar.Infof("Receive %v message", this.MsgType)
	var err error
	var response *entity.ChainMessage
	if chainMessage.TargetPeerId != "" && global.IsMyself(chainMessage.TargetPeerId) {
		signal := chainMessage.Payload.(map[string]interface{})
		if this.receiver == nil {
			logger.Sugar.Errorf("NoReceiver")
			err = errors.New("NoReceiver")
		} else {
			netPeer := &p2p.NetPeer{TargetPeerId: chainMessage.SrcPeerId, ConnectPeerId: chainMessage.SrcConnectPeerId, ConnectSessionId: chainMessage.SrcConnectSessionId}
			res, err := this.receiver(netPeer, signal)
			if err != nil {
				return nil, err
			}
			response = handler.Response(chainMessage.MessageType, res)
		}
	} else {
		return sender.RelaySend(chainMessage)
	}

	return response, err
}

func init() {
	IonSignalAction = ionSignalAction{}
	IonSignalAction.MsgType = msgtype.IONSIGNAL
	handler.RegistChainMessageHandler(msgtype.IONSIGNAL, IonSignalAction.Send, IonSignalAction.Receive, IonSignalAction.Response)
}
