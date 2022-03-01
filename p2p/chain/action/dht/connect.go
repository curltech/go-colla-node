package dht

import (
	"context"
	"errors"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/crypto/std"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	"sync"
	"time"
)

type connectAction struct {
	action.BaseAction
}

var ConnectAction connectAction

/**
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (this *connectAction) Receive(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Sugar.Infof("Receive %v message", this.MsgType)
	start := time.Now()
	var response *msg.ChainMessage = nil
	v := chainMessage.Payload
	peerClient, ok := v.(*entity.PeerClient)
	if !ok {
		response = handler.Error(chainMessage.MessageType, errors.New("PayloadDataTypeError"))
		return response, nil
	}
	err := service.GetPeerClientService().Validate(peerClient)
	if err != nil {
		response = handler.Error(chainMessage.MessageType, err)
		return response, nil
	}
	peerClient.ConnectSessionId = chainMessage.ConnectSessionId

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

	// 返回peerClient信息
	key := ns.GetPeerClientKey(peerId)
	var pcs []*entity.PeerClient
	if config.Libp2pParams.FaultTolerantLevel == 0 {
		pcs = make([]*entity.PeerClient, 0)
		recvdVals, err := dht.PeerEndpointDHT.GetValues(key, config.Libp2pParams.Nvals)
		if err != nil {
			response = handler.Error(chainMessage.MessageType, err)
			return response, nil
		}
		for _, recvdVal := range recvdVals {
			pcArr := make([]*entity.PeerClient, 0)
			err = message.TextUnmarshal(string(recvdVal.Val), &pcArr)
			if err != nil {
				logger.Sugar.Errorf("failed to TextUnmarshal PeerClient value: %v, err: %v", recvdVal.Val, err)
				response = handler.Error(chainMessage.MessageType, err)
				return response, nil
			}
			for _, pc := range pcArr {
				pcs = append(pcs, pc)
			}
		}
	} else if config.Libp2pParams.FaultTolerantLevel == 1 {
		
	} else if config.Libp2pParams.FaultTolerantLevel == 2 {
		// 查询删除local历史记录
		locals, err := service.GetPeerClientService().GetLocals(key, "")
		if err != nil {
			response = handler.Error(chainMessage.MessageType, err)
			return response, nil
		}
		if len(locals) > 0 {
			service.GetPeerClientService().Delete(locals, "")
		}
		// 查询non-local历史记录
		recvdVals, err := dht.PeerEndpointDHT.GetValues(key, config.Libp2pParams.Nvals)
		if err != nil {
			response = handler.Error(chainMessage.MessageType, err)
			return response, nil
		}
		// 恢复local历史记录
		err = service.GetPeerClientService().PutLocals(locals)
		if err != nil {
			response = handler.Error(chainMessage.MessageType, err)
			return response, nil
		}
		// 更新local历史记录
		for _, recvdVal := range recvdVals {
			pcArr := make([]*entity.PeerClient, 0)
			err = message.TextUnmarshal(string(recvdVal.Val), &pcArr)
			if err != nil {
				logger.Sugar.Errorf("failed to TextUnmarshal PeerClient value: %v, err: %v", recvdVal.Val, err)
				response = handler.Error(chainMessage.MessageType, err)
				return response, nil
			}
			err = service.GetPeerClientService().PutLocals(pcArr)
			if err != nil {
				logger.Sugar.Errorf("failed to PutLocalPCs PeerClient value: %v, err: %v", recvdVal.Val, err)
				response = handler.Error(chainMessage.MessageType, err)
				return response, nil
			}
		}
		// 再次查询local历史记录
		pcs, err = service.GetPeerClientService().GetLocals(key, "")
		if err != nil {
			response = handler.Error(chainMessage.MessageType, err)
			return response, nil
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
				pc.Mobile = std.EncodeBase64(std.Hash(peerClient.Mobile, "sha3_256"))
				pc.PublicKey = peerClient.PublicKey // 可能resetKey
				pc.LastUpdateTime = peerClient.LastUpdateTime
				pc.DeviceToken = peerClient.DeviceToken
				pc.Language = peerClient.Language
				/*err := service.GetPeerClientService().PutValues(pc)
				if err != nil {
					response = handler.Error(chainMessage.MessageType, err)
					return response, nil
				}*/
				go service.GetPeerClientService().PutValues(pc)
				break
			}
		}
	}
	// 新增信息
	if isNew {
		peerClient.LastAccessTime = &currentTime
		peerClient.ActiveStatus = entity.ActiveStatus_Up
		peerClient.Mobile = std.EncodeBase64(std.Hash(peerClient.Mobile, "sha3_256"))
		/*err := service.GetPeerClientService().PutValues(peerClient)
		if err != nil {
			response = handler.Error(chainMessage.MessageType, err)
			return response, nil
		}*/
		go service.GetPeerClientService().PutValues(peerClient)
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
					chat := make(map[string]interface{}, 0)
					chat["type"] = msgtype.CHAT_LOGOUT
					chat["srcClientId"] = peerClient.ClientId
					chat["srcPeerId"] = peerClient.PeerId
					chat["srcClientDevice"] = peerClient.ClientDevice
					chat["srcClientType"] = peerClient.ClientType
					chat["createDate"] = &currentTime
					var connectSessionId string = ""
					if global.IsMyself(pc.ConnectPeerId) {
						connectSessionId = pc.ConnectSessionId
					}
					go ChatAction.Chat("", chat, pc.PeerId, connectSessionId)
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
		response = handler.Error(chainMessage.MessageType, err)
		return response, nil
	}
	peerEndpoint := entity.PeerEndpoint{}
	err = message.Unmarshal(myself, &peerEndpoint)
	if err != nil {
		response = handler.Error(chainMessage.MessageType, err)
		return response, nil
	}
	peers = append(peers, &peerEndpoint)
	// 添加最近节点
	parray, err := dht.PeerEndpointDHT.GetClosestPeers(key)
	if err != nil {
		logger.Sugar.Errorf("failed to GetClosestPeers by key: %v, err: %v", key, err)
	} else {
		wg := sync.WaitGroup{}
		for _, p := range parray {
			wg.Add(1)
			go func(p peer.ID) {
				ctx, cancel := context.WithCancel(global.Global.Context)
				defer cancel()
				defer wg.Done()
				logger.Sugar.Infof("ClosestPeers-PeerId: %v", p.Pretty())
				routing.PublishQueryEvent(ctx, &routing.QueryEvent{
					Type: routing.Value,
					ID:   p,
				})
				k := ns.GetPeerEndpointKey(p.Pretty())
				recvdVals, err := dht.PeerEndpointDHT.GetValues(k, config.Libp2pParams.Nvals)
				if err != nil {
					logger.Sugar.Errorf("failed to GetValues by PeerEndpoint key: %v, err: %v", k, err)
				} else {
					for _, recvdVal := range recvdVals {
						entities := make([]*entity.PeerEndpoint, 0)
						err = message.TextUnmarshal(string(recvdVal.Val), &entities)
						if err != nil {
							logger.Sugar.Errorf("failed to TextUnmarshal PeerEndpoint value: %v, err: %v", recvdVal.Val, err)
						} else {
							if len(entities) > 0 {
								peer := entities[0]
								logger.Sugar.Infof("PeerEndpoint: %v", peer.PeerId+"-"+peer.DiscoveryAddress)
								peers = append(peers, peer)
							}
						}
					}
				}
			}(p)
		}
		wg.Wait()
	}

	response = handler.Response(chainMessage.MessageType, []interface{}{peers, pcs})
	end := time.Now()
	logger.Sugar.Infof("==============================connect time: %v", end.Sub(start))
	return response, nil
}

func init() {
	ConnectAction = connectAction{}
	ConnectAction.MsgType = msgtype.CONNECT
	handler.RegistChainMessageHandler(msgtype.CONNECT, ConnectAction.Send, ConnectAction.Receive, ConnectAction.Response)
}
