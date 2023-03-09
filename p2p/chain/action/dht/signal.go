package dht

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/chain/handler/sender"
	"github.com/curltech/go-colla-node/p2p/msg/entity"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"time"
)

type signalAction struct {
	action.BaseAction
	receivers map[string]func(peerId string, signal map[string]interface{},
		clientId string, connectPeerId string, connectSessionId string)
}

var SignalAction signalAction

func (this *signalAction) RegistReceiver(name string, receiver func(peerId string, signal map[string]interface{},
	clientId string, connectPeerId string, connectSessionId string)) error {
	if this.receivers == nil {
		this.receivers = make(map[string]func(peerId string, signal map[string]interface{},
			clientId string, connectPeerId string, connectSessionId string))
	}
	_, ok := this.receivers[name]
	if ok {
		return errors.New("Exist")
	}
	this.receivers[name] = receiver

	return nil
}

/**
peerId如果为空，发送的对象是自己，需要检查如果是自己，则检查最终目标，考虑转发
*/
func (this *signalAction) Signal(connectPeerId string, data interface{}, targetPeerId string) (interface{}, error) {
	chainMessage := this.PrepareSend(connectPeerId, data, targetPeerId)

	response, err := this.Send(chainMessage)
	if err != nil {
		return nil, err
	}
	if response != nil {
		return response.Payload, nil
	}

	return nil, nil
}

func (this *signalAction) Receive(chainMessage *entity.ChainMessage) (*entity.ChainMessage, error) {
	logger.Sugar.Infof("Receive %v message", this.MsgType)
	var err error
	///目标是自己节点
	if chainMessage.TargetPeerId != "" && global.IsMyself(chainMessage.TargetPeerId) {
		signal := chainMessage.Payload.(map[string]interface{})
		if this.receivers == nil || len(this.receivers) == 0 {
			logger.Sugar.Errorf("NoReceiver")
			err = errors.New("NoReceiver")
		} else {
			for _, receiver := range this.receivers {
				go receiver(chainMessage.SrcPeerId, signal, chainMessage.SrcClientId,
					chainMessage.SrcConnectPeerId, chainMessage.SrcConnectSessionId)
			}
		}
	} else {
		go sender.RelaySend(chainMessage)
	}
	response := handler.Response(chainMessage.MessageType, time.Now())

	return response, err
}

func init() {
	SignalAction = signalAction{}
	SignalAction.MsgType = msgtype.SIGNAL
	handler.RegistChainMessageHandler(msgtype.SIGNAL, SignalAction.Send, SignalAction.Receive, SignalAction.Response)
}
