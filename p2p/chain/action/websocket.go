package action

import (
	"github.com/curltech/go-colla-biz/app/websocket"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	service1 "github.com/curltech/go-colla-node/p2p/chain/service"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/kataras/golog"
	ws "github.com/kataras/iris/v12/websocket"
	"strings"
)

type websocketAction struct {
	BaseAction
}

var WebsocketAction websocketAction

/**
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (this *websocketAction) PCReceive(chainMessage *msg.PCChainMessage) (interface{}, error) {
	golog.Infof("Receive %v message", this.MsgType)
	wcm := chainMessage.MessagePayload.Payload.(*msg.WebsocketChainMessage)
	targetPeerId := wcm.TargetPeerClient.PeerId

	peerClients := make([]*entity.PeerClient, 0)
	key := ns.GetPeerClientKey(targetPeerId)
	if config.Libp2pParams.FaultTolerantLevel == 0 {
		recvdVals, err := dht.PeerEndpointDHT.GetValues(key, config.Libp2pParams.Nvals)
		if err != nil {
			return msgtype.ERROR, err
		}
		for _, recvdVal := range recvdVals {
			pcs := make([]*entity.PeerClient, 0)
			err = message.TextUnmarshal(string(recvdVal.Val), &pcs)
			if err != nil {
				golog.Errorf("failed to TextUnmarshal PeerClient value: %v, err: %v", recvdVal.Val, err)
				return msgtype.ERROR, err
			}
			for _, pc := range pcs {
				peerClients = append(peerClients, pc)
			}
		}
	} else if config.Libp2pParams.FaultTolerantLevel == 1 {
		// 查询删除local记录
		locals, err := service1.GetLocalPCs(ns.PeerClient_KeyKind, targetPeerId, "", "")
		if err != nil {
			return msgtype.ERROR, err
		}
		if len(locals) > 0 {
			for _, local := range locals {
				peerClients = append(peerClients, local)
			}
			service.GetPeerClientService().Delete(locals, "")
		}
		// 查询non-local记录
		recvdVals, err := dht.PeerEndpointDHT.GetValues(key, config.Libp2pParams.Nvals)
		if err != nil {
			return msgtype.ERROR, err
		}
		// 恢复local记录
		err = service1.PutLocalPCs(locals)
		if err != nil {
			return msgtype.ERROR, err
		}
		// 整合记录
		for _, recvdVal := range recvdVals {
			pcs := make([]*entity.PeerClient, 0)
			err = message.TextUnmarshal(string(recvdVal.Val), &pcs)
			if err != nil {
				golog.Errorf("failed to TextUnmarshal PeerClient value: %v, err: %v", recvdVal.Val, err)
				return msgtype.ERROR, err
			}
			for _, pc := range pcs {
				peerClients = append(peerClients, pc)
			}
		}
	}
	// 转发
	if len(peerClients) > 0 {
		processedPeerIds := make(map[string]bool, 0)
		for _, peerClient := range peerClients {
			if peerClient.ActiveStatus == entity.ActiveStatus_Up {
				peerIdClientId := peerClient.PeerId + peerClient.ClientId
				if processedPeerIds[peerIdClientId] != true {
					processedPeerIds[peerIdClientId] = true
					connectPeerId := peerClient.ConnectPeerId
					if connectPeerId == global.Global.MyselfPeer.PeerId {
						targetPeerId := peerClient.PeerId
						connectSessionId := peerClient.ConnectSessionId
						connectionPool := websocket.ConnectionPool
						connectionIndex := websocket.ConnectionIndex
						idMap, ok := connectionIndex[targetPeerId]
						if ok {
							_, ok1 := idMap[connectSessionId]
							if ok1 {
								conn, ok2 := connectionPool[connectSessionId]
								if ok2 {
									response := service1.InitPCResponse(peerClient, msgtype.WEBSOCKET)
									response.MessagePayload.Payload = wcm
									response, err := handler.EncryptPC(response)
									if err != nil {
										return nil, err
									}
									data, err := message.Marshal(response)
									if err != nil {
										return nil, err
									}
									message := ws.Message{
										Body: data,
									}
									conn.Write(message)
								}
							}
						}
					} else {
						cm := msg.ChainMessage{}
						wcm.TargetPeerClient = peerClient
						cm.Payload = wcm
						cm.ConnectPeerId = peerClient.ConnectPeerId
						addrPort := strings.Split(peerClient.ConnectAddress, ":")
						cm.ConnectAddress = addrPort[0]
						cm.PayloadType = handler.PayloadType_WebsocketChainMessage
						cm.MessageType = msgtype.PEERWEBSOCKET
						cm.MessageDirect = msgtype.MsgDirect_Request
						this.Send(&cm)
					}
				}
			}
		}
	}

	return msgtype.OK, nil
}

/**
处理返回消息
*/
func (this *websocketAction) PCResponse(chainMessage *msg.PCChainMessage) error {
	golog.Infof("Response %v message:%v", this.MsgType, chainMessage)

	return nil
}

func init() {
	WebsocketAction = websocketAction{}
	WebsocketAction.MsgType = msgtype.WEBSOCKET
	handler.RegistChainMessageHandler(msgtype.WEBSOCKET, WebsocketAction.Send, WebsocketAction.Receive, WebsocketAction.Response)
	handler.RegistPCChainMessageHandler(msgtype.WEBSOCKET, WebsocketAction.Send, WebsocketAction.PCReceive, WebsocketAction.PCResponse)
}
