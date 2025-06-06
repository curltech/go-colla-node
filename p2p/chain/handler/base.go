package handler

import (
	"errors"
	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/curltech/go-colla-core/crypto/openpgp"
	"github.com/curltech/go-colla-core/crypto/std"
	"github.com/curltech/go-colla-core/util/compress"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/util"
	entity2 "github.com/curltech/go-colla-node/p2p/chain/entity"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	msg1 "github.com/curltech/go-colla-node/p2p/msg/entity"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"net/http"
)

/*
*
在发送前校验各字段，然后再加密等处理
*/
func SendValidate(msg *msg1.ChainMessage) error {
	return nil
}

/*
*
在接收解密处理后校验，然后再进行业务处理
*/
func ReceiveValidate(msg *msg1.ChainMessage) error {
	return nil
}

/*
*
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

	var openpgpPub *crypto.Key
	if msg.NeedEncrypt == true {
		targetPeerId := msg.TargetPeerId
		if targetPeerId == "" {
			targetPeerId, _ = util.GetIdAddr(msg.ConnectPeerId)
		}
		openpgpPub, err = GetPublicKey(targetPeerId)
		if err != nil {
			msg.NeedEncrypt = false
			return msg, err
		}

		signature, _ := openpgp.Sign(global.Global.PrivateKey, data)
		msg.PayloadSignature = std.EncodeBase64(signature)
	}
	if msg.NeedCompress == true && len(string(data)) > CompressLimit {
		data = compress.GzipCompress(data)
	} else {
		msg.NeedCompress = false
	}
	if msg.NeedEncrypt == true {
		key := std.GenerateSecretKey(32)
		data, err = openpgp.EncryptSymmetrical([]byte(key), data)
		if err != nil {
			msg.NeedEncrypt = false
			return msg, err
		}
		payloadKey, err := openpgp.EncryptKey([]byte(key), openpgpPub)
		if err != nil {
			msg.NeedEncrypt = false
			return msg, err
		}
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
	peerClients, err := service.GetPeerClientService().GetValues(targetPeerId, "", "", "")
	if err == nil && len(peerClients) > 0 {
		latestPeerClient := &entity.PeerClient{}
		for _, peerClient := range peerClients {
			if latestPeerClient.PublicKey == "" {
				latestPeerClient = peerClient
			} else {
				if peerClient.LastUpdateTime != nil && latestPeerClient.LastUpdateTime != nil &&
					(peerClient.LastUpdateTime.UTC().After(latestPeerClient.LastUpdateTime.UTC()) ||
						(peerClient.LastUpdateTime.UTC().Equal(latestPeerClient.LastUpdateTime.UTC()) &&
							peerClient.LastAccessTime != nil && latestPeerClient.LastAccessTime != nil &&
							peerClient.LastAccessTime.UTC().After(latestPeerClient.LastAccessTime.UTC()))) {
					latestPeerClient = peerClient
				}
			}
		}
		if latestPeerClient.PublicKey != "" {
			targetPublicKey = latestPeerClient.PublicKey
		}
	} else {
		peerEndpoint, err := service.GetPeerEndpointService().GetValue(targetPeerId)
		if err == nil && peerEndpoint != nil {
			if peerEndpoint.PublicKey != "" {
				targetPublicKey = peerEndpoint.PublicKey
			}
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

const CompressLimit = 2048

const PayloadLimit = 32 * 1024

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
)

func Decrypt(msg *msg1.ChainMessage) (*msg1.ChainMessage, error) {
	if msg.TransportPayload == "" {
		return msg, errors.New("NoTransportPayload")
	}
	data := std.DecodeBase64(msg.TransportPayload)
	if msg.NeedEncrypt == true && msg.PayloadKey != "" {
		srcPublicKey, err := GetPublicKey(msg.SrcPeerId)
		if err == nil {
			payloadSignature := std.DecodeBase64(msg.PayloadSignature)
			pass, _ := openpgp.Verify(srcPublicKey, data, payloadSignature)
			if pass != true {
				previousPublicKeyPayloadSignature := std.DecodeBase64(msg.PreviousPublicKeyPayloadSignature)
				pass, _ = openpgp.Verify(srcPublicKey, data, previousPublicKeyPayloadSignature)
				if pass != true {
					return nil, errors.New("PayloadVerifyFailure")
				}
			}
		}
		payloadKey := std.DecodeBase64(msg.PayloadKey)
		secretKey, err := openpgp.DecryptKey(payloadKey, global.Global.PrivateKey)
		if err != nil {
			return msg, err
		}
		data, err = openpgp.DecryptSymmetrical(secretKey, data)
		if err != nil {
			return msg, err
		}
	}
	if msg.NeedCompress == true {
		data = compress.GzipUncompress(data)
	}
	var err error
	var payload interface{}
	switch msg.PayloadType {
	case PayloadType_String:
		payload = ""
	case PayloadType_PeerClient:
		payload = &entity.PeerClient{}
	case PayloadType_PeerEndpoint:
		payload = &entity.PeerEndpoint{}
	case PayloadType_DataBlock:
		payload = &entity2.DataBlock{}
	case PayloadType_ConsensusLog:
		payload = &entity2.ConsensusLog{}
	default: // PayloadType_Map
		payload = make(map[string]interface{})
	}
	err = message.Unmarshal(data, &payload)
	msg.Payload = payload
	msg.TransportPayload = ""
	return msg, err
}

func Error(msgType string, err error) *msg1.ChainMessage {
	errMessage := msg1.ChainMessage{}
	errMessage.Payload = msgtype.ERROR
	errMessage.PayloadType = PayloadType_String
	errMessage.Tip = err.Error()
	errMessage.StatusCode = http.StatusInternalServerError
	errMessage.MessageType = msgType
	errMessage.MessageDirect = msgtype.MsgDirect_Response

	return &errMessage
}

func Response(msgType string, payload interface{}) *msg1.ChainMessage {
	responseMessage := msg1.ChainMessage{}
	responseMessage.Payload = payload
	responseMessage.MessageType = msgType
	responseMessage.MessageDirect = msgtype.MsgDirect_Response
	responseMessage.StatusCode = http.StatusOK

	return &responseMessage
}

func Ok(msgType string) *msg1.ChainMessage {
	okMessage := msg1.ChainMessage{}
	okMessage.Payload = msgtype.OK
	okMessage.PayloadType = PayloadType_String
	okMessage.Tip = "OK"
	okMessage.MessageType = msgType
	okMessage.MessageDirect = msgtype.MsgDirect_Response
	okMessage.StatusCode = http.StatusOK

	return &okMessage
}

func SetResponse(request *msg1.ChainMessage, response *msg1.ChainMessage) {
	response.UUID = request.UUID
	response.SrcConnectAddress = request.SrcConnectAddress
	response.SrcPeerId = request.SrcPeerId
	response.ConnectAddress = request.ConnectAddress
	response.ConnectPeerId = request.ConnectPeerId
	response.ConnectSessionId = request.ConnectSessionId
	response.Topic = request.Topic
	response.MessageDirect = msgtype.MsgDirect_Response
	response.StatusCode = http.StatusOK
}
