package action

import (
	"errors"
	"github.com/curltech/go-colla-core/crypto/std"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	service1 "github.com/curltech/go-colla-node/p2p/chain/service"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"time"
)

type storeClientAction struct {
	BaseAction
}

var StoreClientAction storeClientAction

/**
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (this *storeClientAction) PCReceive(chainMessage *msg.PCChainMessage) (interface{}, error) {
	logger.Infof("Receive %v message", this.MsgType)
	peerClient := chainMessage.MessagePayload.Payload.(*entity.PeerClient)
	err := service1.ValidatePC(chainMessage.MessagePayload)
	if err != nil {
		return msgtype.ERROR, err
	}

	peerId := peerClient.PeerId
	clientId := peerClient.ClientId
	/*connectAddress := peerClient.ConnectAddress
	connectPeerId := peerClient.ConnectPeerId
	connectPublicKey := peerClient.ConnectPublicKey
	connectSessionId := peerClient.ConnectSessionId*/
	previousPublicKeySignature := peerClient.PreviousPublicKeySignature
	signature := peerClient.Signature
	signatureData := peerClient.SignatureData
	expireDate := peerClient.ExpireDate

	// 更新信息
	pcs, err := service1.GetLocalPCs(ns.PeerClient_KeyKind, peerId, "", "")
	if err != nil {
		return msgtype.ERROR, err
	}
	if pcs == nil {
		return msgtype.ERROR, errors.New("NoLocalPCs")
	} else {
		currentTime := time.Now()
		for _, pc := range pcs {
			// 更新信息
			if pc.ClientId == clientId {
				pc.LastAccessTime = &currentTime
				/*pc.ActiveStatus = entity.ActiveStatus_Up
				pc.ConnectAddress = connectAddress
				pc.ConnectPeerId = connectPeerId
				pc.ConnectPublicKey = connectPublicKey
				pc.ConnectSessionId = connectSessionId*/
				// Mobile只能修改本实例，其它实例仍需从客户端修改
				pc.Mobile = std.EncodeBase64(std.Hash(peerClient.Mobile, "sha3_256"))
				pc.MobileVerified = peerClient.MobileVerified
				// resetKey也限于本实例，且在connect中处理
				//pc.PublicKey = peerClient.PublicKey
			}
			pc.PreviousPublicKeySignature = previousPublicKeySignature
			pc.Signature = signature
			pc.SignatureData = signatureData
			pc.ExpireDate = expireDate
			pc.LastUpdateTime = peerClient.LastUpdateTime
			pc.Name = peerClient.Name
			pc.Avatar = peerClient.Avatar
			pc.VisibilitySetting = peerClient.VisibilitySetting
			err := service1.PutPCs(pc)
			if err != nil {
				return msgtype.ERROR, err
			}
		}
	}

	return msgtype.OK, nil
}

/**
处理返回消息
*/
func (this *storeClientAction) PCResponse(chainMessage *msg.PCChainMessage) error {
	logger.Infof("Response %v message:%v", this.MsgType, chainMessage)

	return nil
}

func init() {
	StoreClientAction = storeClientAction{}
	StoreClientAction.MsgType = msgtype.STORECLIENT
	handler.RegistChainMessageHandler(msgtype.STORECLIENT, StoreClientAction.Send, StoreClientAction.Receive, StoreClientAction.Response)
	handler.RegistPCChainMessageHandler(msgtype.STORECLIENT, StoreClientAction.Send, StoreClientAction.PCReceive, StoreClientAction.PCResponse)
}
