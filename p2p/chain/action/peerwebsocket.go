package action

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

type peerWebsocketAction struct {
	BaseAction
}

var PeerWebsocketAction peerWebsocketAction

/**
主动发送消息
*/
func (this *peerWebsocketAction) Send(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Infof("Send %v message", this.MsgType)
	response := &msg.ChainMessage{}

	return response, nil
}

/**
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (this *peerWebsocketAction) Receive(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Infof("Receive %v message", this.MsgType)
	//wcm := chainMessage.Payload.(*msg.WebsocketChainMessage)
	//targetPeerClient := wcm.TargetPeerClient
	//targetPeerId := targetPeerClient.PeerId
	//connectSessionId := targetPeerClient.ConnectSessionId
	//connectionPool := websocket.ConnectionPool
	//connectionIndex := websocket.ConnectionIndex
	//idMap, ok := connectionIndex[targetPeerId]
	//if ok {
	//	_, ok1 := idMap[connectSessionId]
	//	if ok1 {
	//		conn, ok2 := connectionPool[connectSessionId]
	//		if ok2 {
	//			response := service.InitPCResponse(targetPeerClient, msgtype.WEBSOCKET)
	//			response.MessagePayload.Payload = wcm
	//			response, err := handler.EncryptPC(response)
	//			if err != nil {
	//				return nil, err
	//			}
	//			data, err := message.Marshal(response)
	//			if err != nil {
	//				return nil, err
	//			}
	//			message := ws.Message{
	//				Body: data,
	//			}
	//			conn.Write(message)
	//		}
	//	}
	//}

	return nil, nil
}

/**
处理返回消息
*/
func (this *peerWebsocketAction) Response(chainMessage *msg.ChainMessage) error {
	logger.Infof("Response %v message:%v", this.MsgType, chainMessage)

	return nil
}

func init() {
	PeerWebsocketAction = peerWebsocketAction{}
	PeerWebsocketAction.MsgType = msgtype.PEERWEBSOCKET
	handler.RegistChainMessageHandler(msgtype.PEERWEBSOCKET, PeerWebsocketAction.Send, PeerWebsocketAction.Receive, PeerWebsocketAction.Response)
}
