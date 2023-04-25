package biz

import (
	"github.com/curltech/go-colla-node/p2p/chain/handler/sender"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	entity3 "github.com/curltech/go-colla-node/p2p/msg/entity"
	service2 "github.com/curltech/go-colla-node/p2p/msg/service"
	"github.com/robfig/cron"
	"time"
)

// RelaySend 发送保存的转发消息，并把转发的消息删除
func RelaySend(peerClient *entity.PeerClient) error {
	targetPeerId := peerClient.PeerId
	chainMessages := make([]*entity3.ChainMessage, 0)
	err := service2.GetChainMessageService().Find(&chainMessages, nil, "", 0, 0, "targetPeerId=?", targetPeerId)
	if err != nil {
		return err
	}
	for _, chainMessage := range chainMessages {
		//chainMessage.TargetConnectSessionId = peerClient.ConnectSessionId
		//chainMessage.TargetConnectPeerId = peerClient.ConnectPeerId
		//chainMessage.TargetConnectAddress = peerClient.ConnectAddress
		//chainMessage.TargetClientId = peerClient.ClientId
		go sender.RelaySend(chainMessage)
	}

	chainMessage := &entity3.ChainMessage{}
	_, err = service2.GetChainMessageService().Delete(chainMessage, "targetPeerId=?", targetPeerId)
	return err
}

func DeleteTimeout() {
	timeout := time.Now().Add(time.Hour * 24 * 7)
	chainMessage := &entity3.ChainMessage{}
	service2.GetChainMessageService().Delete(chainMessage, "createDate<=?", &timeout)
}

func Cron() *cron.Cron {
	c := cron.New()
	c.AddFunc("0 0 2 * * *", DeleteTimeout)
	c.Start()

	return c
}

func init() {
	Cron()
}
