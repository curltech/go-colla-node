package service

import (
	"errors"
	"fmt"
	"github.com/curltech/go-colla-core/container"
	"github.com/curltech/go-colla-core/crypto/openpgp"
	"github.com/curltech/go-colla-core/crypto/std"
	baseentity "github.com/curltech/go-colla-core/entity"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/service"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/chain/entity"
	handler2 "github.com/curltech/go-colla-node/p2p/chain/handler"
	entity2 "github.com/curltech/go-colla-node/p2p/dht/entity"
	"strconv"
	"time"
)

/**
同步表结构，服务继承基本服务的方法
*/
type DataBlockService struct {
	service.OrmBaseService
}

var dataBlockService = &DataBlockService{}

func GetDataBlockService() *DataBlockService {
	return dataBlockService
}

var seqname = "seq_block"

func (this *DataBlockService) GetSeqName() string {
	return seqname
}

func (this *DataBlockService) NewEntity(data []byte) (interface{}, error) {
	entity := &entity.DataBlock{}
	if data == nil {
		return entity, nil
	}
	err := message.Unmarshal(data, entity)
	if err != nil {
		return nil, err
	}

	return entity, err
}

func (this *DataBlockService) NewEntities(data []byte) (interface{}, error) {
	entities := make([]*entity.DataBlock, 0)
	if data == nil {
		return &entities, nil
	}
	err := message.Unmarshal(data, &entities)
	if err != nil {
		return nil, err
	}

	return &entities, err
}

func (this *DataBlockService) Validate(dataBlock *entity.DataBlock) error {
	transportPayload := dataBlock.TransportPayload
	expireDate := dataBlock.ExpireDate
	if len(transportPayload) == 0 && expireDate == 0 {
		return errors.New("Invalid expireDate")
	}
	return nil
}

func (this *DataBlockService) GetLocalDBs(keyKind string, createPeerId string, blockId string, receiverPeerId string, sliceNumber uint64) ([]*entity.DataBlock, error) {
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
		dataBlocks := make([]*entity.DataBlock, 0)
		err = message.Unmarshal(rec.GetValue(), &dataBlocks)
		if err != nil {
			logger.Sugar.Errorf("failed to Unmarshal record value with key: %v, err: %v", key, err)
			return nil, err
		}
		dbs := make([]*entity.DataBlock, 0)
		for _, dataBlock := range dataBlocks {
			var receivable bool
			if len(receiverPeerId) > 0 {
				if sliceNumber != 1 {
					receivable = true
				} else {
					receivable = false
					for _, transactionKey := range dataBlock.TransactionKeys {
						if transactionKey.PeerId == receiverPeerId {
							receivable = true
							break
						}
					}
				}
			}
			if ((len(receiverPeerId) == 0 && len(dataBlock.PayloadKey) == 0) || (len(receiverPeerId) > 0 && receivable == true)) &&
				(sliceNumber > 0 && dataBlock.SliceNumber == uint64(sliceNumber)) {
				dbs = append(dbs, dataBlock)
			}
		}
		return dbs, nil
	}

	return nil, nil
}

func (this *DataBlockService) PutLocalDBs(keyKind string, dataBlocks []*entity.DataBlock) error {
	var key string
	for _, dataBlock := range dataBlocks {
		if keyKind == ns.DataBlock_Owner_KeyKind {
			if len(dataBlock.PeerId) == 0 {
				return errors.New("NullPeerId")
			}
			key = ns.GetDataBlockOwnerKey(dataBlock.PeerId)
		} else if keyKind == ns.DataBlock_KeyKind {
			if len(dataBlock.BlockId) == 0 {
				return errors.New("NullBlockId")
			}
			key = ns.GetDataBlockKey(dataBlock.BlockId)
		} else {
			logger.Sugar.Errorf("InvalidDataBlockKeyKind: %v", keyKind)
			return errors.New("InvalidDataBlockKeyKind")
		}
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

func (this *DataBlockService) PutDBs(dataBlock *entity.DataBlock) error {
	err := this.PutDB(dataBlock, ns.DataBlock_KeyKind)
	if err != nil {
		return err
	}
	return this.PutDB(dataBlock, ns.DataBlock_Owner_KeyKind)
}

func (this *DataBlockService) PutDB(dataBlock *entity.DataBlock, keyKind string) error {
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

func (this *DataBlockService) StoreValue(db *entity.DataBlock) error {
	if db == nil {
		logger.Sugar.Errorf("NoDataBlock")
		return errors.New("NoDataBlock")
	}
	blockId := db.BlockId
	if blockId == "" {
		logger.Sugar.Errorf("NoBlockId")
		return errors.New("NoBlockId")
	}
	sliceNumber := db.SliceNumber
	if sliceNumber == 0 {
		logger.Sugar.Errorf("NoSliceNumber")
		return errors.New("NoSliceNumber")
	}
	transactionKeys := db.TransactionKeys
	if sliceNumber == 1 && transactionKeys == nil {
		logger.Sugar.Warnf("NoTransactionKeys")
	}
	businessNumber := db.BusinessNumber
	if businessNumber == "" {
		logger.Sugar.Errorf("NoBusinessNumber")
		return errors.New("NoBusinessNumber")
	}
	myselfPeerId := global.Global.MyselfPeer.PeerId
	currentTime := time.Now()
	oldDb := &entity.DataBlock{}
	oldDb.BlockId = blockId
	oldDb.SliceNumber = sliceNumber
	dbFound := this.Get(oldDb, false, "", "")
	if dbFound {
		db.Id = oldDb.Id
		// 校验Owner
		if db.BlockType == entity.BlockType_P2pChat && len(db.TransportPayload) == 0 {
			if oldDb.BusinessNumber != db.PeerId {
				return errors.New(fmt.Sprintf("InconsistentDataBlockPeerId, blockId: %v, peerId: %v, oldBusinessNumber: %v", db.BlockId, db.PeerId, oldDb.BusinessNumber))
			} else {
				// 校验Signature
				if db.ExpireDate > 0 {
					publicKey, err := handler2.GetPublicKey(oldDb.BusinessNumber)
					if err != nil {
						return errors.New(fmt.Sprintf("GetPublicKey failure, blockId: %v, oldBusinessNumber: %v", db.BlockId, oldDb.BusinessNumber))
					} else {
						signatureData := strconv.FormatInt(db.ExpireDate, 10) + db.PeerId
						signature := std.DecodeBase64(db.Signature)
						pass := openpgp.Verify(publicKey, []byte(signatureData), signature)
						if pass != true {
							return errors.New(fmt.Sprintf("SignatureVerifyFailure, blockId: %v, PeerId: %v", db.BlockId, db.PeerId))
						}
					}
				}
			}
		} else {
			if oldDb.PeerId != db.PeerId {
				return errors.New(fmt.Sprintf("InconsistentDataBlockPeerId, blockId: %v, peerId: %v, oldPeerId: %v", db.BlockId, db.PeerId, oldDb.PeerId))
			} else {
				// 校验Signature
				publicKey, err := handler2.GetPublicKey(oldDb.PeerId)
				if err != nil {
					return errors.New(fmt.Sprintf("GetPublicKey failure, blockId: %v, oldPeerId: %v", db.BlockId, oldDb.PeerId))
				} else {
					var signatureData string
					if len(db.TransportPayload) > 0 {
						signatureData = db.TransportPayload
					} else if db.ExpireDate > 0 {
						signatureData = strconv.FormatInt(db.ExpireDate, 10) + db.PeerId
					}
					if len(signatureData) > 0 {
						signature := std.DecodeBase64(db.Signature)
						pass := openpgp.Verify(publicKey, []byte(signatureData), signature)
						if pass != true {
							return errors.New(fmt.Sprintf("SignatureVerifyFailure, blockId: %v, PeerId: %v", db.BlockId, db.PeerId))
						}
					}
				}
			}
		}
		// 检查时间戳
		if db.CreateTimestamp <= oldDb.CreateTimestamp {
			return errors.New("can't replace a newer value with an older value")
		}
		// 负载为空表示删除
		if len(db.TransportPayload) == 0 {
			// 只针对第一个分片处理一次
			if sliceNumber == 1 {
				dbCondition := &entity.DataBlock{}
				dbCondition.BlockId = blockId
				this.Delete(dbCondition, "")
				// 删除TransactionKeys
				tkCondition := &entity.TransactionKey{}
				tkCondition.BlockId = blockId
				GetTransactionKeyService().Delete(tkCondition, "")
				// 删除PeerTransaction
				for i := uint64(1); i <= oldDb.SliceSize; i++ {
					peerTransaction := entity.PeerTransaction{}
					peerTransaction.SrcPeerId = db.PeerId
					peerTransaction.TargetPeerId = myselfPeerId
					peerTransaction.BlockId = blockId
					peerTransaction.SliceNumber = i
					peerTransaction.TransactionType = fmt.Sprintf("%v-%v", entity2.TransactionType_DataBlock, db.BlockType)
					peerTransaction.BusinessNumber = db.BusinessNumber
					peerTransaction.Status = baseentity.EntityState_Deleted
					err := GetPeerTransactionService().PutPTs(&peerTransaction)
					if err != nil {
						return err
					}
				}
			}
			return nil
		}
	} else {
		db.Id = uint64(0)
		// 负载为空表示删除
		if len(db.TransportPayload) == 0 {
			return nil
		}
	}

	dbAffected := this.Upsert(db)
	if dbAffected > 0 {
		logger.Sugar.Infof("BlockId: %v, upsert DataBlock successfully", blockId)
		// 只针对第一个分片处理一次
		if sliceNumber == 1 {
			// 删除多余废弃分片
			if db.SliceSize < oldDb.SliceSize {
				dbCondition := &entity.DataBlock{}
				dbCondition.BlockId = blockId
				this.Delete(dbCondition, "SliceNumber > ?", db.SliceSize)
				// 删除PeerTransaction
				for i := db.SliceSize + 1; i <= oldDb.SliceSize; i++ {
					peerTransaction := entity.PeerTransaction{}
					peerTransaction.SrcPeerId = db.PeerId
					peerTransaction.TargetPeerId = myselfPeerId
					peerTransaction.BlockId = blockId
					peerTransaction.SliceNumber = i
					peerTransaction.TransactionType = fmt.Sprintf("%v-%v", entity2.TransactionType_DataBlock, db.BlockType)
					peerTransaction.BusinessNumber = db.BusinessNumber
					peerTransaction.Status = baseentity.EntityState_Deleted
					err := GetPeerTransactionService().PutPTs(&peerTransaction)
					if err != nil {
						return err
					}
				}
			}
			// 保存TransactionKeys
			for _, tk := range transactionKeys {
				tkBlockId := tk.BlockId
				if tkBlockId != blockId {
					logger.Sugar.Errorf("InvalidTKBlockId")
					return errors.New("InvalidTKBlockId")
				}
				tkPeerId := tk.PeerId
				if tkPeerId == "" {
					logger.Sugar.Errorf("NoTKPeerId")
					return errors.New("NoTKPeerId")
				}
				oldTk := &entity.TransactionKey{}
				oldTk.BlockId = tkBlockId
				oldTk.PeerId = tkPeerId
				tkFound := GetTransactionKeyService().Get(oldTk, false, "", "")
				if tkFound {
					tk.Id = oldTk.Id
				} else {
					tk.Id = uint64(0)
				}

				tkAffected := GetTransactionKeyService().Upsert(tk)
				if tkAffected > 0 {
					logger.Sugar.Infof("BlockId: %v, PeerId: %v, upsert TransactionKey successfully", tkBlockId, tkPeerId)
				} else {
					logger.Sugar.Errorf("BlockId: %v, PeerId: %v, upsert TransactionKey fail", tkBlockId, tkPeerId)
					return errors.New(fmt.Sprintf("BlockId: %v, PeerId: %v, upsert TransactionKey fail", tkBlockId, tkPeerId))
				}
			}
		}
		// 更新交易金额
		// MyselfPeer
		/*global.Global.MyselfPeer.BlockId = blockId
		global.Global.MyselfPeer.LastTransactionTime = &currentTime
		global.Global.MyselfPeer.Balance = global.Global.MyselfPeer.Balance + db.TransactionAmount
		affected := service.GetMyselfPeerService().Update([]interface{}{global.Global.MyselfPeer}, nil, "")
		if affected == 0 {
			return errors.New("NoUpdateOfMyselfPeer")
		}
		// PeerEndpoint
		dht.PeerEndpointDHT.PutMyself()
		// PeerClient
		pcs, err := service1.GetLocalPCs(ns.PeerClient_KeyKind, db.PeerId, "", "") // 可能查不到或查到的为旧版本
		if err != nil {
			return err
		}
		for _, pc := range pcs {
			pc.LastAccessTime = &currentTime
			pc.BlockId = blockId
			pc.LastTransactionTime = &currentTime
			pc.Balance = pc.Balance - db.TransactionAmount
			err := service1.PutPCs(pc)
			if err != nil {
				return err
			}
		}*/
		// PeerTransaction（BlockType_ChatAttach不需要保存PeerTransaction）
		if db.BlockType != entity.BlockType_ChatAttach {
			peerTransaction := entity.PeerTransaction{}
			peerTransaction.SrcPeerId = db.PeerId
			peerTransaction.SrcPeerType = entity2.PeerType_PeerClient
			peerTransaction.TargetPeerId = myselfPeerId
			peerTransaction.TargetPeerType = entity2.PeerType_PeerEndpoint
			peerTransaction.BlockId = blockId
			peerTransaction.SliceNumber = sliceNumber
			peerTransaction.BusinessNumber = db.BusinessNumber
			peerTransaction.TransactionTime = &currentTime
			peerTransaction.CreateTimestamp = db.CreateTimestamp
			peerTransaction.Amount = db.TransactionAmount
			peerTransaction.TransactionType = fmt.Sprintf("%v-%v", entity2.TransactionType_DataBlock, db.BlockType)
			err := GetPeerTransactionService().PutPTs(&peerTransaction)
			if err != nil {
				return err
			}
		}
	} else {
		logger.Sugar.Errorf("BlockId: %v, upsert DataBlock fail", blockId)
		return errors.New(fmt.Sprintf("BlockId: %v, upsert DataBlock fail", blockId))
	}

	return nil
}

func (this *DataBlockService) QueryValue(dataBlocks *[]*entity.DataBlock, blockId string, sliceNumber uint64) error {
	condition := &entity.DataBlock{}
	condition.BlockId = blockId
	condition.SliceNumber = sliceNumber
	this.Find(dataBlocks, condition, "", 0, 0, "")
	if len(*dataBlocks) > 0 && sliceNumber == 1 {
		for _, dataBlock := range *dataBlocks {
			condition := &entity.TransactionKey{}
			condition.BlockId = dataBlock.BlockId
			transactionKeys := make([]*entity.TransactionKey, 0)
			GetTransactionKeyService().Find(&transactionKeys, condition, "", 0, 0, "")
			if len(transactionKeys) > 0 {
				dataBlock.TransactionKeys = transactionKeys
			}
		}
	}

	return nil
}

func (this *DataBlockService) GetTransactionAmount(transportPayload []byte) float64 {
	return float64(len(transportPayload)) / float64(1024*1024)
}

func init() {
	service.GetSession().Sync(new(entity.DataBlock))

	dataBlockService.OrmBaseService.GetSeqName = dataBlockService.GetSeqName
	dataBlockService.OrmBaseService.FactNewEntity = dataBlockService.NewEntity
	dataBlockService.OrmBaseService.FactNewEntities = dataBlockService.NewEntities
	service.RegistSeq(seqname, 0)
	container.RegistService(ns.DataBlock_Prefix, dataBlockService)
	container.RegistService(ns.DataBlock_Owner_Prefix, dataBlockService)
}
