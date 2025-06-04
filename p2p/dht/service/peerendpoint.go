package service

import (
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/container"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/service"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/libp2p/go-libp2p/core/peer"
	"math/rand"
	"sync"
)

/*
*
同步表结构，服务继承基本服务的方法
*/
type PeerEndpointService struct {
	PeerEntityService
	Mutex sync.Mutex
}

var peerEndpointService = &PeerEndpointService{Mutex: sync.Mutex{}}

func GetPeerEndpointService() *PeerEndpointService {
	return peerEndpointService
}

func (svc *PeerEndpointService) GetSeqName() string {
	return seqname
}

func (svc *PeerEndpointService) NewEntity(data []byte) (interface{}, error) {
	peerEndpoint := &entity.PeerEndpoint{}
	if data == nil {
		return peerEndpoint, nil
	}
	err := message.Unmarshal(data, peerEndpoint)
	if err != nil {
		return nil, err
	}

	return peerEndpoint, err
}

func (svc *PeerEndpointService) NewEntities(data []byte) (interface{}, error) {
	entities := make([]*entity.PeerEndpoint, 0)
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
	_ = service.GetSession().Sync(new(entity.PeerEndpoint))
	peerEndpointService.OrmBaseService.GetSeqName = peerEndpointService.GetSeqName
	peerEndpointService.OrmBaseService.FactNewEntity = peerEndpointService.NewEntity
	peerEndpointService.OrmBaseService.FactNewEntities = peerEndpointService.NewEntities
	container.RegistService(ns.PeerEndpoint_Prefix, peerEndpointService)
}

func (svc *PeerEndpointService) getCacheKey(key string) string {
	return "PeerEndpoint:" + key
}

func (svc *PeerEndpointService) GetFromCache(peerId string) *entity.PeerEndpoint {
	key := svc.getCacheKey(peerId)
	ptr, found := MemCache.Get(key)
	if found {
		return ptr.(*entity.PeerEndpoint)
	}
	svc.Mutex.Lock()
	defer svc.Mutex.Unlock()
	ptr, found = MemCache.Get(key)
	if !found {
		peerEndpoint := entity.PeerEndpoint{}
		peerEndpoint.PeerId = peerId
		found, _ = svc.Get(&peerEndpoint, false, "", "")
		if found {
			ptr = &peerEndpoint
		} else {
			ptr = &entity.PeerEndpoint{}
		}

		MemCache.SetDefault(key, ptr)
	}

	return ptr.(*entity.PeerEndpoint)
}

/*
*
根据peerId分布式查找PeerEndpoint
*/
func (svc *PeerEndpointService) FindPeer(peerId string) (string, error) {
	id, err := peer.Decode(peerId)
	if err != nil {
		logger.Sugar.Errorf("not effective peer endpoint peerId: %v, error: %v", peerId, err.Error())
		return "", err
	}
	addrInfo, err := dht.PeerEndpointDHT.FindPeer(id)
	if err != nil {
		logger.Sugar.Errorf("find peer endpoint peerId: %v, error: %v", peerId, err.Error())
		return "", err
	}
	return addrInfo.String(), nil
}

func (svc *PeerEndpointService) GetLocal(peerId string) ([]*entity.PeerEndpoint, error) {
	key := ns.GetPeerEndpointKey(peerId)
	rec, err := dht.PeerEndpointDHT.GetLocal(key)
	if err != nil {
		logger.Sugar.Errorf("failed to GetLocal by key: %v, err: %v", key, err)
		return nil, err
	}
	if rec != nil {
		peerEndpoints := make([]*entity.PeerEndpoint, 0)
		err = message.Unmarshal(rec.GetValue(), &peerEndpoints)
		if err != nil {
			logger.Sugar.Errorf("failed to Unmarshal record value with key: %v, err: %v", key, err)
			return nil, err
		}
		return peerEndpoints, nil
	}

	return nil, nil
}

func (svc *PeerEndpointService) PutLocal(peerEndpoint *entity.PeerEndpoint) error {
	key := ns.GetPeerEndpointKey(peerEndpoint.PeerId)
	bytePeerEndpoint, err := message.Marshal(peerEndpoint)
	if err != nil {
		return err
	}
	return dht.PeerEndpointDHT.PutLocal(key, bytePeerEndpoint)
}

func (svc *PeerEndpointService) GetValue(peerId string) (*entity.PeerEndpoint, error) {
	key := ns.GetPeerClientKey(peerId)
	buf, err := dht.PeerEndpointDHT.GetValue(key)
	if err != nil {
		return nil, err
	}
	peerEndpoint := &entity.PeerEndpoint{}
	err = message.Unmarshal(buf, peerEndpoint)
	if err != nil {
		return nil, err
	}
	return peerEndpoint, nil
}

func (svc *PeerEndpointService) PutValue(peerEndpoint *entity.PeerEndpoint) error {
	key := ns.GetPeerEndpointKey(peerEndpoint.PeerId)
	value, err := message.Marshal(peerEndpoint)
	if err != nil {
		return err
	}
	err = dht.PeerEndpointDHT.PutValue(key, value)

	return err
}

func (svc *PeerEndpointService) GetRand(seed int64) []*entity.PeerEndpoint {
	limit := config.ConsensusParams.PeerNum
	peerEndpoints := make([]*entity.PeerEndpoint, 0)
	peerEndpoint := &entity.PeerEndpoint{}
	//peerEndpoint.Status = entity2.EntityStatus_Effective
	peerEndpoint.ActiveStatus = entity.ActiveStatus_Up
	count, _ := svc.Count(peerEndpoint, "")
	from := 0
	if int(count) > limit {
		rand.Seed(seed)
		from = rand.Intn(int(count) - limit)
	}
	err := svc.Find(&peerEndpoints, peerEndpoint, "", from, limit, "")
	if err == nil {
		return peerEndpoints
	}

	return nil
}
