package sender

import (
	"errors"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/pipe/handler"
	"github.com/curltech/go-colla-node/libp2p/pubsub"
	handler1 "github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	msg1 "github.com/curltech/go-colla-node/p2p/msg"
)

/**
发送ChainMessage消息的唯一方法
1.找出发送的目标地址和方式
2.根据情况处理校验，加密，压缩等
3.建立合适的通道并发送，比如libp2p的Pipe并Write消息流
4.等待即时的返回，校验，解密，解压缩等
*/
func Send(msg *msg1.ChainMessage) (*msg1.ChainMessage, error) {
	handler1.Encrypt(msg)

	return RelaySend(msg)
}

/**
直接发送到下一个节点，报文不做处理，有3种发送目标：
1.如果主题不为空，发送到主题
2.发送到peerClient：这时候TargetPeerId和TargetConnectSessionId不为空、TargetConnectPeerId可以为空或者就是自己（connectPeerId可以为空或者就是TargetPeerId）
3.不满足上面的条件，发送到peerEndpoint，这时候connectPeerId不为空并且不是自己
*/
func send(msg *msg1.ChainMessage) (*msg1.ChainMessage, error) {
	topic := msg.Topic
	connectPeerId := msg.ConnectPeerId
	targetPeerId := msg.TargetPeerId
	targetConnectSessionId := msg.TargetConnectSessionId
	targetConnectPeerId := msg.TargetConnectPeerId
	data, err := message.Marshal(msg)
	if err != nil {
		return nil, err
	}
	if topic != "" {
		go pubsub.SendRaw(topic, data)
	}
	if config.AppParams.P2pProtocol == "libp2p" {
		/**
		如果目标会话不为空，则查询管道，直接发送到客户端
		否则，转发到下一步节点
		*/
		if targetPeerId != "" && targetConnectSessionId != "" && (targetConnectPeerId == "" || global.IsMyself(targetConnectPeerId)) {
			pipe := handler.GetPipePool().GetResponsePipe(targetPeerId, targetConnectSessionId)
			if pipe != nil {
				logger.Infof("Write data length:%v", len(data))
				_, _, err = pipe.Write(data, false)
				if err != nil {
					logger.Errorf("pipe.Write failure")
					return nil, err
				}
			} else {
				return nil, errors.New("NoPipe")
			}
		} else {
			if connectPeerId != "" && !global.IsMyself(connectPeerId) {
				pipe := handler.GetPipePool().GetRequestPipe(connectPeerId, config.P2pParams.ChainProtocolID)
				if pipe != nil {
					logger.Infof("Write data length:%v", len(data))
					_, _, err := pipe.Write(data, false)
					if err != nil {
						logger.Errorf("pipe.Write failure")
						return nil, err
					}
				} else {
					return nil, errors.New("NoPipe")
				}
			} else {
				//也许可以找targetPeerId最近的节点发送
				logger.Errorf("InvalidConnectPeerId")
				return nil, errors.New("InvalidConnectPeerId")
			}
		}
	} else {
		if targetConnectSessionId != "" {
			//websocket.SendRaw(targetConnectSessionId, data)
		} else {
			return nil, errors.New("NoConnectSessionId")
		}
	}

	return msg, nil
}

/**
无错无返回表示不需要转发
*/
func RelaySend(chainMessage *msg1.ChainMessage) (*msg1.ChainMessage, error) {
	if chainMessage.Topic == "" {
		if chainMessage.TargetPeerId == "" {
			return nil, errors.New("NullTargetPeerId")
		}
		if global.IsMyself(chainMessage.TargetPeerId) {
			return nil, errors.New("SendMyself")
		}
	}
	// 最终目标会话不为空，说明最终目标是PeerClient，不需要查询PeerClient，直接转发
	if chainMessage.TargetConnectSessionId != "" {
		chainMessage.ConnectPeerId = chainMessage.TargetPeerId
		return send(chainMessage)
	} else {
		// 最终目标会话为空，先检查PeerClient是否有对应的目标，如果有，填写最终目标会话，设置下一步的目标
		peerClients, err := service.GetPeerClientService().GetValues(chainMessage.TargetPeerId, "")
		if err == nil && len(peerClients) > 0 {
			for _, peerClient := range peerClients {
				// 如果PeerClient的连接节点是自己，下一步就是最终目标，将目标会话放入消息中
				if global.IsMyself(peerClient.ConnectPeerId) {
					chainMessage.TargetConnectSessionId = peerClient.ConnectSessionId
					chainMessage.TargetConnectPeerId = peerClient.ConnectPeerId
					chainMessage.ConnectPeerId = chainMessage.TargetPeerId
				} else { // 否则下一步就是连接节点
					chainMessage.TargetConnectSessionId = peerClient.ConnectSessionId
					chainMessage.TargetConnectPeerId = peerClient.ConnectPeerId
					chainMessage.ConnectPeerId = peerClient.ConnectPeerId
				}
				go send(chainMessage)
			}
			return chainMessage, nil
		} else {
			// 如果PeerClient不是最终目标，那么查找定位器节点是否是最终目标，如果是，下一步是定位器节点
			connectPeerId, err := service.GetPeerEndpointService().FindPeer(chainMessage.TargetPeerId)
			if err != nil {
				return nil, err
			} else {
				if connectPeerId != "" {
					chainMessage.ConnectPeerId = connectPeerId
				} else {
					targetConnectPeerId := chainMessage.TargetConnectPeerId
					if targetConnectPeerId != "" {
						chainMessage.ConnectPeerId = targetConnectPeerId
					}
				}
				return send(chainMessage)
			}
		}
	}
}
