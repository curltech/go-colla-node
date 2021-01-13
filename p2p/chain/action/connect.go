package action

import (
	"context"
	"curltech.io/camsi/camsi-biz/app/websocket"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/crypto/std"
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
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	"strings"
	"sync"
	"time"
)

type connectAction struct {
	BaseAction
}

var ConnectAction connectAction

/**
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (this *connectAction) PCReceive(chainMessage *msg.PCChainMessage) (interface{}, error) {
	golog.Infof("Receive %v message", this.MsgType)
	peerClient := chainMessage.MessagePayload.Payload.(*entity.PeerClient)
	err := service1.ValidatePC(chainMessage.MessagePayload)
	if err != nil {
		return msgtype.ERROR, err
	}

	peerId := peerClient.PeerId
	clientId := peerClient.ClientId
	clientDevice := peerClient.ClientDevice
	connectAddress := peerClient.ConnectAddress
	connectPeerId := peerClient.ConnectPeerId
	connectPublicKey := peerClient.ConnectPublicKey
	connectSessionId := peerClient.ConnectSessionId
	previousPublicKeySignature := peerClient.PreviousPublicKeySignature
	signature := peerClient.Signature
	signatureData := peerClient.SignatureData
	expireDate := peerClient.ExpireDate
	connectionPool := websocket.ConnectionPool
	connectionIndex := websocket.ConnectionIndex

	// 更新连接池connectionPool & connectionIndex
	_, ok := connectionPool[connectSessionId]
	if ok {
		idMap, ok := connectionIndex[peerId]
		if !ok {
			connectionIndex[peerId] = map[string]string{connectSessionId: clientId}
		} else {
			_, ok := idMap[connectSessionId]
			if !ok {
				idMap[connectSessionId] = clientId
			}
		}
	}

	// 返回peerClient信息
	key := ns.GetPeerClientKey(peerId)
	var pcs []*entity.PeerClient
	if config.Libp2pParams.FaultTolerantLevel == 0 {
		pcs = make([]*entity.PeerClient, 0)
		recvdVals, err := dht.PeerEndpointDHT.GetValues(key, config.Libp2pParams.Nvals)
		if err != nil {
			return msgtype.ERROR, err
		}
		for _, recvdVal := range recvdVals {
			pcArr := make([]*entity.PeerClient, 0)
			err = message.TextUnmarshal(string(recvdVal.Val), &pcArr)
			if err != nil {
				golog.Errorf("failed to TextUnmarshal PeerClient value: %v, err: %v", recvdVal.Val, err)
				return msgtype.ERROR, err
			}
			for _, pc := range pcArr {
				pcs = append(pcs, pc)
			}
		}
	} else if config.Libp2pParams.FaultTolerantLevel == 1 {
		// 查询删除local历史记录
		locals, err := service1.GetLocalPCs(ns.PeerClient_KeyKind, peerId, "", "")
		if err != nil {
			return msgtype.ERROR, err
		}
		if len(locals) > 0 {
			service.GetPeerClientService().Delete(locals, "")
		}
		// 查询non-local历史记录
		recvdVals, err := dht.PeerEndpointDHT.GetValues(key, config.Libp2pParams.Nvals)
		if err != nil {
			return msgtype.ERROR, err
		}
		// 恢复local历史记录
		err = service1.PutLocalPCs(locals)
		if err != nil {
			return msgtype.ERROR, err
		}
		// 更新local历史记录
		for _, recvdVal := range recvdVals {
			pcArr := make([]*entity.PeerClient, 0)
			err = message.TextUnmarshal(string(recvdVal.Val), &pcArr)
			if err != nil {
				golog.Errorf("failed to TextUnmarshal PeerClient value: %v, err: %v", recvdVal.Val, err)
				return msgtype.ERROR, err
			}
			err = service1.PutLocalPCs(pcArr)
			if err != nil {
				golog.Errorf("failed to PutLocalPCs PeerClient value: %v, err: %v", recvdVal.Val, err)
				return msgtype.ERROR, err
			}
		}
		// 再次查询local历史记录
		pcs, err = service1.GetLocalPCs(ns.PeerClient_KeyKind, peerId, "", "")
		if err != nil {
			return msgtype.ERROR, err
		}
	}

	currentTime := time.Now()
	var isNew bool = true
	if len(pcs) > 0 {
		for _, pc := range pcs {
			// 更新信息
			if pc.ClientId == clientId {
				isNew = false
				pc.LastAccessTime = &currentTime
				pc.ActiveStatus = entity.ActiveStatus_Up
				pc.ConnectAddress = connectAddress
				pc.ConnectPeerId = connectPeerId
				pc.ConnectPublicKey = connectPublicKey
				pc.ConnectSessionId = connectSessionId
				pc.PreviousPublicKeySignature = previousPublicKeySignature
				pc.Signature = signature
				pc.SignatureData = signatureData
				pc.ExpireDate = expireDate
				pc.Mobile = std.EncodeBase64(std.Hash(pc.Mobile, "sha3_256"))
				pc.PublicKey = peerClient.PublicKey // 可能resetKey
				err = service1.PutPCs(pc)
				if err != nil {
					return msgtype.ERROR, err
				}
				break
			}
		}
	}
	// 新增信息
	if isNew {
		peerClient.LastAccessTime = &currentTime
		peerClient.ActiveStatus = entity.ActiveStatus_Up
		peerClient.Mobile = std.EncodeBase64(std.Hash(peerClient.Mobile, "sha3_256"))
		err := service1.PutPCs(peerClient)
		if err != nil {
			return msgtype.ERROR, err
		}
		if pcs == nil {
			pcs = make([]*entity.PeerClient, 0)
		}
		pcs = append(pcs, peerClient)
	}
	if len(pcs) > 1 {
		for _, pc := range pcs {
			// 同种设备实例踢下线
			if pc.ClientId != clientId {
				if pc.ClientDevice == clientDevice && pc.ActiveStatus == entity.ActiveStatus_Up {
					wcm := msg.WebsocketChainMessage{}
					wcm.SrcPeerClient = peerClient
					wcm.TargetPeerClient = pc
					wcm.MessageType = msgtype.SOCKET_LOGOUT
					payload := make(map[string]interface{}, 0)
					payload["clientId"] = clientId
					wcm.Payload = &payload
					if pc.ConnectPeerId == global.Global.MyselfPeer.PeerId {
						targetPeerId := pc.PeerId
						connectSessionId := pc.ConnectSessionId
						connectionPool := websocket.ConnectionPool
						connectionIndex := websocket.ConnectionIndex
						idMap, ok := connectionIndex[targetPeerId]
						if ok {
							_, ok1 := idMap[connectSessionId]
							if ok1 {
								conn, ok2 := connectionPool[connectSessionId]
								if ok2 {
									response := service1.InitPCResponse(pc, msgtype.WEBSOCKET)
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
						cm.Payload = wcm
						cm.ConnectPeerId = pc.ConnectPeerId
						addrPort := strings.Split(pc.ConnectAddress, ":")
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

	global.Global.PeerEndpointDHT.RoutingTable().Print()

	// 返回peerEndPoint信息
	peers := make([]*entity.PeerEndpoint, 0)
	// 添加自己
	myself, err := message.Marshal(global.Global.MyselfPeer)
	if err != nil {
		return msgtype.ERROR, err
	}
	peerEndpoint := entity.PeerEndpoint{}
	err = message.Unmarshal(myself, &peerEndpoint)
	if err != nil {
		return msgtype.ERROR, err
	}
	peers = append(peers, &peerEndpoint)
	// 添加最近节点
	pchan, err := dht.PeerEndpointDHT.GetClosestPeers(key)
	if err != nil {
		return msgtype.ERROR, err
	}
	wg := sync.WaitGroup{}
	for p := range pchan {
		wg.Add(1)
		go func(p peer.ID) {
			ctx, cancel := context.WithCancel(global.Global.Context)
			defer cancel()
			defer wg.Done()
			golog.Infof("ClosestPeers-PeerId: %v", p.Pretty())
			routing.PublishQueryEvent(ctx, &routing.QueryEvent{
				Type: routing.Value,
				ID:   p,
			})
			k := ns.GetPeerEndpointKey(p.Pretty())
			recvdVals, err := dht.PeerEndpointDHT.GetValues(k, config.Libp2pParams.Nvals)
			if err != nil {
				golog.Errorf("failed to GetValues by PeerEndpoint key: %v, err: %v", k, err)
			} else {
				for _, recvdVal := range recvdVals {
					entities := make([]*entity.PeerEndpoint, 0)
					err = message.TextUnmarshal(string(recvdVal.Val), &entities)
					if err != nil {
						golog.Errorf("failed to TextUnmarshal PeerEndpoint value: %v, err: %v", recvdVal.Val, err)
					} else {
						if len(entities) > 0 {
							peer := entities[0]
							golog.Infof("PeerEndpoint: %v", peer.PeerId+"-"+peer.DiscoveryAddress)
							peers = append(peers, peer)
						}
					}
				}
			}
		}(p)
	}
	wg.Wait()

	return []interface{}{peers, pcs}, nil
}

/**
处理返回消息
*/
func (this *connectAction) PCResponse(chainMessage *msg.PCChainMessage) error {
	golog.Infof("Response %v message:%v", this.MsgType, chainMessage)

	return nil
}

func init() {
	ConnectAction = connectAction{}
	ConnectAction.MsgType = msgtype.CONNECT
	handler.RegistChainMessageHandler(msgtype.CONNECT, ConnectAction.Send, ConnectAction.Receive, ConnectAction.Response)
	handler.RegistPCChainMessageHandler(msgtype.CONNECT, ConnectAction.Send, ConnectAction.PCReceive, ConnectAction.PCResponse)
}
