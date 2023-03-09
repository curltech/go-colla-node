package pubsub

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

var messageHandler func(data []byte, remotePeerId string, clientId string, connectSessionId string, remoteAddr string) ([]byte, error)

func RegistMessageHandler(receiveHandler func(data []byte, remotePeerId string, clientId string, connectSessionId string, remoteAddr string) ([]byte, error)) {
	messageHandler = receiveHandler
}

/**
订阅主题和订阅者
*/
type PubsubTopic struct {
	Topic *pubsub.Topic
	Sub   *pubsub.Subscription
}

var Pubsub *pubsub.PubSub

var PubsubTopicPool = make(map[string]*PubsubTopic, 0)

/**
加入主题，可以向这个主题发消息
如果主题不存在，创建新的
*/
func joinTopic(topicname string) (*PubsubTopic, error) {
	// create a new PubSub service using the GossipSub router
	var err error
	pubsubTopic, ok := PubsubTopicPool[topicname]
	if !ok {
		Pubsub, err = pubsub.NewGossipSub(global.Global.Context, global.Global.Host)
		if err != nil {
			return nil, err
		}
		pubsubTopic = &PubsubTopic{}
		PubsubTopicPool[topicname] = pubsubTopic
	}
	if pubsubTopic.Topic != nil {
		pubsubTopic.Topic.Close()
		pubsubTopic.Topic = nil
	}
	topic, err := Pubsub.Join(topicname, nil)
	if err != nil {
		return nil, err
	}
	pubsubTopic.Topic = topic

	return pubsubTopic, nil
}

/**
订阅主题，可以接收发给这个主题的消息
*/
func Subscribe(topicname string) (*PubsubTopic, error) {
	var err error
	pubsubTopic, ok := PubsubTopicPool[topicname]
	if !ok {
		pubsubTopic, err = joinTopic(topicname)
		if err != nil {
			return nil, err
		}
	}
	pubsubTopic.Sub, err = pubsubTopic.Topic.Subscribe()
	if err != nil {
		return nil, err
	}
	go subLoop(pubsubTopic.Sub)

	return pubsubTopic, nil
}

/**
无限循环读取
*/
func subLoop(sub *pubsub.Subscription) error {
	for {
		topicMsg, err := sub.Next(global.Global.Context)
		if err != nil {
			logger.Sugar.Errorf("%v", err)
			continue
		}
		messageHandler(topicMsg.Data, "", "", "", "")
	}

	return nil
}

func SendRaw(topicname string, data []byte) {
	pubsubTopic, ok := PubsubTopicPool[topicname]
	if ok {
		pubsubTopic.Topic.Publish(global.Global.Context, data, pubHandle)
	}
}

/**
在发送消息之前循环调用
*/
func pubHandle(pub *pubsub.PublishOptions) error {
	return nil
}

func ListPeers(topicname string) []peer.ID {
	return Pubsub.ListPeers(topicname)
}

func GetTopics() []string {
	return Pubsub.GetTopics()
}
