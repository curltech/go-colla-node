package receiver

import (
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/chain/handler/sender"
	msg1 "github.com/curltech/go-colla-node/p2p/msg/entity"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"time"
)

// Dispatch 接收ChainMessage报文处理的入口，无论何种方式（libp2p,wss,stdhttp）发送过来
// 的任何ChainMessage类型都统一在此处理分发
func Dispatch(chainMessage *msg1.ChainMessage) (*msg1.ChainMessage, error) {
	targetPeerId := chainMessage.TargetPeerId
	//目标是自己，则对payload解密，否则直接转发
	if targetPeerId == "" || global.IsMyself(targetPeerId) {
		handler.Decrypt(chainMessage)
	} else {
		go sender.RelaySend(chainMessage)
		response := handler.Response(chainMessage.MessageType, time.Now())
		return response, nil
	}
	typ := chainMessage.MessageType
	direct := chainMessage.MessageDirect
	chainMessageHandler, err := handler.GetChainMessageHandler(string(typ))
	var response *msg1.ChainMessage
	if err != nil {
		response = handler.Error(typ, err)
		return response, err
	}
	//分发到对应注册好的处理器，主要是Receive和Response方法
	if direct == msgtype.MsgDirect_Request {
		response, err = chainMessageHandler.ReceiveHandler(chainMessage)
		if err != nil {
			response = handler.Error(typ, err)
			return response, err
		} else if response == nil {
			response = handler.Ok(typ)

			return response, nil
		}
	} else if direct == msgtype.MsgDirect_Response {
		chainMessageHandler.ResponseHandler(chainMessage)
	}

	return response, nil
}
