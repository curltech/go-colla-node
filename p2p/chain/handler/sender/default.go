package sender

import (
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/libp2p/pipe/handler"
	"github.com/curltech/go-colla-node/libp2p/pubsub"
	"github.com/curltech/go-colla-node/libp2p/util"
	handler1 "github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	msg1 "github.com/curltech/go-colla-node/p2p/msg/entity"
	service2 "github.com/curltech/go-colla-node/p2p/msg/service"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/curltech/go-colla-node/transport/websocket/stdhttp"
	"github.com/gorilla/websocket"
	errors2 "github.com/pkg/errors"
	"strings"
)

// Send 发送ChainMessage消息的唯一方法
// 1.找出发送的目标地址和方式
// 2.根据情况处理校验，加密，压缩等
// 3.建立合适的通道并发送，比如libp2p的Pipe并Write消息流
// 4.等待即时的返回，校验，解密，解压缩等
func Send(msg *msg1.ChainMessage) (*msg1.ChainMessage, error) {
	_, _ = handler1.Encrypt(msg)

	return RelaySend(msg)
}

// DirectSend 定位器之间直接发送方法
func DirectSend(msg *msg1.ChainMessage) (*msg1.ChainMessage, error) {
	_, _ = handler1.Encrypt(msg)

	return ForwardPeerEndpoint(msg, msg.ConnectPeerId)
}

func ForwardPeerEndpoint(msg *msg1.ChainMessage, connectPeerId string) (*msg1.ChainMessage, error) {
	//转发到另一个定位器
	if connectPeerId != "" && !global.IsMyself(connectPeerId) {
		pipe := handler.GetRequestPipe(connectPeerId, config.P2pParams.ChainProtocolID)
		if pipe != nil {
			data, err := message.Marshal(msg)
			if err == nil {
				logger.Sugar.Debugf("Write data length:%v", len(data))
				_, _, err := pipe.Write(data, false)
				if err == nil {
					return msg, nil
				} else {
					logger.Sugar.Errorf("pipe.Write failure: %v", err)
				}
			}
		} else {
			logger.Sugar.Errorf("targetConnectSessionId has no pipe")
		}
	} else {
		//也许可以找targetPeerId最近的节点发送
		logger.Sugar.Errorf("InvalidConnectPeerId:%v", connectPeerId)
	}
	//如果websocket连接没找到，先保存本地
	if msgtype.CHAT == msg.MessageType {
		_, _ = service2.GetChainMessageService().Insert(msg)
	}
	return msg, nil
}

func ForwardPeerClient(chainMessage *msg1.ChainMessage, peerClient *entity.PeerClient) (*msg1.ChainMessage, error) {
	data, err := message.Marshal(chainMessage)
	if err != nil {
		return nil, err
	}
	connectAddress := peerClient.ConnectAddress
	//如果connectAddress表明是websocket，根据targetConnectSessionId直接转发
	if strings.HasPrefix(connectAddress, "ws") {
		if peerClient.ConnectSessionId != "" {
			websocketConnection, ok := stdhttp.WebsocketConnectionPool[peerClient.ConnectSessionId]
			if ok {
				err = websocketConnection.Write(websocket.BinaryMessage, data)
				if err == nil {
					return chainMessage, nil
				} else {
					logger.Sugar.Errorf("pipe.Write failure: %v", err)
				}
			} else {
				logger.Sugar.Errorf("targetConnectSessionId: %v has no websocketConnection", peerClient.ConnectSessionId)
			}
		} else {
			logger.Sugar.Errorf("targetConnectSessionId is nil")
		}
	} else if config.AppParams.P2pProtocol == "libp2p" {
		if peerClient.ConnectSessionId != "" {
			pipe := handler.GetResponsePipe(peerClient.ConnectSessionId)
			if pipe != nil {
				logger.Sugar.Debugf("Write data length:%v", len(data))
				_, _, err = pipe.Write(data, false)
				if err == nil {
					return chainMessage, nil
				}
			} else {
				logger.Sugar.Errorf("targetConnectSessionId has no pipe")
			}
		} else {
			logger.Sugar.Errorf("targetConnectSessionId is nil")
		}
	}
	logger.Sugar.Errorf("ForwardPeerClient fail")
	if msgtype.CHAT == chainMessage.MessageType {
		_, _ = service2.GetChainMessageService().Insert(chainMessage)
	}

	return chainMessage, nil
}

// RelaySend 转发chainmessage，
// 根据TargetPeerId查询TargetConnectSessionId，找到如何到达目标
func RelaySend(chainMessage *msg1.ChainMessage) (*msg1.ChainMessage, error) {
	topic := chainMessage.Topic
	if topic != "" {
		data, err := message.Marshal(chainMessage)
		if err != nil {
			return nil, err
		}
		go pubsub.SendRaw(topic, data)
		return chainMessage, nil
	}
	//当没有主题的时候，必须有TargetPeerId且不是自己，如果是自己不需要转发
	if chainMessage.TargetPeerId == "" {
		return nil, errors2.New("NullTargetPeerId")
	}
	if global.IsMyself(chainMessage.TargetPeerId) {
		return nil, errors2.New("SendMyself")
	}
	// 查找最终目标会话
	peerClient, connectPeerId, err := Lookup(chainMessage.TargetPeerId, chainMessage.TargetClientId)
	//处理查找结果
	if err == nil {
		// 找到peerClient
		if peerClient != nil {
			chainMessage.TargetConnectAddress = peerClient.ConnectAddress
			chainMessage.TargetConnectPeerId = peerClient.ConnectPeerId
			// 如果PeerClient的连接节点是自己，下一步就是最终目标，将目标会话放入消息中
			if global.IsMyself(peerClient.ConnectPeerId) {
				_, _ = ForwardPeerClient(chainMessage, peerClient)
			} else { // 否则下一步就是连接节点
				_, _ = ForwardPeerEndpoint(chainMessage, peerClient.ConnectPeerId)
			}
			return chainMessage, nil
		} else {
			// 目标是PeerEndPoint，下一步是定位器节点
			if connectPeerId != "" {
				_, _ = ForwardPeerEndpoint(chainMessage, connectPeerId)
				return chainMessage, nil
			}
		}
	}
	//如果无法转发，先保存本地
	if msgtype.CHAT == chainMessage.MessageType {
		_, _ = service2.GetChainMessageService().Insert(chainMessage)
	}
	return nil, err
}

/*
*
本地和分布式查询PeerClient，如果找不到则查找PeerEndpoint
第一个参数返回找到的PeerClient，第二个参数返回找到的PeerEndpoint的peerId，即targetPeerId
*/
func Lookup(targetPeerId string, targetClientId string) (*entity.PeerClient, string, error) {
	// 本地和分布式查询PeerClient，如果找不到则查找PeerEndpoint
	targetPeerId = util.GetPeerId(targetPeerId)
	key := ns.GetPeerClientKey(targetPeerId)
	//本地查询peerClients
	peerClients, err := service.GetPeerClientService().GetLocals(key, targetClientId)
	if len(peerClients) > 0 {
		for _, peerClient := range peerClients {
			if peerClient.ActiveStatus == entity.ActiveStatus_Up {
				return peerClient, "", nil
			}
		}
	}
	//如果本地没有查询到peerClients，则
	peerClients, err = service.GetPeerClientService().GetValues(targetPeerId, "", "", "")
	//客户端连接到另一台PeerEndpoint
	if len(peerClients) > 0 {
		for _, peerClient := range peerClients {
			if peerClient.ActiveStatus == entity.ActiveStatus_Up {
				return peerClient, "", nil
			}
		}
		logger.Sugar.Errorf("find peer client peerId: %v, clientId: %v, but no one has up active status", targetPeerId, targetClientId)
	} else {
		connectPeerId, err := service.GetPeerEndpointService().FindPeer(targetPeerId)

		return nil, connectPeerId, err
	}
	logger.Sugar.Errorf("failed to find peer client peerId: %v, clientId: %v, but no one has up active status", targetPeerId, targetClientId)

	return nil, "", err
}
