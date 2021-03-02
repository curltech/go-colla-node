package service

import (
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	msg1 "github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

func InitPCResponse(targetPC *entity.PeerClient, messageType msgtype.MsgType) *msg1.PCChainMessage {
	peerId := targetPC.PeerId
	//address := targetPC.Address
	publicKey := targetPC.PublicKey

	response := msg1.PCChainMessage{}
	response.NeedEncrypt = true
	messagePayload := msg1.MessagePayload{}
	messagePayload.TargetPeerId = peerId
	//messagePayload.TargetAddress = address
	messagePayload.SrcPeer = global.Global.MyselfPeer
	response.TargetPublicKey = publicKey
	response.SrcPublicKey = global.Global.MyselfPeer.PublicKey
	response.SecurityContextString = global.Global.MyselfPeer.SecurityContext

	messagePayload.MessageType = messageType
	//messagePayload.MessageDirect = msgtype.MsgDirect_Response
	response.MessagePayload = &messagePayload

	return &response
}
