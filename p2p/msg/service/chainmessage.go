package service

import (
	coreservice "github.com/curltech/go-colla-core/service"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	entity3 "github.com/curltech/go-colla-node/p2p/msg/entity"
	"sync"
)

/**
同步表结构，服务继承基本服务的方法
*/
type ChainMessageService struct {
	service.PeerEntityService
	Mutex sync.Mutex
}

var chainMessageService = &ChainMessageService{Mutex: sync.Mutex{}}

func GetChainMessageService() *ChainMessageService {
	return chainMessageService
}

var seqname = "seq_block"

func (this *ChainMessageService) GetSeqName() string {
	return seqname
}

func (this *ChainMessageService) NewEntity(data []byte) (interface{}, error) {
	entity := &entity3.ChainMessage{}
	if data == nil {
		return entity, nil
	}
	err := message.Unmarshal(data, entity)
	if err != nil {
		return nil, err
	}

	return entity, err
}

func (this *ChainMessageService) NewEntities(data []byte) (interface{}, error) {
	entities := make([]*entity3.ChainMessage, 0)
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
	coreservice.GetSession().Sync(new(entity3.ChainMessage))

	chainMessageService.OrmBaseService.GetSeqName = chainMessageService.GetSeqName
	chainMessageService.OrmBaseService.FactNewEntity = chainMessageService.NewEntity
	chainMessageService.OrmBaseService.FactNewEntities = chainMessageService.NewEntities
}
