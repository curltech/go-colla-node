package receiver

import (
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/pubsub"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	p2phandler "github.com/curltech/go-colla-node/p2p/handler"
	msg1 "github.com/curltech/go-colla-node/p2p/msg/entity"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/curltech/go-colla-node/transport/websocket/stdhttp"
	"net/http"
)

// HandleChainMessage libp2p,wss,https的处理handler，将原始数据还原成ChainMessage，
// 然后根据消息类型进行分支处理
// 对于libp2p，可以提前获取远程peerId，remotePeerId不为空，已经建立了remotePeerId和sessId的连接池
// 用于信息返回
// 对于wss，remotePeerId为空，必须从ChainMessage中获取，再建立连接池
// 对于https，remotePeerId为空，必须从ChainMessage中获取，无须连接池，不能异步返回信息
func ReceiveRaw(data []byte, remotePeerId string, clientId string, connectSessionId string, remoteAddr string) ([]byte, error) {
	//defer func() {
	//	if p := recover(); p != nil {
	//		logger.Sugar.Errorf("recover receiveRaw:%s\r\n", p)
	//	}
	//}()
	var response *msg1.ChainMessage
	chainMessage := &msg1.ChainMessage{}
	err := message.Unmarshal(data, chainMessage)
	if err != nil {
		response = handler.Error(msgtype.ERROR, err)
		goto responseProcess
	}
	clientId = chainMessage.SrcClientId
	if remotePeerId == "" {
		remotePeerId = chainMessage.SrcPeerId
	}
	if chainMessage.SrcPeerId == "" {
		chainMessage.SrcPeerId = remotePeerId
	}
	if chainMessage.SrcConnectPeerId == "" {
		chainMessage.SrcConnectPeerId = string(global.Global.PeerId)
	}
	if chainMessage.SrcConnectSessionId == "" {
		chainMessage.SrcConnectSessionId = connectSessionId
	}
	chainMessage.ConnectSessionId = connectSessionId
	PutPeeClientId(connectSessionId, remotePeerId, clientId)
	response, err = Dispatch(chainMessage)

responseProcess:
	if err != nil {
		if response != nil {
			response.StatusCode = http.StatusInternalServerError
		} else {
			response = handler.Error(chainMessage.MessageType, err)
		}
	} else {
		if response != nil {
			_, err = handler.Encrypt(response)
			if err != nil {
				response = handler.Error(chainMessage.MessageType, err)
			} else {
				response.StatusCode = http.StatusOK
			}
		} else {
			response = handler.Ok(chainMessage.MessageType)
		}
	}

	handler.SetResponse(chainMessage, response)
	data, _ = message.Marshal(response)

	return data, nil
}

func init() {
	//注册websocket的消息处理
	stdhttp.RegistMessageHandler(ReceiveRaw)
	stdhttp.RegistDisconnectedHandler(HandleDisconnected)
	//注册libp2p协议的消息处理
	p2phandler.RegistProtocolMessageHandler(config.P2pParams.ChainProtocolID, ReceiveRaw)
	//注册libp2p订阅的消息处理
	pubsub.RegistMessageHandler(ReceiveRaw)
}
