package service

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/ns"
	entity2 "github.com/curltech/go-colla-node/p2p/chain/entity"
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

// DataBlock

func ValidateDB(messagePayload *msg1.MessagePayload) error {
	dataBlock := messagePayload.Payload.(*entity2.DataBlock)
	if dataBlock == nil {
		return errors.New("NullDataBlock")
	}
	srcPeer := messagePayload.SrcPeer.(*entity.PeerClient)
	srcPeerId := srcPeer.PeerId
	createPeerId := dataBlock.PeerId
	if createPeerId != srcPeerId {
		return errors.New("CreatePeerIdAndSrcPeerIdAreDifferent")
	}
	srcPublicKey := srcPeer.PublicKey
	createPublicKey := dataBlock.PublicKey
	if createPublicKey != srcPublicKey {
		return errors.New("CreatePublicKeyAndSrcPublicKeyAreDifferent")
	}
	return nil
}

func GetLocalDBs(keyKind string, createPeerId string, blockId string, receiverPeerId string, txSequenceId uint64, sliceNumber uint64) ([]*entity2.DataBlock, error) {
	var key string
	if keyKind == ns.DataBlock_Owner_KeyKind {
		if len(createPeerId) == 0 {
			return nil, errors.New("NullCreatePeerId")
		}
		key = ns.GetDataBlockOwnerKey(createPeerId)
	} else if keyKind == ns.DataBlock_KeyKind {
		if len(blockId) == 0 {
			return nil, errors.New("NullBlockId")
		}
		key = ns.GetDataBlockKey(blockId)
	} else {
		logger.Sugar.Errorf("InvalidDataBlockKeyKind: %v", keyKind)
		return nil, errors.New("InvalidDataBlockKeyKind")
	}
	rec, err := dht.PeerEndpointDHT.GetLocal(key)
	if err != nil {
		logger.Sugar.Errorf("failed to GetLocal by key: %v, err: %v", key, err)
		return nil, err
	}
	if rec != nil {
		dataBlocks := make([]*entity2.DataBlock, 0)
		err = message.Unmarshal(rec.GetValue(), &dataBlocks)
		if err != nil {
			logger.Sugar.Errorf("failed to Unmarshal record value with key: %v, err: %v", key, err)
			return nil, err
		}
		dbs := make([]*entity2.DataBlock, 0)
		for _, dataBlock := range dataBlocks {
			var receivable bool
			if len(receiverPeerId) > 0 {
				receivable = false
				for _, transactionKey := range dataBlock.TransactionKeys {
					if transactionKey.PeerId == receiverPeerId {
						receivable = true
						break
					}
				}
			}
			if ((len(receiverPeerId) == 0 && len(dataBlock.PayloadKey) == 0) || (len(receiverPeerId) > 0 && receivable == true)) &&
				(txSequenceId > 0 /* && dataBlock.TxSequenceId == uint64(txSequenceId)*/) &&
				(sliceNumber > 0 && dataBlock.SliceNumber == uint64(sliceNumber)) {
				dbs = append(dbs, dataBlock)
			}
		}
		return dbs, nil
	}

	return nil, nil
}

func PutLocalDBs(dataBlocks []*entity2.DataBlock) error {
	for _, dataBlock := range dataBlocks {
		key := ns.GetDataBlockKey(dataBlock.BlockId)
		byteDataBlock, err := message.Marshal(dataBlock)
		if err != nil {
			return err
		}
		err = dht.PeerEndpointDHT.PutLocal(key, byteDataBlock)
		if err != nil {
			return err
		}
	}

	return nil
}

func PutDBs(dataBlock *entity2.DataBlock) error {
	err := PutDB(dataBlock, ns.DataBlock_KeyKind)
	if err != nil {
		return err
	}
	return PutDB(dataBlock, ns.DataBlock_Owner_KeyKind)
}

func PutDB(dataBlock *entity2.DataBlock, keyKind string) error {
	byteDataBlock, err := message.Marshal(dataBlock)
	if err != nil {
		return err
	}
	var key string
	if keyKind == ns.DataBlock_KeyKind {
		key = ns.GetDataBlockKey(dataBlock.BlockId)
	} else if keyKind == ns.DataBlock_Owner_KeyKind {
		key = ns.GetDataBlockOwnerKey(dataBlock.PeerId)
	} else {
		logger.Sugar.Errorf("InvalidDataBlockKeyKind: %v", keyKind)
		return errors.New("InvalidDataBlockKeyKind")
	}

	return dht.PeerEndpointDHT.PutValue(key, byteDataBlock)
}

func GetTransactionAmount(transportPayload []byte) float64 {
	return float64(len(transportPayload)) / float64(1024*1024)
}
