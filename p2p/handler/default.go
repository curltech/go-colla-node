package handler

import (
	"errors"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/pipe"
	"github.com/curltech/go-colla-node/p2p/chain/handler/receiver"
)

type ProtocolMessageHandler struct {
	ProtocolID     string
	ReceiveHandler func(data []byte, p *pipe.Pipe) ([]byte, error)
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
		logger.Errorf("ChainMessageHandler:%v is not exist", protocolID)

		return nil, errors.New("NotExist")
	}
}

func RegistProtocolMessageHandler(protocolID string,
	receiveHandler func(data []byte, p *pipe.Pipe) ([]byte, error)) {
	_, found := protocolMessageHandlers[protocolID]
	if !found {
		protocolMessageHandler := ProtocolMessageHandler{}
		protocolMessageHandler.ProtocolID = protocolID
		protocolMessageHandler.ReceiveHandler = receiveHandler
		protocolMessageHandlers[protocolID] = &protocolMessageHandler
	} else {
		logger.Errorf("ReceiveHandler:%v exist", protocolID)
	}
}

func init() {
	RegistProtocolMessageHandler(config.P2pParams.ChainProtocolID, receiver.HandleChainMessage)
}
