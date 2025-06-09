package dht

import (
	"context"
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	entity2 "github.com/curltech/go-colla-node/p2p/msg/entity"
	"github.com/curltech/go-colla-node/p2p/msg/service/biz"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"sync"
	"time"
)

type connectAction struct {
	action.BaseAction
}

var ConnectAction connectAction

// Receive 接收Connect消息进行处理，保存peerclient，返回peerclients（类似findclient）
func (conn *connectAction) Receive(chainMessage *entity2.ChainMessage) (*entity2.ChainMessage, error) {
	var response *entity2.ChainMessage = nil
	v := chainMessage.Payload
	peerClient, ok := v.(*entity.PeerClient)
	if !ok {
		return response, errors.New("PayloadDataTypeError")
	}
	err := service.GetPeerClientService().Validate(peerClient)
	if err != nil {
		return response, err
	}
	peerClient.ConnectSessionId = chainMessage.SrcConnectSessionId
	peerClient.ConnectPeerId = chainMessage.SrcConnectPeerId
	peerClient.ConnectAddress = chainMessage.SrcConnectAddress
	currentTime := time.Now()
	peerClient.LastAccessTime = &currentTime
	err = service.GetPeerClientService().PutValues(peerClient)
	if err != nil {
		return response, err
	}
	logger.Sugar.Infof("peer connected successfully and put peer client, peerId: %v, connectSessionId: %v, activeStatus: %v", peerClient.PeerId, peerClient.ConnectSessionId, peerClient.ActiveStatus)
	go conn.returnPeerEndpoint(chainMessage)
	peerId := peerClient.PeerId
	activeStatus := peerClient.ActiveStatus
	//对连接的客户端，发回保存的转发消息
	if activeStatus == entity.ActiveStatus_Up {
		go func() {
			err := biz.RelaySend(peerClient)
			if err != nil {

			}
		}()
	}
	peerClients, err := service.GetPeerClientService().GetValues(peerId, "", "", "")
	if err != nil {
		response = handler.Error(chainMessage.MessageType, err)
	} else {
		response = handler.Response(chainMessage.MessageType, peerClients)
		response.PayloadType = handler.PayloadType_PeerClients
		response.TargetPeerId = peerId
		response.ConnectPeerId = peerId
		response.NeedCompress = false
	}
	return response, nil
}

// 返回节点以及附近节点的列表
func (conn *connectAction) returnPeerEndpoint(chainMessage *entity2.ChainMessage) {
	var response *entity2.ChainMessage = nil
	// 返回peerEndPoint信息
	peers := make([]*entity.PeerEndpoint, 0)
	// 添加自己
	myself, err := message.Marshal(global.Global.MyselfPeer)
	if err != nil {
		response = handler.Error(msgtype.FINDPEER, err)
	}
	peerEndpoint := entity.PeerEndpoint{}
	err = message.Unmarshal(myself, &peerEndpoint)
	if err != nil {
		response = handler.Error(msgtype.FINDPEER, err)
	}
	peers = append(peers, &peerEndpoint)
	var key = ns.GetPeerEndpointKey(peerEndpoint.PeerId)
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
				logger.Sugar.Infof("ClosestPeers-PeerId: %v", p.String())
				routing.PublishQueryEvent(ctx, &routing.QueryEvent{
					Type: routing.Value,
					ID:   p,
				})
				k := ns.GetPeerEndpointKey(p.String())
				recvdVals, err := dht.PeerEndpointDHT.GetValues(k)
				if err != nil {
					logger.Sugar.Errorf("failed to GetValues by PeerEndpoint key: %v, err: %v", k, err)
				} else {
					for _, recvdVal := range recvdVals {
						entities := make([]*entity.PeerEndpoint, 0)
						err = message.TextUnmarshal(string(recvdVal), &entities)
						if err != nil {
							logger.Sugar.Errorf("failed to TextUnmarshal PeerEndpoint value: %v, err: %v", recvdVal, err)
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

	response = handler.Response(msgtype.FINDPEER, peers)
	response.PayloadType = handler.PayloadType_PeerEndpoints
	if err == nil {
		handler.SetResponse(chainMessage, response)
		response.NeedCompress = false
	}
	_, _ = conn.Send(response)
}

func init() {
	ConnectAction = connectAction{}
	ConnectAction.MsgType = msgtype.CONNECT
	handler.RegistChainMessageHandler(msgtype.CONNECT, ConnectAction.Send, ConnectAction.Receive, ConnectAction.Response)
}
