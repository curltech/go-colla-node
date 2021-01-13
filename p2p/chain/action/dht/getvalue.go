package dht

import (
	"errors"
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
在chain目录下的采用自定义protocol "/chain"的方式自己实现的功能
*/
func (this *getValueAction) GetValue(peerId string, payloadType string, data interface{}) (interface{}, error) {
	chainMessage := msg.ChainMessage{}
	chainMessage.Payload = data
	chainMessage.ConnectPeerId = peerId
	chainMessage.PayloadType = payloadType
	chainMessage.MessageType = msgtype.GETVALUE
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

func (this *getValueAction) Receive(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
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
