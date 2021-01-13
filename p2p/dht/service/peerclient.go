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
type PeerClientService struct {
	PeerEntityService
	Mutex sync.Mutex
}

var peerClientService = &PeerClientService{Mutex: sync.Mutex{}}

func GetPeerClientService() *PeerClientService {
	return peerClientService
}

func (this *PeerClientService) GetSeqName() string {
	return seqname
}

func (this *PeerClientService) NewEntity(data []byte) (interface{}, error) {
	entity := &entity.PeerClient{}
	if data == nil {
		return entity, nil
	}
	err := message.Unmarshal(data, entity)
	if err != nil {
		return nil, err
	}

	return entity, err
}

func (this *PeerClientService) NewEntities(data []byte) (interface{}, error) {
	entities := make([]*entity.PeerClient, 0)
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
	service.GetSession().Sync(new(entity.PeerClient))

	peerClientService.OrmBaseService.GetSeqName = peerClientService.GetSeqName
	peerClientService.OrmBaseService.FactNewEntity = peerClientService.NewEntity
	peerClientService.OrmBaseService.FactNewEntities = peerClientService.NewEntities
	container.RegistService(ns.PeerClient_Prefix, peerClientService)
	container.RegistService(ns.PeerClient_Mobile_Prefix, peerClientService)
}

func (this *PeerClientService) getCacheKey(key string) string {
	return "PeerClient:" + key
}

func (this *PeerClientService) GetFromCache(peerId string) *entity.PeerClient {
	key := this.getCacheKey(peerId)
	ptr, found := MemCache.Get(key)
	if found {
		return ptr.(*entity.PeerClient)
	}
	this.Mutex.Lock()
	defer this.Mutex.Unlock()
	ptr, found = MemCache.Get(key)
	if !found {
		peerClient := entity.PeerClient{}
		peerClient.PeerId = peerId
		found = this.Get(&peerClient, false, "", "")
		if found {
			ptr = &peerClient
		} else {
			ptr = &entity.PeerClient{}
		}

		MemCache.SetDefault(key, ptr)
	}

	return ptr.(*entity.PeerClient)
}

func (this *PeerClientService) GetValue(peerId string) ([]*entity.PeerClient, error) {
	key := ns.GetPeerClientKey(peerId)
	buf, err := dht.PeerEndpointDHT.GetValue(key)
	if err != nil {
		return nil, err
	}
	peerClients := make([]*entity.PeerClient, 0)
	err = message.Unmarshal(buf, &peerClients)
	if err != nil {
		return nil, err
	}
	return peerClients, nil
}

func (this *PeerClientService) PutValue(peerClient *entity.PeerClient) error {
	key := ns.GetPeerClientKey(peerClient.PeerId)
	value, err := message.Marshal(peerClient)
	if err != nil {
		return err
	}
	err = dht.PeerEndpointDHT.PutValue(key, value)

	return err
}
