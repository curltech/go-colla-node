package ns

import (
	"errors"
	"fmt"
	"github.com/curltech/go-colla-core/crypto/std"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	entity2 "github.com/curltech/go-colla-node/p2p/chain/entity"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	record "github.com/libp2p/go-libp2p-record"
)

const PeerEndpoint_Prefix = "peerEndpoint"
const PeerClient_Prefix = "peerClient"
const PeerClient_Mobile_Prefix = "peerClientMobile"
const PeerClient_Name_Prefix = "peerClientName"
const ChainApp_Prefix = "chainApp"
const DataBlock_Prefix = "dataBlock"
const DataBlock_Owner_Prefix = "dataBlockOwner"
const PeerTransaction_Src_Prefix = "peerTransactionSrc"
const PeerTransaction_Target_Prefix = "peerTransactionTarget"
const PeerTransaction_P2PChat_Prefix = "peerTransactionP2PChat"
const PeerTransaction_GroupFile_Prefix = "peerTransactionGroupFile"
const PeerTransaction_Channel_Prefix = "peerTransactionChannel"
const PeerTransaction_ChannelArticle_Prefix = "peerTransactionChannelArticle"
const TransactionKey_Prefix = "transactionKey"

const PeerClient_KeyKind = "PeerId"
const PeerClient_Mobile_KeyKind = "Mobile"
const PeerClient_Name_KeyKind = "Name"

const DataBlock_KeyKind = "BlockId"
const DataBlock_Owner_KeyKind = "PeerId"

const PeerTransaction_Src_KeyKind = "SrcPeerId"
const PeerTransaction_Target_KeyKind = "TargetPeerId"

const PeerTransaction_P2PChat_KeyKind = "BusinessNumber"

const PeerTransaction_GroupFile_KeyKind = "BusinessNumber"

const PeerTransaction_Channel_KeyKind = "TransactionType"

const PeerTransaction_ChannelArticle_KeyKind = "ParentBusinessNumber"

const PeerTransaction_Type_KeyKind = "TransactionType"

func GetPeerEndpointKey(id string) string {
	key := fmt.Sprintf("/%v/%v", PeerEndpoint_Prefix, id)

	return key
}

func GetPeerClientKey(peerId string) string {
	key := fmt.Sprintf("/%v/%v", PeerClient_Prefix, peerId)

	return key
}

func GetPeerClientMobileKey(mobile string, hashed bool) string {
	mobileHash := mobile
	if hashed == false {
		mobileHash = std.EncodeBase64(std.Hash(mobile, "sha3_256"))
	}
	key := fmt.Sprintf("/%v/%v", PeerClient_Mobile_Prefix, mobileHash)

	return key
}

func GetPeerClientNameKey(name string) string {
	key := fmt.Sprintf("/%v/%v", PeerClient_Name_Prefix, name)

	return key
}

func GetChainAppKey(id string) string {
	key := fmt.Sprintf("/%v/%v", ChainApp_Prefix, id)

	return key
}

func GetDataBlockKey(blockId string) string {
	key := fmt.Sprintf("/%v/%v", DataBlock_Prefix, blockId)

	return key
}

func GetDataBlockOwnerKey(createPeerId string) string {
	key := fmt.Sprintf("/%v/%v", DataBlock_Owner_Prefix, createPeerId)

	return key
}

func GetTransactionKeyKey(id string) string {
	key := fmt.Sprintf("/%v/%v", TransactionKey_Prefix, id)

	return key
}

func GetPeerTransactionSrcKey(id string) string {
	key := fmt.Sprintf("/%v/%v", PeerTransaction_Src_Prefix, id)

	return key
}

func GetPeerTransactionTargetKey(id string) string {
	key := fmt.Sprintf("/%v/%v", PeerTransaction_Target_Prefix, id)

	return key
}

func GetPeerTransactionP2pChatKey(id string) string {
	key := fmt.Sprintf("/%v/%v", PeerTransaction_P2PChat_Prefix, id)

	return key
}

func GetPeerTransactionGroupFileKey(id string) string {
	key := fmt.Sprintf("/%v/%v", PeerTransaction_GroupFile_Prefix, id)

	return key
}

func GetPeerTransactionChannelKey(id string) string {
	key := fmt.Sprintf("/%v/%v", PeerTransaction_Channel_Prefix, id)

	return key
}

func GetPeerTransactionChannelArticleKey(id string) string {
	key := fmt.Sprintf("/%v/%v", PeerTransaction_ChannelArticle_Prefix, id)

	return key
}

type PeerEndpointValidator struct {
}

// Validate conforms to the Validator interface.
func (v PeerEndpointValidator) Validate(key string, value []byte) error {
	ns, key, err := record.SplitKey(key)
	if err != nil {
		return err
	}
	if ns != PeerEndpoint_Prefix {
		return errors.New("invalid namespace:" + ns)
	}

	return nil
}

// Select conforms to the Validator interface.
func (v PeerEndpointValidator) Select(key string, vals [][]byte) (int, error) {
	currentVal := vals[0]
	existingVal := vals[1]
	currentEntity := entity.PeerEndpoint{}
	err := message.Unmarshal(currentVal, &currentEntity)
	if err != nil {
		logger.Sugar.Errorf("failed to unmarshal current record from value", "key", key, "error", err)
		return 1, err
	}
	existingEntities := make([]*entity.PeerEndpoint, 0)
	err = message.Unmarshal(existingVal, &existingEntities)
	if err != nil {
		logger.Sugar.Errorf("failed to unmarshal existing records from value", "key", key, "error", err)
		return 1, err
	}
	for _, existingEntity := range existingEntities {
		if existingEntity.PeerId == currentEntity.PeerId &&
			currentEntity.LastUpdateTime != nil && existingEntity.LastUpdateTime != nil &&
			currentEntity.LastUpdateTime.UTC().Before(existingEntity.LastUpdateTime.UTC()) {
			return 1, nil
		}
	}

	return 0, nil
}

var _ record.Validator = PeerEndpointValidator{}

type PeerClientValidator struct {
}

// Validate conforms to the Validator interface.
func (v PeerClientValidator) Validate(key string, value []byte) error {
	ns, key, err := record.SplitKey(key)
	if err != nil {
		return err
	}
	if ns != PeerClient_Prefix && ns != PeerClient_Mobile_Prefix && ns != PeerClient_Name_Prefix {
		return errors.New("invalid namespace:" + ns)
	}

	return nil
}

// Select conforms to the Validator interface.
func (v PeerClientValidator) Select(key string, vals [][]byte) (int, error) {
	currentVal := vals[0]
	existingVal := vals[1]
	currentEntity := entity.PeerClient{}
	err := message.Unmarshal(currentVal, &currentEntity)
	if err != nil {
		logger.Sugar.Errorf("failed to unmarshal current record from value", "key", key, "error", err)
		return 1, err
	}
	existingEntities := make([]*entity.PeerClient, 0)
	err = message.Unmarshal(existingVal, &existingEntities)
	if err != nil {
		logger.Sugar.Errorf("failed to unmarshal existing records from value", "key", key, "error", err)
		return 1, err
	}
	for _, existingEntity := range existingEntities {
		if existingEntity.PeerId == currentEntity.PeerId && existingEntity.ClientId == currentEntity.ClientId &&
			currentEntity.LastUpdateTime != nil && existingEntity.LastUpdateTime != nil &&
			(currentEntity.LastUpdateTime.UTC().Before(existingEntity.LastUpdateTime.UTC()) ||
				(currentEntity.LastUpdateTime.UTC().Equal(existingEntity.LastUpdateTime.UTC()) &&
					currentEntity.LastAccessTime != nil && existingEntity.LastAccessTime != nil &&
					currentEntity.LastAccessTime.UTC().Before(existingEntity.LastAccessTime.UTC()))) {
			return 1, nil
		}
	}

	return 0, nil
}

var _ record.Validator = PeerClientValidator{}

type ChainAppValidator struct {
}

// Validate conforms to the Validator interface.
func (v ChainAppValidator) Validate(key string, value []byte) error {
	ns, key, err := record.SplitKey(key)
	if err != nil {
		return err
	}
	if ns != ChainApp_Prefix {
		return errors.New("invalid namespace:" + ns)
	}

	return nil
}

// Select conforms to the Validator interface.
func (v ChainAppValidator) Select(key string, vals [][]byte) (int, error) {
	return 0, nil
}

var _ record.Validator = ChainAppValidator{}

type DataBlockValidator struct {
}

// Validate conforms to the Validator interface.
func (v DataBlockValidator) Validate(key string, value []byte) error {
	ns, key, err := record.SplitKey(key)
	if err != nil {
		return err
	}
	if ns != DataBlock_Prefix && ns != DataBlock_Owner_Prefix {
		return errors.New("invalid namespace:" + ns)
	}

	/*entities := make([]*entity2.DataBlock, 0)
	err = message.Unmarshal(value, &entities)
	if err != nil {
		p := entity2.DataBlock{}
		err = message.Unmarshal(value, &p)
		if err != nil {
			logger.Sugar.Errorf("failed to unmarshal record from value", "key", key, "error", err)
			return err
		}
		entities = append(entities, &p)
	}
	// 校验Hash
	for _, p := range entities {
		transportPayload := p.TransportPayload
		if len(transportPayload) > 0 {
			payloadHash := p.PayloadHash
			hash := std.EncodeBase64(std.Hash(transportPayload, "sha3_256"))
			if payloadHash != hash {
				logger.Sugar.Errorf("VerifyHashFailed", "key", key)
				return err
			}
		}
	}*/

	return nil
}

// Select conforms to the Validator interface.
func (v DataBlockValidator) Select(key string, vals [][]byte) (int, error) {
	currentVal := vals[0]
	existingVal := vals[1]
	currentEntity := entity2.DataBlock{}
	err := message.Unmarshal(currentVal, &currentEntity)
	if err != nil {
		logger.Sugar.Errorf("failed to unmarshal current record from value", "key", key, "error", err)
		return 1, err
	}
	existingEntities := make([]*entity2.DataBlock, 0)
	err = message.Unmarshal(existingVal, &existingEntities)
	if err != nil {
		logger.Sugar.Errorf("failed to unmarshal existing records from value", "key", key, "error", err)
		return 1, err
	}
	for _, existingEntity := range existingEntities {
		if existingEntity.BlockId == currentEntity.BlockId &&
			existingEntity.SliceNumber == currentEntity.SliceNumber && currentEntity.CreateTimestamp <= existingEntity.CreateTimestamp {
			return 1, nil
		}
	}

	return 0, nil
}

var _ record.Validator = DataBlockValidator{}

type PeerTransactionValidator struct {
}

// Validate conforms to the Validator interface.
func (v PeerTransactionValidator) Validate(key string, value []byte) error {
	ns, key, err := record.SplitKey(key)
	if err != nil {
		return err
	}
	if ns != PeerTransaction_Src_Prefix && ns != PeerTransaction_Target_Prefix &&
		ns != PeerTransaction_P2PChat_Prefix && ns != PeerTransaction_GroupFile_Prefix &&
		ns != PeerTransaction_Channel_Prefix && ns != PeerTransaction_ChannelArticle_Prefix {
		return errors.New("invalid namespace:" + ns)
	}

	return nil
}

// Select conforms to the Validator interface.
func (v PeerTransactionValidator) Select(key string, vals [][]byte) (int, error) {
	return 0, nil
}

var _ record.Validator = PeerTransactionValidator{}

type TransactionKeyValidator struct {
}

// Validate conforms to the Validator interface.
func (v TransactionKeyValidator) Validate(key string, value []byte) error {
	ns, key, err := record.SplitKey(key)
	if err != nil {
		return err
	}
	if ns != TransactionKey_Prefix {
		return errors.New("invalid namespace:" + ns)
	}

	return nil
}

// Select conforms to the Validator interface.
func (v TransactionKeyValidator) Select(key string, vals [][]byte) (int, error) {
	return 0, nil
}

var _ record.Validator = TransactionKeyValidator{}
