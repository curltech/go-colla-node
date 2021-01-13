package dht

import (
	"errors"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/p2p"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/chain/handler/sender"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/kataras/golog"
	"time"
)

type signalAction struct {
	action.BaseAction
	receivers map[string]func(peer *p2p.NetPeer, payload map[string]interface{})
}

var SignalAction signalAction

func (this *signalAction) RegistReceiver(name string, receiver func(peer *p2p.NetPeer, payload map[string]interface{})) error {
	if this.receivers == nil {
		this.receivers = make(map[string]func(peer *p2p.NetPeer, payload map[string]interface{}))
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
func (this *signalAction) Signal(peerId string, payloadType string, data interface{}, targetPeerId string) (interface{}, error) {
	golog.Infof("Send %v message", this.MsgType)
	chainMessage := msg.ChainMessage{}
	chainMessage.TargetPeerId = targetPeerId
	chainMessage.Payload = data
	chainMessage.ConnectPeerId = peerId
	chainMessage.PayloadType = payloadType
	chainMessage.MessageType = msgtype.SIGNAL
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

func (this *signalAction) Receive(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	var err error
	if chainMessage.TargetPeerId != "" && global.IsMyself(chainMessage.TargetPeerId) {
		signal := chainMessage.Payload.(map[string]interface{})
		if this.receivers == nil || len(this.receivers) == 0 {
			golog.Errorf("NoReceiver")
			err = errors.New("NoReceiver")
		} else {
			for _, receiver := range this.receivers {
				netPeer := &p2p.NetPeer{TargetPeerId: chainMessage.SrcPeerId, ConnectPeerId: chainMessage.SrcConnectPeerId, ConnectSessionId: chainMessage.SrcConnectSessionId}
				go receiver(netPeer, signal)
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
