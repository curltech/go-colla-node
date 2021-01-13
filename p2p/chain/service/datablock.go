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

func init() {
	service.GetSession().Sync(new(entity.DataBlock))

	dataBlockService.OrmBaseService.GetSeqName = dataBlockService.GetSeqName
	dataBlockService.OrmBaseService.FactNewEntity = dataBlockService.NewEntity
	dataBlockService.OrmBaseService.FactNewEntities = dataBlockService.NewEntities
	service.RegistSeq(seqname, 0)
	container.RegistService(ns.DataBlock_Prefix, dataBlockService)
	container.RegistService(ns.DataBlock_Owner_Prefix, dataBlockService)
}
