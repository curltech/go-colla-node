package handler

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/p2p/msg/entity"
)

type ChainMessageHandler struct {
	MsgType string
	/**
	接收函数，返回的整数：0，不关闭流；1，发送完处理结果后关闭写流，不能再写；2，发送完处理结果后重置流，不能读写，完全关闭
	*/
	ReceiveHandler  func(chainMessage *entity.ChainMessage) (*entity.ChainMessage, error)
	SendHandler     func(chainMessage *entity.ChainMessage) (*entity.ChainMessage, error)
	ResponseHandler func(chainMessage *entity.ChainMessage) error
}

/**
为每个消息类型注册接收和发送的处理函数，从ChainMessage中解析出消息类型，自动分发到合适的处理函数
*/
var chainMessageHandlers = make(map[string]*ChainMessageHandler)

func registChainMessageHandler(msgType string, handler *ChainMessageHandler) {
	_, found := chainMessageHandlers[msgType]
	if !found {
		chainMessageHandlers[msgType] = handler
	} else {
		logger.Sugar.Errorf("ReceiveHandler:%v exist", msgType)
	}
}

func GetChainMessageHandler(msgType string) (*ChainMessageHandler, error) {
	fn, found := chainMessageHandlers[msgType]
	if found {
		return fn, nil
	} else {
		logger.Sugar.Errorf("ChainMessageHandler:%v is not exist", msgType)

		return nil, errors.New("NotExist")
	}
}

/**
注册各action的消息处理函数
*/
func RegistChainMessageHandler(msgType string,
	/**
	  发送函数，
	*/
	sendHandler func(chainMessage *entity.ChainMessage) (*entity.ChainMessage, error),
	receiveHandler func(chainMessage *entity.ChainMessage) (*entity.ChainMessage, error),
	responseHandler func(chainMessage *entity.ChainMessage) error) {
	_, found := chainMessageHandlers[msgType]
	if !found {
		chainMessageHandler := ChainMessageHandler{}
		chainMessageHandler.MsgType = msgType
		chainMessageHandler.SendHandler = sendHandler
		chainMessageHandler.ReceiveHandler = receiveHandler
		chainMessageHandler.ResponseHandler = responseHandler
		chainMessageHandlers[msgType] = &chainMessageHandler
	} else {
		logger.Sugar.Errorf("ReceiveHandler:%v exist", msgType)
	}
}
