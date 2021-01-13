package service

import (
	"github.com/curltech/go-colla-core/container"
	"github.com/curltech/go-colla-core/service"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/chain/entity"
)

/**
同步表结构，服务继承基本服务的方法
*/
type TransactionKeyService struct {
	service.OrmBaseService
}

var transactionKeyService = &TransactionKeyService{}

func GetTransactionKeyService() *TransactionKeyService {
	return transactionKeyService
}

func (this *TransactionKeyService) GetSeqName() string {
	return seqname
}

func (this *TransactionKeyService) NewEntity(data []byte) (interface{}, error) {
	entity := &entity.TransactionKey{}
	if data == nil {
		return entity, nil
	}
	err := message.Unmarshal(data, entity)
	if err != nil {
		return nil, err
	}

	return entity, err
}

func (this *TransactionKeyService) NewEntities(data []byte) (interface{}, error) {
	entities := make([]*entity.TransactionKey, 0)
	if data == nil {
		return &entities, nil
	}
	err := message.Unmarshal(data, &entities)
	if err != nil {
		return nil, err
	}

	return &entities, err
}

func init() {
	service.GetSession().Sync(new(entity.TransactionKey))

	transactionKeyService.OrmBaseService.GetSeqName = transactionKeyService.GetSeqName
	transactionKeyService.OrmBaseService.FactNewEntity = transactionKeyService.NewEntity
	transactionKeyService.OrmBaseService.FactNewEntities = transactionKeyService.NewEntities
	service.RegistSeq(seqname, 0)
	container.RegistService(ns.TransactionKey_Prefix, transactionKeyService)
}
