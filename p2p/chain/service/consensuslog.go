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
type ConsensusLogService struct {
	service.OrmBaseService
}

var consensusLogService = &ConsensusLogService{}

func GetConsensusLogService() *ConsensusLogService {
	return consensusLogService
}

func (this *ConsensusLogService) GetSeqName() string {
	return seqname
}

func (this *ConsensusLogService) NewEntity(data []byte) (interface{}, error) {
	entity := &entity.ConsensusLog{}
	if data == nil {
		return entity, nil
	}
	err := message.Unmarshal(data, entity)
	if err != nil {
		return nil, err
	}

	return entity, err
}

func (this *ConsensusLogService) NewEntities(data []byte) (interface{}, error) {
	entities := make([]*entity.ConsensusLog, 0)
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
	service.GetSession().Sync(new(entity.ConsensusLog))

	consensusLogService.OrmBaseService.GetSeqName = consensusLogService.GetSeqName
	consensusLogService.OrmBaseService.FactNewEntity = consensusLogService.NewEntity
	consensusLogService.OrmBaseService.FactNewEntities = consensusLogService.NewEntities
	service.RegistSeq(seqname, 0)
	container.RegistService("consensusLog", consensusLogService)
}
