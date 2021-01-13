package service

import (
	"github.com/curltech/go-colla-core/container"
	"github.com/curltech/go-colla-core/service"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"sync"
)

/**
同步表结构，服务继承基本服务的方法
*/
type ChainAppService struct {
	PeerEntityService
	Mutex sync.Mutex
}

var chainAppService = &ChainAppService{Mutex: sync.Mutex{}}

func GetChainAppService() *ChainAppService {
	return chainAppService
}

func (this *ChainAppService) GetSeqName() string {
	return seqname
}

func (this *ChainAppService) NewEntity(data []byte) (interface{}, error) {
	entity := &entity.ChainApp{}
	if data == nil {
		return entity, nil
	}
	err := message.Unmarshal(data, entity)
	if err != nil {
		return nil, err
	}

	return entity, err
}

func (this *ChainAppService) NewEntities(data []byte) (interface{}, error) {
	entities := make([]*entity.ChainApp, 0)
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
	service.GetSession().Sync(new(entity.ChainApp))

	chainAppService.OrmBaseService.GetSeqName = chainAppService.GetSeqName
	chainAppService.OrmBaseService.FactNewEntity = chainAppService.NewEntity
	chainAppService.OrmBaseService.FactNewEntities = chainAppService.NewEntities
	container.RegistService(ns.ChainApp_Prefix, chainAppService)
}

func (this *ChainAppService) getCacheKey(key string) string {
	return "ChainApp:" + key
}

func (this *ChainAppService) GetFromCache(peerId string) *entity.ChainApp {
	key := this.getCacheKey(peerId)
	ptr, found := MemCache.Get(key)
	if found {
		return ptr.(*entity.ChainApp)
	}
	this.Mutex.Lock()
	defer this.Mutex.Unlock()
	ptr, found = MemCache.Get(key)
	if !found {
		chainApp := entity.ChainApp{}
		chainApp.PeerId = peerId
		found = this.Get(&chainApp, false, "", "")
		if found {
			ptr = &chainApp
		} else {
			ptr = &entity.ChainApp{}
		}

		MemCache.SetDefault(key, ptr)
	}

	return ptr.(*entity.ChainApp)
}

func (this *ChainAppService) GetValue(peerId string) (*entity.ChainApp, error) {
	key := ns.GetChainAppKey(peerId)
	buf, err := dht.PeerEndpointDHT.GetValue(key)
	if err != nil {
		return nil, err
	}
	chainApp := &entity.ChainApp{}
	err = message.Unmarshal(buf, chainApp)
	if err != nil {
		return nil, err
	}
	return chainApp, nil
}

func (this *ChainAppService) PutValue(chainApp *entity.ChainApp) error {
	key := ns.GetChainAppKey(chainApp.PeerId)
	value, err := message.Marshal(chainApp)
	if err != nil {
		return err
	}
	err = dht.PeerEndpointDHT.PutValue(key, value)

	return err
}
