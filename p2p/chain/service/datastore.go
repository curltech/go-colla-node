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

// PeerEndPoint

func GetLocalPEP(peerId string) ([]*entity.PeerEndpoint, error) {
	key := ns.GetPeerEndpointKey(peerId)
	rec, err := dht.PeerEndpointDHT.GetLocal(key)
	if err != nil {
		logger.Errorf("failed to GetLocal by key: %v, err: %v", key, err)
		return nil, err
	}
	if rec != nil {
		peerEndpoints := make([]*entity.PeerEndpoint, 0)
		err = message.Unmarshal(rec.GetValue(), &peerEndpoints)
		if err != nil {
			logger.Errorf("failed to Unmarshal record value with key: %v, err: %v", key, err)
			return nil, err
		}
		return peerEndpoints, nil
	}

	return nil, nil
}

func PutLocalPEP(peerEndpoint *entity.PeerEndpoint) error {
	key := ns.GetPeerEndpointKey(peerEndpoint.PeerId)
	bytePeerEndpoint, err := message.Marshal(peerEndpoint)
	if err != nil {
		return err
	}
	return dht.PeerEndpointDHT.PutLocal(key, bytePeerEndpoint)
}

// PeerClient

func ValidatePC(messagePayload *msg1.MessagePayload) error {
	peerClient := messagePayload.Payload.(*entity.PeerClient)
	if peerClient == nil {
		return errors.New("NullPeerClient")
	}
	srcPeer := messagePayload.SrcPeer.(*entity.PeerClient)
	srcPeerId := srcPeer.PeerId
	senderPeerId := peerClient.PeerId
	if senderPeerId != srcPeerId {
		return errors.New("SenderAndSrcPeerPeerIdAreDifferent")
	}
	srcPublicKey := srcPeer.PublicKey
	senderPublicKey := peerClient.PublicKey
	if senderPublicKey != srcPublicKey {
		return errors.New("SenderAndSrcPeerPublicKeyAreDifferent")
	}
	expireDate := peerClient.ExpireDate
	if expireDate == 0 {
		return errors.New("Invalid expireDate")
	}
	return nil
}

func GetLocalPCs(keyKind string, peerId string, mobile string, clientId string) ([]*entity.PeerClient, error) {
	var key string
	if keyKind == ns.PeerClient_KeyKind {
		if len(peerId) == 0 {
			return nil, errors.New("NullPeerId")
		}
		key = ns.GetPeerClientKey(peerId)
	} else if keyKind == ns.PeerClient_Mobile_KeyKind {
		if len(mobile) == 0 {
			return nil, errors.New("NullMobile")
		}
		key = ns.GetPeerClientMobileKey(mobile)
	} else {
		logger.Errorf("InvalidPeerClientKeyKind: %v", keyKind)
		return nil, errors.New("InvalidPeerClientKeyKind")
	}
	rec, err := dht.PeerEndpointDHT.GetLocal(key)
	if err != nil {
		logger.Errorf("failed to GetLocal by key: %v, err: %v", key, err)
		return nil, err
	}
	if rec != nil {
		peerClients := make([]*entity.PeerClient, 0)
		err = message.Unmarshal(rec.GetValue(), &peerClients)
		if err != nil {
			logger.Errorf("failed to Unmarshal record value with key: %v, err: %v", key, err)
			return nil, err
		}
		if len(clientId) > 0 {
			pcs := make([]*entity.PeerClient, 0)
			for _, peerClient := range peerClients {
				if peerClient.ClientId == clientId {
					pcs = append(pcs, peerClient)
				}
			}
			return pcs, nil
		} else {
			return peerClients, nil
		}
	}

	return nil, nil
}

func PutLocalPCs(peerClients []*entity.PeerClient) error {
	for _, peerClient := range peerClients {
		key := ns.GetPeerClientKey(peerClient.PeerId)
		bytePeerClient, err := message.Marshal(peerClient)
		if err != nil {
			return err
		}
		err = dht.PeerEndpointDHT.PutLocal(key, bytePeerClient)
		if err != nil {
			return err
		}
	}

	return nil
}

func PutPCs(peerClient *entity.PeerClient) error {
	err := PutPC(peerClient, ns.PeerClient_KeyKind)
	if err != nil {
		return err
	}
	return PutPC(peerClient, ns.PeerClient_Mobile_KeyKind)
}

func PutPC(peerClient *entity.PeerClient, keyKind string) error {
	bytePeerClient, err := message.Marshal(peerClient)
	if err != nil {
		return err
	}
	var key string
	if keyKind == ns.PeerClient_KeyKind {
		key = ns.GetPeerClientKey(peerClient.PeerId)
	} else if keyKind == ns.PeerClient_Mobile_KeyKind {
		key = ns.GetPeerClientMobileKey(peerClient.Mobile)
	} else {
		logger.Errorf("InvalidPeerClientKeyKind: %v", keyKind)
		return errors.New("InvalidPeerClientKeyKind")
	}

	return dht.PeerEndpointDHT.PutValue(key, bytePeerClient)
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
		logger.Errorf("InvalidDataBlockKeyKind: %v", keyKind)
		return nil, errors.New("InvalidDataBlockKeyKind")
	}
	rec, err := dht.PeerEndpointDHT.GetLocal(key)
	if err != nil {
		logger.Errorf("failed to GetLocal by key: %v, err: %v", key, err)
		return nil, err
	}
	if rec != nil {
		dataBlocks := make([]*entity2.DataBlock, 0)
		err = message.Unmarshal(rec.GetValue(), &dataBlocks)
		if err != nil {
			logger.Errorf("failed to Unmarshal record value with key: %v, err: %v", key, err)
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
				(txSequenceId > 0 && dataBlock.TxSequenceId == uint64(txSequenceId)) &&
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
		logger.Errorf("InvalidDataBlockKeyKind: %v", keyKind)
		return errors.New("InvalidDataBlockKeyKind")
	}

	return dht.PeerEndpointDHT.PutValue(key, byteDataBlock)
}

func GetTransactionAmount(transportPayload []byte) float64 {
	return float64(len(transportPayload)) / float64(1024*1024)
}

// PeerTransaction

func GetLocalPTs(keyKind string, srcPeerId string, targetPeerId string) ([]*entity2.PeerTransaction, error) {
	var key string
	if keyKind == ns.PeerTransaction_Src_KeyKind {
		if len(srcPeerId) == 0 {
			return nil, errors.New("NullSrcPeerId")
		}
		key = ns.GetPeerTransactionSrcKey(srcPeerId)
	} else if keyKind == ns.PeerTransaction_Target_KeyKind {
		if len(targetPeerId) == 0 {
			return nil, errors.New("NullTargetPeerId")
		}
		key = ns.GetPeerTransactionTargetKey(targetPeerId)
	} else {
		logger.Errorf("InvalidPeerTransactionKeyKind: %v", keyKind)
		return nil, errors.New("InvalidPeerTransactionKeyKind")
	}
	rec, err := dht.PeerEndpointDHT.GetLocal(key)
	if err != nil {
		logger.Errorf("failed to GetLocal by key: %v, err: %v", key, err)
		return nil, err
	}
	if rec != nil {
		peerTransactions := make([]*entity2.PeerTransaction, 0)
		err = message.Unmarshal(rec.GetValue(), &peerTransactions)
		if err != nil {
			logger.Errorf("failed to Unmarshal record value with key: %v, err: %v", key, err)
			return nil, err
		}
		pts := make([]*entity2.PeerTransaction, 0)
		for _, peerTransaction := range peerTransactions {
			pts = append(pts, peerTransaction)
		}
		return pts, nil
	}

	return nil, nil
}

func PutLocalPTs(peerTransactions []*entity2.PeerTransaction) error {
	for _, peerTransaction := range peerTransactions {
		key := ns.GetPeerTransactionSrcKey(peerTransaction.SrcPeerId)
		bytePeerTransaction, err := message.Marshal(peerTransaction)
		if err != nil {
			return err
		}
		err = dht.PeerEndpointDHT.PutLocal(key, bytePeerTransaction)
		if err != nil {
			return err
		}
	}

	return nil
}

func PutPTs(peerTransaction *entity2.PeerTransaction) error {
	err := PutPT(peerTransaction, ns.PeerTransaction_Src_KeyKind)
	if err != nil {
		return err
	}
	return PutPT(peerTransaction, ns.PeerTransaction_Target_KeyKind)
}

func PutPT(peerTransaction *entity2.PeerTransaction, keyKind string) error {
	bytePeerTransaction, err := message.Marshal(peerTransaction)
	if err != nil {
		return err
	}
	var key string
	if keyKind == ns.PeerTransaction_Src_KeyKind {
		key = ns.GetPeerTransactionSrcKey(peerTransaction.SrcPeerId)
	} else if keyKind == ns.PeerTransaction_Target_KeyKind {
		key = ns.GetPeerTransactionTargetKey(peerTransaction.TargetPeerId)
	} else {
		logger.Errorf("InvalidPeerTransactionKeyKind: %v", keyKind)
		return errors.New("InvalidPeerTransactionKeyKind")
	}

	return dht.PeerEndpointDHT.PutValue(key, bytePeerTransaction)
}
