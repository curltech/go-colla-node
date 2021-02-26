package dht

import (
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

type pingAction struct {
	action.BaseAction
}

var PingAction pingAction

/**
在chain目录下的采用自定义protocol "/chain"的方式自己实现的功能
Ping只是一个演示，适合点对点的通信，这种方式灵活度高，但是需要自己实现全网遍历的功能
chat就可以采用这种方式
*/
func (this *pingAction) Ping(peerId string, data interface{}, targetPeerId string) (interface{}, error) {
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

func init() {
	PingAction = pingAction{}
	PingAction.MsgType = msgtype.PING
	handler.RegistChainMessageHandler(msgtype.PING, PingAction.Send, PingAction.Receive, PingAction.Response)
}
