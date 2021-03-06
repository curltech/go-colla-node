package service

import (
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	msg1 "github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

/**
接收ChainMessage报文处理的入口，无论何种方式发送过来的的任何chain消息类型都统一在此处理分发
*/
func Receive(chainMessage *msg1.ChainMessage) (*msg1.ChainMessage, error) {
	handler.Decrypt(chainMessage)
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
