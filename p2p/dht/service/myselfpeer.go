package service

import (
	"github.com/curltech/go-colla-core/cache"
	"github.com/curltech/go-colla-core/service"
	"github.com/curltech/go-colla-core/util/message"
	//"github.com/curltech/go-colla-core/p2p"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
)

/**
同步表结构，服务继承基本服务的方法
*/
type MyselfPeerService struct {
	PeerEntityService
}

var myselfPeerService = &MyselfPeerService{}

func GetMyselfPeerService() *MyselfPeerService {
	return myselfPeerService
}

func (this *MyselfPeerService) GetSeqName() string {
	return seqname
}

var MemCache = cache.NewMemCache("dht", 0, 0)

func (this *MyselfPeerService) NewEntity(data []byte) (interface{}, error) {
	entity := &entity.MyselfPeer{}
	if data == nil {
		return entity, nil
	}
	err := message.Unmarshal(data, entity)
	if err != nil {
		return nil, err
	}

	return entity, err
}

func (this *MyselfPeerService) NewEntities(data []byte) (interface{}, error) {
	entities := make([]*entity.MyselfPeer, 0)
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
	service.GetSession().Sync(new(entity.MyselfPeer))

	myselfPeerService.OrmBaseService.GetSeqName = myselfPeerService.GetSeqName
	myselfPeerService.OrmBaseService.FactNewEntity = myselfPeerService.NewEntity
	myselfPeerService.OrmBaseService.FactNewEntities = myselfPeerService.NewEntities
}
