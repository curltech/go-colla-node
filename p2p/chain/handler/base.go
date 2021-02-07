package handler

import (
	"errors"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/curltech/go-colla-core/crypto/openpgp"
	"github.com/curltech/go-colla-core/crypto/std"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/compress"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/util"
	entity2 "github.com/curltech/go-colla-node/p2p/chain/entity"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	msg1 "github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"unsafe"
)

/**
在发送前校验各字段，然后再加密等处理
*/
func SendValidate(msg *msg1.ChainMessage) error {
	return nil
}

/**
在接收解密处理后校验，然后再进行业务处理
*/
func ReceiveValidate(msg *msg1.ChainMessage) error {
	return nil
}

/**
在发送回应数据前校验，然后再加密等处理
*/
func ResponseValidate(msg *msg1.ChainMessage) error {
	return nil
}

func Encrypt(msg *msg1.ChainMessage) (*msg1.ChainMessage, error) {
	if msg.SrcPeerId != "" && !global.IsMyself(msg.SrcPeerId) {
		return msg, nil
	}
	if msg.Payload == nil {
		return msg, nil
	}

	data, err := message.Marshal(msg.Payload)
	if err != nil {
		return nil, errors.New("PayloadMarshalFailure")
	}

	if msg.NeedEncrypt == true {
		targetPeerId := msg.TargetPeerId
		if targetPeerId == "" {
			targetPeerId, _ = util.GetIdAddr(msg.ConnectPeerId)
		}
		openpgpPub, err := GetPublicKey(targetPeerId)
		if err != nil {
			msg.NeedEncrypt = false
			return msg, err
		}

		signature := openpgp.Sign(global.Global.PrivateKey, nil, data)
		msg.PayloadSignature = std.EncodeBase64(signature)
		if msg.NeedCompress == true {
			data = compress.GzipCompress(data)
		}
		key := std.GenerateSecretKey(32)
		data = openpgp.EncryptSymmetrical([]byte(key), data)

		payloadKey := openpgp.EncryptKey([]byte(key), openpgpPub)
		msg.PayloadKey = std.EncodeBase64(payloadKey)
	}
	msg.TransportPayload = std.EncodeBase64(data)
	msg.Payload = nil

	return msg, nil
}

func GetPublicKey(targetPeerId string) (*crypto.Key, error) {
	targetPublicKey := ""
	if targetPeerId == "" {
		return nil, errors.New("NoTargetPeerId")
	}
	peerClients, err := service.GetPeerClientService().GetValues(targetPeerId, "")
	if err == nil && len(peerClients) > 0 {
		peerClient := peerClients[0]
		targetPublicKey = peerClient.PublicKey
	} else {
		peerEndpoint, err := service.GetPeerEndpointService().GetValue(targetPeerId)
		if err == nil && peerEndpoint != nil {
			targetPublicKey = peerEndpoint.PublicKey
		}
	}
	if targetPublicKey == "" {
		return nil, errors.New("NoTargetPublicKey")
	}
	openpgpPublicKey := std.DecodeBase64(targetPublicKey)
	openpgpPub, err := openpgp.LoadPublicKey(openpgpPublicKey)
	if err != nil {
		return nil, errors.New("LoadSrcPublicKeyFailure")
	}
	return openpgpPub, nil
}

const (
	PayloadType_PeerClient   = "peerClient"
	PayloadType_PeerEndpoint = "peerEndpoint"
	PayloadType_ChainApp     = "chainApp"
	PayloadType_DataBlock    = "dataBlock"
	PayloadType_ConsensusLog = "consensusLog"

	PayloadType_PeerClients   = "peerClients"
	PayloadType_PeerEndpoints = "peerEndpoints"
	PayloadType_ChainApps     = "chainApps"
	PayloadType_DataBlocks    = "dataBlocks"

	PayloadType_String = "string"
	PayloadType_Map    = "map"

	PayloadType_WebsocketChainMessage = "websocketChainMessage"
)

func Decrypt(msg *msg1.ChainMessage) (*msg1.ChainMessage, error) {
	targetPeerId := msg.TargetPeerId
	if targetPeerId == "" {
		targetPeerId = msg.ConnectPeerId
	}
	if !global.IsMyself(targetPeerId) {
		return msg, nil
	}

	if msg.TransportPayload == "" {
		return msg, errors.New("NoTransportPayload")
	}
	data := std.DecodeBase64(msg.TransportPayload)
	if msg.NeedEncrypt == true && msg.PayloadKey != "" {
		srcPublicKey, err := GetPublicKey(msg.SrcPeerId)
		if err == nil {
			payloadSignature := std.DecodeBase64(msg.PayloadSignature)
			pass := openpgp.Verify(srcPublicKey, data, payloadSignature)
			if pass != true {
				previousPublicKeyPayloadSignature := std.DecodeBase64(msg.PreviousPublicKeyPayloadSignature)
				pass = openpgp.Verify(srcPublicKey, data, previousPublicKeyPayloadSignature)
				if pass != true {
					return nil, errors.New("PayloadVerifyFailure")
				}
			}
		}
		payloadKey := std.DecodeBase64(msg.PayloadKey)
		secretKey := openpgp.DecryptKey(payloadKey, global.Global.PrivateKey)
		data = openpgp.DecryptSymmetrical(secretKey, data)
	}
	if msg.NeedCompress == true {
		data = compress.GzipUncompress(data)
	}
	var err error
	switch msg.PayloadType {
	case PayloadType_String:
		payload := ""
		err = message.Unmarshal(data, &payload)
		msg.Payload = payload
	case PayloadType_Map:
		payload := make(map[string]interface{})
		err = message.Unmarshal(data, &payload)
		msg.Payload = payload
	case PayloadType_PeerClient:
		payload := &entity.PeerClient{}
		err = message.Unmarshal(data, &payload)
		msg.Payload = payload
	case PayloadType_PeerEndpoint:
		payload := &entity.PeerEndpoint{}
		err = message.Unmarshal(data, &payload)
		msg.Payload = payload
	case PayloadType_WebsocketChainMessage:
		payload := &msg1.WebsocketChainMessage{}
		err = message.Unmarshal(data, &payload)
		msg.Payload = payload
	default:
		payload := make(map[string]interface{})
		err = message.Unmarshal(data, &payload)
		msg.Payload = payload
	}
	msg.TransportPayload = ""
	return msg, err
}

func Error(msgType msgtype.MsgType, err error) *msg1.ChainMessage {
	errMessage := msg1.ChainMessage{}
	errMessage.Payload = msgtype.ERROR
	errMessage.PayloadType = PayloadType_String
	errMessage.Tip = err.Error()
	errMessage.MessageType = msgType
	errMessage.MessageDirect = msgtype.MsgDirect_Response

	return &errMessage
}

func Response(msgType msgtype.MsgType, payload interface{}) *msg1.ChainMessage {
	responseMessage := msg1.ChainMessage{}
	responseMessage.Payload = payload
	responseMessage.MessageType = msgType
	responseMessage.MessageDirect = msgtype.MsgDirect_Response

	return &responseMessage
}

func Ok(msgType msgtype.MsgType) *msg1.ChainMessage {
	okMessage := msg1.ChainMessage{}
	okMessage.Payload = msgtype.OK
	okMessage.PayloadType = PayloadType_String
	okMessage.Tip = "OK"
	okMessage.MessageType = msgType
	okMessage.MessageDirect = msgtype.MsgDirect_Response

	return &okMessage
}

func SetResponse(request *msg1.ChainMessage, response *msg1.ChainMessage) {
	response.UUID = request.UUID
	response.LocalConnectPeerId = ""
	response.LocalConnectAddress = ""
	response.SrcAddress = request.SrcAddress
	response.SrcPeerId = request.SrcPeerId
	response.ConnectAddress = request.LocalConnectAddress
	response.ConnectPeerId = request.LocalConnectPeerId
	response.ConnectSessionId = request.ConnectSessionId
	response.Topic = request.Topic
}

////////////////////

func EncryptPC(msg *msg1.PCChainMessage) (*msg1.PCChainMessage, error) {
	payload, err := message.TextMarshal(msg.MessagePayload.Payload)
	if err != nil {
		return nil, errors.New("PayloadTextMarshalFailure")
	}
	msg.MessagePayload.TransportPayload = payload
	msg.MessagePayload.Payload = nil
	messagePayload, err := message.TextMarshal(msg.MessagePayload)
	if err != nil {
		return nil, errors.New("MessagePayloadTextMarshalFailure")
	}
	data := []byte(messagePayload)
	msg.MessagePayload = nil

	signature := openpgp.Sign(global.Global.PrivateKey, nil, data)
	msg.PayloadSignature = std.EncodeBase64(signature)
	if msg.NeedCompress != "none" {
		data = compress.GzipCompress(data)
	}
	if msg.NeedEncrypt == true {
		key := std.GenerateSecretKey(32)
		data = openpgp.EncryptSymmetrical([]byte(key), data)
		openpgpPublicKey := std.DecodeBase64(msg.TargetPublicKey)
		openpgpPub, err := openpgp.LoadPublicKey(openpgpPublicKey)
		if err != nil {
			return nil, errors.New("LoadSrcPublicKeyFailure")
		}
		payloadKey := openpgp.EncryptKey([]byte(key), openpgpPub)
		msg.PayloadKey = std.EncodeBase64(payloadKey)
	}
	msg.TransportMessagePayload = std.EncodeBase64(data)

	return msg, nil
}

// ChainMessage PayloadClass

const (
	// PeerClient
	PayloadClass_PeerClient = "com.castalia.blockchain.client.domain.PeerClient"
	// Block
	PayloadClass_DataBlock = "com.castalia.blockchain.client.domain.DataBlock"
	// WebsocketChainMessage
	PayloadClass_WebsocketChainMessage = "com.castalia.blockchain.websocket.domain.WebsocketChainMessage"
	// Map
	PayloadClass_Map = "java.util.Map"
)

func DecryptPC(msg *msg1.PCChainMessage) (*msg1.PCChainMessage, error) {
	data := std.DecodeBase64(msg.TransportMessagePayload)
	if msg.NeedEncrypt == true {
		payloadKey := std.DecodeBase64(msg.PayloadKey)
		secretKey := openpgp.DecryptKey(payloadKey, global.Global.PrivateKey)
		data = openpgp.DecryptSymmetrical(secretKey, data)
	}
	if msg.NeedCompress != "none" {
		data = compress.GzipUncompress(data)
	}
	byteSrcPublicKey := std.DecodeBase64(msg.SrcPublicKey)
	srcPublicKey, err := openpgp.LoadPublicKey(byteSrcPublicKey)
	if err != nil {
		return nil, errors.New("LoadSrcPublicKeyFailure")
	}
	payloadSignature := std.DecodeBase64(msg.PayloadSignature)
	pass := openpgp.Verify(srcPublicKey, data, payloadSignature)
	if pass != true {
		return nil, errors.New("PayloadVerifyFailure")
	}
	transportMessagePayload := (*string)(unsafe.Pointer(&data))
	messagePayload := &msg1.MessagePayload{}
	messagePayload.SrcPeer = &entity.PeerClient{}
	err = message.TextUnmarshal(*transportMessagePayload, messagePayload)
	if err != nil {
		return nil, errors.New("MessagePayloadTextUnmarshalFailure")
	}
	msg.MessagePayload = messagePayload
	msg.TransportMessagePayload = ""
	if msg.MessagePayload.PayloadClass == PayloadClass_PeerClient {
		payload := &entity.PeerClient{}
		err = message.TextUnmarshal(msg.MessagePayload.TransportPayload, payload)
		msg.MessagePayload.Payload = payload
	} else if msg.MessagePayload.PayloadClass == PayloadClass_Map {
		payload := make(map[string]interface{}, 0)
		err = message.TextUnmarshal(msg.MessagePayload.TransportPayload, &payload)
		msg.MessagePayload.Payload = payload
	} else if msg.MessagePayload.PayloadClass == PayloadClass_WebsocketChainMessage {
		payload := &msg1.WebsocketChainMessage{}
		err = message.TextUnmarshal(msg.MessagePayload.TransportPayload, payload)
		msg.MessagePayload.Payload = payload
	} else if msg.MessagePayload.PayloadClass == PayloadClass_DataBlock {
		payload := &entity2.DataBlock{}
		err = message.TextUnmarshal(msg.MessagePayload.TransportPayload, payload)
		msg.MessagePayload.Payload = payload
	} else {
		logger.Sugar.Errorf("InvalidPayloadClass: %v", msg.MessagePayload.PayloadClass)
		err = errors.New("InvalidPayloadClass")
	}
	if err != nil {
		return nil, err
	}
	msg.MessagePayload.TransportPayload = ""

	return msg, nil
}

func ErrorPC(msgType msgtype.MsgType, err error) *msg1.PCChainMessage {
	errMessage := msg1.PCChainMessage{}
	errMessage.MessagePayload.Payload = msgtype.ERROR
	//errMessage.Tip = err.Error()
	errMessage.MessagePayload.MessageType = msgType
	//errMessage.MessageDirect = msgtype.MsgDirect_Response

	return &errMessage
}

func ResponsePC(msgType msgtype.MsgType) *msg1.PCChainMessage {
	responseMessage := msg1.PCChainMessage{}
	responseMessage.MessagePayload.Payload = msgtype.RESPONSE
	//responseMessage.Tip = "async response"
	responseMessage.MessagePayload.MessageType = msgType
	//responseMessage.MessageDirect = msgtype.MsgDirect_Response

	return &responseMessage
}

func OkPC(msgType msgtype.MsgType) *msg1.PCChainMessage {
	okMessage := msg1.PCChainMessage{}
	okMessage.MessagePayload.Payload = msgtype.OK
	//okMessage.Tip = "OK"
	okMessage.MessagePayload.MessageType = msgType
	//okMessage.MessageDirect = msgtype.MsgDirect_Response

	return &okMessage
}
