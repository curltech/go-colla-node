package dht

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

type getValueAction struct {
	action.BaseAction
}

var GetValueAction getValueAction

/**
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (this *getValueAction) Receive(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Sugar.Infof("Receive %v message", this.MsgType)
	v := chainMessage.Payload
	key, ok := v.(string)
	if !ok {
		return nil, errors.New("ErrorKey")
	}
	buf, err := dht.PeerEndpointDHT.GetValue(key)
	if err != nil {
		return nil, err
	}
	response := handler.Response(chainMessage.MessageType, string(buf))

	return response, nil
}

func init() {
	GetValueAction = getValueAction{}
	GetValueAction.MsgType = msgtype.GETVALUE
	handler.RegistChainMessageHandler(msgtype.GETVALUE, GetValueAction.Send, GetValueAction.Receive, GetValueAction.Response)
}
