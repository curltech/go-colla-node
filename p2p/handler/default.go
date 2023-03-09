package handler

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
)

type ProtocolMessageHandler struct {
	ProtocolID     string
	MessageHandler func(data []byte, remotePeerId string, clientId string, connectSessionId string, remoteAddr string) ([]byte, error)
}

/**
为每个消息类型注册接收和发送的处理函数，从ChainMessage中解析出消息类型，自动分发到合适的处理函数
*/
var protocolMessageHandlers = make(map[string]*ProtocolMessageHandler)

func GetProtocolMessageHandler(protocolID string) (*ProtocolMessageHandler, error) {
	fn, found := protocolMessageHandlers[protocolID]
	if found {
		return fn, nil
	} else {
		logger.Sugar.Errorf("ChainMessageHandler:%v is not exist", protocolID)

		return nil, errors.New("NotExist")
	}
}

func RegistProtocolMessageHandler(protocolID string,
	receiveHandler func(data []byte, remotePeerId string, clientId string, connectSessionId string, remoteAddr string) ([]byte, error)) {
	_, found := protocolMessageHandlers[protocolID]
	if !found {
		protocolMessageHandler := ProtocolMessageHandler{}
		protocolMessageHandler.ProtocolID = protocolID
		protocolMessageHandler.MessageHandler = receiveHandler
		protocolMessageHandlers[protocolID] = &protocolMessageHandler
	} else {
		logger.Sugar.Errorf("ReceiveHandler:%v exist", protocolID)
	}
}
