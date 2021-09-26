package service

import (
	"errors"
	"fmt"
	"github.com/curltech/go-colla-core/container"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/service"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/chain/entity"
	entity2 "github.com/curltech/go-colla-node/p2p/dht/entity"
	chainentity "github.com/curltech/go-colla-node/p2p/chain/entity"
)

/**
同步表结构，服务继承基本服务的方法
*/
type PeerTransactionService struct {
	service.OrmBaseService
}

var peerTransactionService = &PeerTransactionService{}

func GetPeerTransactionService() *PeerTransactionService {
	return peerTransactionService
}

func (this *PeerTransactionService) GetSeqName() string {
	return seqname
}

func (this *PeerTransactionService) NewEntity(data []byte) (interface{}, error) {
	entity := &entity.PeerTransaction{}
	if data == nil {
		return entity, nil
	}
	err := message.Unmarshal(data, entity)
	if err != nil {
		return nil, err
	}

	return entity, err
}

func (this *PeerTransactionService) NewEntities(data []byte) (interface{}, error) {
	entities := make([]*entity.PeerTransaction, 0)
	if data == nil {
		return &entities, nil
	}
	err := message.Unmarshal(data, &entities)
	if err != nil {
		return nil, err
	}

	return &entities, err
}

func (this *PeerTransactionService) GetLocalPTs(keyKind string, srcPeerId string, targetPeerId string, businessNumber string, parentBusinessNumber string) ([]*entity.PeerTransaction, error) {
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
	} else if keyKind == ns.PeerTransaction_P2PChat_KeyKind {
		if len(businessNumber) == 0 {
			return nil, errors.New("NullBusinessNumber")
		}
		key = ns.GetPeerTransactionP2pChatKey(businessNumber)
	} else if keyKind == ns.PeerTransaction_GroupFile_KeyKind {
		if len(businessNumber) == 0 {
			return nil, errors.New("NullBusinessNumber")
		}
		key = ns.GetPeerTransactionGroupFileKey(businessNumber)
	} else if keyKind == ns.PeerTransaction_Channel_KeyKind {
		/*if len(businessNumber) == 0 {
			return nil, errors.New("NullBusinessNumber")
		}
		key = ns.GetPeerTransactionChannelKey(businessNumber)*/
		key = ns.GetPeerTransactionChannelKey(fmt.Sprintf("%v-%v", entity2.TransactionType_DataBlock, chainentity.BlockType_Channel))
	} else if keyKind == ns.PeerTransaction_ChannelArticle_KeyKind {
		if len(parentBusinessNumber) == 0 {
			return nil, errors.New("NullParentBusinessNumber")
		}
		key = ns.GetPeerTransactionChannelArticleKey(parentBusinessNumber)
	} else {
		logger.Sugar.Errorf("InvalidPeerTransactionKeyKind: %v", keyKind)
		return nil, errors.New("InvalidPeerTransactionKeyKind")
	}
	rec, err := dht.PeerEndpointDHT.GetLocal(key)
	if err != nil {
		logger.Sugar.Errorf("failed to GetLocal by key: %v, err: %v", key, err)
		return nil, err
	}
	if rec != nil {
		peerTransactions := make([]*entity.PeerTransaction, 0)
		err = message.Unmarshal(rec.GetValue(), &peerTransactions)
		if err != nil {
			logger.Sugar.Errorf("failed to Unmarshal record value with key: %v, err: %v", key, err)
			return nil, err
		}
		pts := make([]*entity.PeerTransaction, 0)
		for _, peerTransaction := range peerTransactions {
			pts = append(pts, peerTransaction)
		}
		return pts, nil
	}

	return nil, nil
}

func (this *PeerTransactionService) PutLocalPTs(keyKind string, peerTransactions []*entity.PeerTransaction) error {
	var key string
	for _, peerTransaction := range peerTransactions {
		if keyKind == ns.PeerTransaction_Src_KeyKind {
			if len(peerTransaction.SrcPeerId) == 0 {
				return errors.New("NullSrcPeerId")
			}
			key = ns.GetPeerTransactionSrcKey(peerTransaction.SrcPeerId)
		} else if keyKind == ns.PeerTransaction_Target_KeyKind {
			if len(peerTransaction.TargetPeerId) == 0 {
				return errors.New("NullTargetPeerId")
			}
			key = ns.GetPeerTransactionTargetKey(peerTransaction.TargetPeerId)
		} else if keyKind == ns.PeerTransaction_P2PChat_KeyKind {
			if len(peerTransaction.BusinessNumber) == 0 {
				return errors.New("NullBusinessNumber")
			}
			key = ns.GetPeerTransactionP2pChatKey(peerTransaction.BusinessNumber)
		} else if keyKind == ns.PeerTransaction_GroupFile_KeyKind {
			if len(peerTransaction.BusinessNumber) == 0 {
				return errors.New("NullBusinessNumber")
			}
			key = ns.GetPeerTransactionGroupFileKey(peerTransaction.BusinessNumber)
		} else if keyKind == ns.PeerTransaction_Channel_KeyKind {
			/*if len(peerTransaction.BusinessNumber) == 0 {
				return errors.New("NullBusinessNumber")
			}
			key = ns.GetPeerTransactionChannelKey(peerTransaction.BusinessNumber)*/
			key = ns.GetPeerTransactionChannelKey(fmt.Sprintf("%v-%v", entity2.TransactionType_DataBlock, chainentity.BlockType_Channel))
		} else if keyKind == ns.PeerTransaction_ChannelArticle_KeyKind {
			if len(peerTransaction.BusinessNumber) == 0 {
				return errors.New("NullBusinessNumber")
			}
			key = ns.GetPeerTransactionChannelArticleKey(peerTransaction.BusinessNumber)
		} else {
			logger.Sugar.Errorf("InvalidPeerTransactionKeyKind: %v", keyKind)
			return errors.New("InvalidPeerTransactionKeyKind")
		}
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

func (this *PeerTransactionService) PutPTs(peerTransaction *entity.PeerTransaction) error {
	if peerTransaction.TransactionType == fmt.Sprintf("%v-%v", entity2.TransactionType_DataBlock, chainentity.BlockType_Collection) {
		err := this.PutPT(peerTransaction, ns.PeerTransaction_Src_KeyKind)
		if err != nil {
			return err
		}
		return this.PutPT(peerTransaction, ns.PeerTransaction_Target_KeyKind)
	} else if peerTransaction.TransactionType == fmt.Sprintf("%v-%v", entity2.TransactionType_DataBlock, chainentity.BlockType_P2pChat) {
		return this.PutPT(peerTransaction, ns.PeerTransaction_P2PChat_KeyKind)
	} else if peerTransaction.TransactionType == fmt.Sprintf("%v-%v", entity2.TransactionType_DataBlock, chainentity.BlockType_GroupFile) {
		return this.PutPT(peerTransaction, ns.PeerTransaction_GroupFile_KeyKind)
	} else if peerTransaction.TransactionType == fmt.Sprintf("%v-%v", entity2.TransactionType_DataBlock, chainentity.BlockType_Channel) {
		return this.PutPT(peerTransaction, ns.PeerTransaction_Channel_KeyKind)
	} else if peerTransaction.TransactionType == fmt.Sprintf("%v-%v", entity2.TransactionType_DataBlock, chainentity.BlockType_ChannelArticle) {
		return this.PutPT(peerTransaction, ns.PeerTransaction_ChannelArticle_KeyKind)
	} else {
		logger.Sugar.Errorf("InvalidTransactionType: %v", peerTransaction.TransactionType)
		return errors.New("InvalidTransactionType")
	}
}

func (this *PeerTransactionService) PutPT(peerTransaction *entity.PeerTransaction, keyKind string) error {
	bytePeerTransaction, err := message.Marshal(peerTransaction)
	if err != nil {
		return err
	}
	var key string
	if keyKind == ns.PeerTransaction_Src_KeyKind {
		if len(peerTransaction.SrcPeerId) == 0 {
			return errors.New("NullSrcPeerId")
		}
		key = ns.GetPeerTransactionSrcKey(peerTransaction.SrcPeerId)
	} else if keyKind == ns.PeerTransaction_Target_KeyKind {
		if len(peerTransaction.TargetPeerId) == 0 {
			return errors.New("NullTargetPeerId")
		}
		key = ns.GetPeerTransactionTargetKey(peerTransaction.TargetPeerId)
	} else if keyKind == ns.PeerTransaction_P2PChat_KeyKind {
		if len(peerTransaction.BusinessNumber) == 0 {
			return errors.New("NullBusinessNumber")
		}
		key = ns.GetPeerTransactionP2pChatKey(peerTransaction.BusinessNumber)
	} else if keyKind == ns.PeerTransaction_GroupFile_KeyKind {
		if len(peerTransaction.BusinessNumber) == 0 {
			return errors.New("NullBusinessNumber")
		}
		key = ns.GetPeerTransactionGroupFileKey(peerTransaction.BusinessNumber)
	} else if keyKind == ns.PeerTransaction_Channel_KeyKind {
		/*if len(peerTransaction.BusinessNumber) == 0 {
			return errors.New("NullBusinessNumber")
		}
		key = ns.GetPeerTransactionChannelKey(peerTransaction.BusinessNumber)*/
		key = ns.GetPeerTransactionChannelKey(fmt.Sprintf("%v-%v", entity2.TransactionType_DataBlock, chainentity.BlockType_Channel))
	} else if keyKind == ns.PeerTransaction_ChannelArticle_KeyKind {
		if len(peerTransaction.BusinessNumber) == 0 {
			return errors.New("NullBusinessNumber")
		}
		key = ns.GetPeerTransactionChannelArticleKey(peerTransaction.BusinessNumber)
	} else {
		logger.Sugar.Errorf("InvalidPeerTransactionKeyKind: %v", keyKind)
		return errors.New("InvalidPeerTransactionKeyKind")
	}

	return dht.PeerEndpointDHT.PutValue(key, bytePeerTransaction)
}

func init() {
	service.GetSession().Sync(new(entity.PeerTransaction))

	peerTransactionService.OrmBaseService.GetSeqName = peerTransactionService.GetSeqName
	peerTransactionService.OrmBaseService.FactNewEntity = peerTransactionService.NewEntity
	peerTransactionService.OrmBaseService.FactNewEntities = peerTransactionService.NewEntities
	service.RegistSeq(seqname, 0)
	container.RegistService(ns.PeerTransaction_Src_Prefix, peerTransactionService)
	container.RegistService(ns.PeerTransaction_Target_Prefix, peerTransactionService)
	container.RegistService(ns.PeerTransaction_P2PChat_Prefix, peerTransactionService)
	container.RegistService(ns.PeerTransaction_GroupFile_Prefix, peerTransactionService)
	container.RegistService(ns.PeerTransaction_Channel_Prefix, peerTransactionService)
	container.RegistService(ns.PeerTransaction_ChannelArticle_Prefix, peerTransactionService)
}
