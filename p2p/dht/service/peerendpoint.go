package service

import (
	"github.com/curltech/go-colla-core/container"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/service"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/libp2p/go-libp2p-core/peer"
	"sync"
	"time"
	"math/rand"
)

/**
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

func (this *PeerEndpointService) GetSeqName() string {
	return seqname
}

func (this *PeerEndpointService) NewEntity(data []byte) (interface{}, error) {
	entity := &entity.PeerEndpoint{}
	if data == nil {
		return entity, nil
	}
	err := message.Unmarshal(data, entity)
	if err != nil {
		return nil, err
	}

	return entity, err
}

func (this *PeerEndpointService) NewEntities(data []byte) (interface{}, error) {
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
	service.GetSession().Sync(new(entity.PeerEndpoint))
	peerEndpointService.OrmBaseService.GetSeqName = peerEndpointService.GetSeqName
	peerEndpointService.OrmBaseService.FactNewEntity = peerEndpointService.NewEntity
	peerEndpointService.OrmBaseService.FactNewEntities = peerEndpointService.NewEntities
	container.RegistService(ns.PeerEndpoint_Prefix, peerEndpointService)
}

func (this *PeerEndpointService) getCacheKey(key string) string {
	return "PeerEndpoint:" + key
}

func (this *PeerEndpointService) GetFromCache(peerId string) *entity.PeerEndpoint {
	key := this.getCacheKey(peerId)
	ptr, found := MemCache.Get(key)
	if found {
		return ptr.(*entity.PeerEndpoint)
	}
	this.Mutex.Lock()
	defer this.Mutex.Unlock()
	ptr, found = MemCache.Get(key)
	if !found {
		peerEndpoint := entity.PeerEndpoint{}
		peerEndpoint.PeerId = peerId
		found = this.Get(&peerEndpoint, false, "", "")
		if found {
			ptr = &peerEndpoint
		} else {
			ptr = &entity.PeerEndpoint{}
		}

		MemCache.SetDefault(key, ptr)
	}

	return ptr.(*entity.PeerEndpoint)
}

func (this *PeerEndpointService) FindPeer(peerId string) (string, error) {
	id, err := peer.Decode(peerId)
	if err != nil {
		return "", err
	}
	addrInfo, err := dht.PeerEndpointDHT.FindPeer(id)
	if err != nil {
		return "", err
	}
	return addrInfo.String(), nil
}

func (this *PeerEndpointService) GetLocal(peerId string) ([]*entity.PeerEndpoint, error) {
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

func (this *PeerEndpointService) PutLocal(peerEndpoint *entity.PeerEndpoint) error {
	key := ns.GetPeerEndpointKey(peerEndpoint.PeerId)
	bytePeerEndpoint, err := message.Marshal(peerEndpoint)
	if err != nil {
		return err
	}
	return dht.PeerEndpointDHT.PutLocal(key, bytePeerEndpoint)
}

func (this *PeerEndpointService) GetValue(peerId string) (*entity.PeerEndpoint, error) {
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

func (this *PeerEndpointService) PutValue(peerEndpoint *entity.PeerEndpoint) error {
	key := ns.GetPeerEndpointKey(peerEndpoint.PeerId)
	value, err := message.Marshal(peerEndpoint)
	if err != nil {
		return err
	}
	err = dht.PeerEndpointDHT.PutValue(key, value)

	return err
}

type PeerEndpointPeerId struct {
	PeerId    string
}

//origin为原数组，count为随机取出的个数，最终返回一个count容量的目标数组
func randomSlice(origin []*PeerEndpointPeerId, count int) []*PeerEndpointPeerId {
	tmpOrigin := make([]*PeerEndpointPeerId, len(origin))
	copy(tmpOrigin, origin)
	rand.Seed(time.Now().Unix())
	rand.Shuffle(len(tmpOrigin), func(i int, j int) {
		tmpOrigin[i], tmpOrigin[j] = tmpOrigin[j], tmpOrigin[i]
	})

	result := make([]*PeerEndpointPeerId, 0, count)
	for index, value := range tmpOrigin {
		if index == count{
			break
		}
		result = append(result, value)
	}
	return result
}

func (this *PeerEndpointService) GetRand(count int) []*entity.PeerEndpoint {
	result := make([]*entity.PeerEndpoint, 0)
	activePeerIds := make([]*PeerEndpointPeerId, 0)
	conditionBean := &entity.PeerEndpoint{}
	//conditionBean.Status = entity2.EntityStatus_Effective
	conditionBean.ActiveStatus = entity.ActiveStatus_Up
	this.Find(&activePeerIds, conditionBean, "", 0, 0, "")
	if len(activePeerIds) > 0 {
		randomPeerIds := randomSlice(activePeerIds, count)
		for _, randomPeerId := range randomPeerIds {
			peerId := randomPeerId.PeerId
			peerEndpoints, err := this.GetLocal(peerId)
			if err != nil {
				logger.Sugar.Errorf("failed to GetLocal PeerEndPoint: %v, err: %v", peerId, err)
			} else {
				if peerEndpoints != nil && len(peerEndpoints) > 0 {
					for _, peerEndpoint := range peerEndpoints {
						result = append(result, peerEndpoint)
					}
				}
			}
		}
		return result
	}

	return nil
}
