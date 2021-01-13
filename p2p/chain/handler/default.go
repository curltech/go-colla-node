package handler

import (
	"errors"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/kataras/golog"
)

type ChainMessageHandler struct {
	MsgType string
	/**
	接收函数，返回的整数：0，不关闭流；1，发送完处理结果后关闭写流，不能再写；2，发送完处理结果后重置流，不能读写，完全关闭
	*/
	ReceiveHandler  func(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error)
	SendHandler     func(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error)
	ResponseHandler func(chainMessage *msg.ChainMessage) error
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
		golog.Errorf("ReceiveHandler:%v exist", msgType)
	}
}

func GetChainMessageHandler(msgType string) (*ChainMessageHandler, error) {
	fn, found := chainMessageHandlers[msgType]
	if found {
		return fn, nil
	} else {
		golog.Errorf("ChainMessageHandler:%v is not exist", msgType)

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
	sendHandler func(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error),
	receiveHandler func(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error),
	responseHandler func(chainMessage *msg.ChainMessage) error) {
	_, found := chainMessageHandlers[msgType]
	if !found {
		chainMessageHandler := ChainMessageHandler{}
		chainMessageHandler.MsgType = msgType
		chainMessageHandler.SendHandler = sendHandler
		chainMessageHandler.ReceiveHandler = receiveHandler
		chainMessageHandler.ResponseHandler = responseHandler
		chainMessageHandlers[msgType] = &chainMessageHandler
	} else {
		golog.Errorf("ReceiveHandler:%v exist", msgType)
	}
}

////////////////////

type PCChainMessageHandler struct {
	MsgType           string
	PCReceiveHandler  func(chainMessage *msg.PCChainMessage) (interface{}, error)
	PCSendHandler     func(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error)
	PCResponseHandler func(chainMessage *msg.PCChainMessage) error
}

/**
为每个消息类型注册接收和发送的处理函数，从PCChainMessage中解析出消息类型，自动分发到合适的处理函数
*/
var pcChainMessageHandlers = make(map[string]*PCChainMessageHandler)

func registPCChainMessageHandler(msgType string, handler *PCChainMessageHandler) {
	_, found := pcChainMessageHandlers[msgType]
	if !found {
		pcChainMessageHandlers[msgType] = handler
	} else {
		golog.Errorf("ReceivePCHandler:%v exist", msgType)
	}
}

func GetPCChainMessageHandler(msgType string) (*PCChainMessageHandler, error) {
	fn, found := pcChainMessageHandlers[msgType]
	if found {
		return fn, nil
	} else {
		golog.Errorf("PCChainMessageHandler:%v is not exist", msgType)

		return nil, errors.New("NotExist")
	}
}

func RegistPCChainMessageHandler(msgType string, sendHandler func(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error), receiveHandler func(chainMessage *msg.PCChainMessage) (interface{}, error), responseHandler func(chainMessage *msg.PCChainMessage) error) {
	_, found := pcChainMessageHandlers[msgType]
	if !found {
		pcChainMessageHandler := PCChainMessageHandler{}
		pcChainMessageHandler.MsgType = msgType
		pcChainMessageHandler.PCSendHandler = sendHandler
		pcChainMessageHandler.PCReceiveHandler = receiveHandler
		pcChainMessageHandler.PCResponseHandler = responseHandler
		pcChainMessageHandlers[msgType] = &pcChainMessageHandler
	} else {
		golog.Errorf("ReceivePCHandler:%v exist", msgType)
	}
}
