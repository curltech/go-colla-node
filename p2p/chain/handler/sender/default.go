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
直接发送到下一个节点，报文不做处理，有两种发送目标：
1.发送到peerClient，这时候TargetPeerId,TargetConnectSessionId和TargetConnectPeerId不为空，
TargetConnectPeerId就是自己，connectPeerId可以为空或者就是TargetPeerId
2.不满足上面的条件，发送到peerEndpoint，这时候connectPeerId不为空并且不是自己
3.如果主题不为空，发送到主题
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
	if config.AppParams.P2pProtocol == "libp2p" {
		/**
		如果目标会话不为空，而且目标连接节点是自己，则查询管道，直接发送到客户端
		否则，转发到下一步节点
		*/
		if targetPeerId != "" && targetConnectSessionId != "" && targetConnectPeerId != "" && global.IsMyself(targetConnectPeerId) {
			p := handler.GetPipePool().GetResponsePipe(targetPeerId, targetConnectSessionId)
			if p != nil {
				_, _, err = p.Write(data, false)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, errors.New("NoPipe")
			}
		} else {
			if connectPeerId == "" && targetConnectPeerId != "" {
				connectPeerId = targetConnectPeerId
				msg.ConnectPeerId = connectPeerId
			}
			if connectPeerId != "" {
				_, err := handler.SendRaw(connectPeerId, config.P2pParams.ChainProtocolID, data)
				if err != nil {
					return nil, err
				}
			} else {
				//也许可以找targetPeerId最近的节点发送
				logger.Errorf("NoConnectPeerId")
				return msg, errors.New("NoConnectPeerId")
			}
			if topic != "" {
				pubsub.SendRaw(topic, data)
			}
		}
	} else {
		if targetConnectSessionId != "" {
			//websocket.SendRaw(targetConnectSessionId, data)
		} else {
			return msg, errors.New("NoConnectSessionId")
		}
	}

	return msg, nil
}

/**
无错无返回表示不需要转发
*/
func RelaySend(chainMessage *msg1.ChainMessage) (*msg1.ChainMessage, error) {
	if chainMessage.TargetPeerId != "" && global.IsMyself(chainMessage.TargetPeerId) {
		return chainMessage, errors.New("SendMyself")
	}
	var response *msg1.ChainMessage
	var err error
	/**
	最终目标节点为空，直接按照connectPeerId发送
	*/
	if chainMessage.TargetPeerId == "" {
		return send(chainMessage)
	}
	// 最终目标会话不为空，说明最终目标是PeerClient，不需要查询PeerClient，直接转发
	if chainMessage.TargetConnectSessionId != "" {
		chainMessage.ConnectPeerId = chainMessage.TargetPeerId
		return send(chainMessage)
	} else {
		// 最终目标会话为空，先检查PeerClient是否有对应的目标，如果有，填写最终目标会话，设置下一步的目标
		peerClients, err := service.GetPeerClientService().GetValue(chainMessage.TargetPeerId)
		if err == nil && len(peerClients) > 0 {
			for _, peerClient := range peerClients {
				// 如果PeerClient的连接节点是自己，下一步就是最终目标
				// 在目标是peerClient的时候将目标的连接节点和会话id放入消息中
				if global.IsMyself(peerClient.ConnectPeerId) {
					chainMessage.TargetConnectPeerId = peerClient.ConnectPeerId
					chainMessage.TargetConnectSessionId = peerClient.ConnectSessionId
					chainMessage.ConnectPeerId = chainMessage.TargetPeerId
				} else { // 否则下一步就是连接节点
					chainMessage.TargetConnectPeerId = peerClient.ConnectPeerId
					chainMessage.TargetConnectSessionId = peerClient.ConnectSessionId
					chainMessage.ConnectPeerId = peerClient.ConnectPeerId
				}
				return send(chainMessage)
			}
		} else {
			// 如果PeerClient不是最终目标，那么查找定位器节点是否是最终目标，如果是，下一步是定位器节点
			connectPeerId, err := service.GetPeerEndpointService().FindPeer(chainMessage.TargetPeerId)
			if err == nil {
				chainMessage.ConnectPeerId = connectPeerId
				return send(chainMessage)
			}
		}
	}

	return response, err
}
