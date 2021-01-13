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
type PbftConsensusLogService struct {
	service.OrmBaseService
}

var pbftConsensusLogService = &PbftConsensusLogService{}

func GetPbftConsensusLogService() *PbftConsensusLogService {
	return pbftConsensusLogService
}

func (this *PbftConsensusLogService) GetSeqName() string {
	return seqname
}

func (this *PbftConsensusLogService) NewEntity(data []byte) (interface{}, error) {
	entity := &entity.PbftConsensusLog{}
	if data == nil {
		return entity, nil
	}
	err := message.Unmarshal(data, entity)
	if err != nil {
		return nil, err
	}

	return entity, err
}

func (this *PbftConsensusLogService) NewEntities(data []byte) (interface{}, error) {
	entities := make([]*entity.PbftConsensusLog, 0)
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
	service.GetSession().Sync(new(entity.PbftConsensusLog))

	pbftConsensusLogService.OrmBaseService.GetSeqName = pbftConsensusLogService.GetSeqName
	pbftConsensusLogService.OrmBaseService.FactNewEntity = pbftConsensusLogService.NewEntity
	pbftConsensusLogService.OrmBaseService.FactNewEntities = pbftConsensusLogService.NewEntities
	service.RegistSeq(seqname, 0)
	container.RegistService("pbftConsensusLog", pbftConsensusLogService)
}
