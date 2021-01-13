package action

import (
	"github.com/curltech/go-colla-biz/app/websocket"
	"github.com/curltech/go-colla-core/crypto/std"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/chain/service"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/kataras/golog"
	"time"
)

type disconnectAction struct {
	BaseAction
}

var DisconnectAction disconnectAction

/**
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (this *disconnectAction) PCReceive(chainMessage *msg.PCChainMessage) (interface{}, error) {
	golog.Infof("Receive %v message", this.MsgType)
	peerClient := chainMessage.MessagePayload.Payload.(*entity.PeerClient)
	err := service.ValidatePC(chainMessage.MessagePayload)
	if err != nil {
		return msgtype.ERROR, err
	}

	peerId := peerClient.PeerId
	clientId := peerClient.ClientId
	connectSessionId := peerClient.ConnectSessionId
	connectionPool := websocket.ConnectionPool
	connectionIndex := websocket.ConnectionIndex

	// 更新连接池connectionPool & connectionIndex
	_, ok := connectionPool[connectSessionId]
	if ok {
		idMap, ok := connectionIndex[peerId]
		if ok {
			cId, ok := idMap[connectSessionId]
			if ok {
				if cId == clientId {
					delete(idMap, cId)
				}
			}
		}
	}

	// 更新信息
	currentTime := time.Now()
	peerClient.LastAccessTime = &currentTime
	peerClient.ActiveStatus = entity.ActiveStatus_Down
	peerClient.Mobile = std.EncodeBase64(std.Hash(peerClient.Mobile, "sha3_256"))
	err = service.PutPCs(peerClient)
	if err != nil {
		return msgtype.ERROR, err
	}

	return msgtype.OK, nil
}

/**
处理返回消息
*/
func (this *disconnectAction) PCResponse(chainMessage *msg.PCChainMessage) error {
	golog.Infof("Response %v message:%v", this.MsgType, chainMessage)

	return nil
}

func init() {
	DisconnectAction = disconnectAction{}
	DisconnectAction.MsgType = msgtype.DISCONNECT
	handler.RegistChainMessageHandler(msgtype.DISCONNECT, DisconnectAction.Send, DisconnectAction.Receive, DisconnectAction.Response)
	handler.RegistPCChainMessageHandler(msgtype.DISCONNECT, DisconnectAction.Send, DisconnectAction.PCReceive, DisconnectAction.PCResponse)
}
