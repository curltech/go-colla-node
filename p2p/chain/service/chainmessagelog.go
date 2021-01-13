package service

import (
	"github.com/curltech/go-colla-core/container"
	"github.com/curltech/go-colla-core/service"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/p2p/chain/entity"
)

/**
同步表结构，服务继承基本服务的方法
*/
type ChainMessageLogService struct {
	service.OrmBaseService
}

var chainMessageLogService = &ChainMessageLogService{}

func GetChainMessageLogService() *ChainMessageLogService {
	return chainMessageLogService
}

func (this *ChainMessageLogService) GetSeqName() string {
	return seqname
}

func (this *ChainMessageLogService) NewEntity(data []byte) (interface{}, error) {
	entity := &entity.ChainMessageLog{}
	if data == nil {
		return entity, nil
	}
	err := message.Unmarshal(data, entity)
	if err != nil {
		return nil, err
	}

	return entity, err
}

func (this *ChainMessageLogService) NewEntities(data []byte) (interface{}, error) {
	entities := make([]*entity.ChainMessageLog, 0)
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
	service.GetSession().Sync(new(entity.ChainMessageLog))

	chainMessageLogService.OrmBaseService.GetSeqName = chainMessageLogService.GetSeqName
	chainMessageLogService.OrmBaseService.FactNewEntity = chainMessageLogService.NewEntity
	chainMessageLogService.OrmBaseService.FactNewEntities = chainMessageLogService.NewEntities
	service.RegistSeq(seqname, 0)
	container.RegistService("chainMessageLog", chainMessageLogService)
}
