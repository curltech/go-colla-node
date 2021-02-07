package dht

import (
	"errors"
	"github.com/curltech/go-colla-core/crypto/std"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	entity1 "github.com/curltech/go-colla-node/p2p/chain/entity"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"time"
)

type putValueAction struct {
	action.BaseAction
}

var PutValueAction putValueAction

/**
在chain目录下的采用自定义protocol "/chain"的方式自己实现的功能
*/
func (this *putValueAction) PutValue(peerId string, payloadType string, data interface{}) (interface{}, error) {
	chainMessage := msg.ChainMessage{}
	chainMessage.Payload = data
	chainMessage.ConnectPeerId = peerId
	chainMessage.PayloadType = payloadType
	chainMessage.MessageType = msgtype.PUTVALUE
	chainMessage.MessageDirect = msgtype.MsgDirect_Request

	response, err := this.Send(&chainMessage)
	if err != nil {
		return nil, err
	}
	if response != nil {
		return response.Payload, nil
	}

	return nil, nil
}

func (this *putValueAction) Receive(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Sugar.Infof("Receive %v message", this.MsgType)
	var response *msg.ChainMessage = nil
	v := chainMessage.Payload
	peerClient, ok := v.(*entity.PeerClient)
	var key string
	if ok {
		err := service.GetPeerClientService().Validate(peerClient)
		if err != nil {
			response = handler.Error(chainMessage.MessageType, err)
			return response, nil
		}
		peerClient.ConnectSessionId = chainMessage.ConnectSessionId

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
		pcs, err := service.GetPeerClientService().GetLocals(ns.PeerClient_KeyKind, peerId, "", "")
		if err != nil {
			response = handler.Error(chainMessage.MessageType, err)
			return response, nil
		}
		if pcs == nil {
			response = handler.Error(chainMessage.MessageType, errors.New("NoLocalPCs"))
			return response, nil
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
				err := service.GetPeerClientService().PutValues(pc)
				if err != nil {
					response = handler.Error(chainMessage.MessageType, err)
					return response, nil
				}
			}
		}

		response = handler.Ok(chainMessage.MessageType)
		return response, nil
	} else {
		peerEndpoint, ok := v.(*entity.PeerEndpoint)
		if ok {
			key = "/" + ns.PeerEndpoint_Prefix + "/" + peerEndpoint.PeerId
			peerEndpoint.ConnectSessionId = chainMessage.ConnectSessionId
		} else {
			chainApp, ok := v.(*entity.ChainApp)
			if ok {
				key = "/" + ns.ChainApp_Prefix + "/" + chainApp.PeerId
				chainApp.ConnectSessionId = chainMessage.ConnectSessionId
			} else {
				dataBlock, ok := v.(*entity1.DataBlock)
				if ok {
					key = "/" + ns.DataBlock_Prefix + "/" + dataBlock.BlockId
				}
			}
		}
		buf, err := message.Marshal(v)
		if err != nil {
			response = handler.Error(chainMessage.MessageType, err)
		}
		err = dht.PeerEndpointDHT.PutValue(key, buf)
		if err != nil {
			response = handler.Error(chainMessage.MessageType, err)
		}
		if response == nil {
			response = handler.Ok(chainMessage.MessageType)
		}

		return response, nil
	}
}

func init() {
	PutValueAction = putValueAction{}
	PutValueAction.MsgType = msgtype.PUTVALUE
	handler.RegistChainMessageHandler(msgtype.PUTVALUE, PutValueAction.Send, PutValueAction.Receive, PutValueAction.Response)
}
