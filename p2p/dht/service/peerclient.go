package service

import (
	"errors"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/container"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/service"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"sync"
)

/*
*
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

func (svc *PeerClientService) GetSeqName() string {
	return seqname
}

func (svc *PeerClientService) NewEntity(data []byte) (interface{}, error) {
	peerClient := &entity.PeerClient{}
	if data == nil {
		return peerClient, nil
	}
	err := message.Unmarshal(data, peerClient)
	if err != nil {
		return nil, err
	}

	return peerClient, err
}

func (svc *PeerClientService) NewEntities(data []byte) (interface{}, error) {
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
	_ = service.GetSession().Sync(new(entity.PeerClient))

	peerClientService.OrmBaseService.GetSeqName = peerClientService.GetSeqName
	peerClientService.OrmBaseService.FactNewEntity = peerClientService.NewEntity
	peerClientService.OrmBaseService.FactNewEntities = peerClientService.NewEntities
	container.RegistService(ns.PeerClient_Prefix, peerClientService)
	container.RegistService(ns.PeerClient_Mobile_Prefix, peerClientService)
	container.RegistService(ns.PeerClient_Email_Prefix, peerClientService)
	container.RegistService(ns.PeerClient_Name_Prefix, peerClientService)
	//把所有客户端的活动状态更新成未连接
	peerClient := new(entity.PeerClient)
	peerClient.ActiveStatus = entity.ActiveStatus_Down
	_, _ = peerClientService.Update(peerClient, nil, "")
}

func (svc *PeerClientService) getCacheKey(key string) string {
	return "PeerClient:" + key
}

func (svc *PeerClientService) GetFromCache(peerId string) *entity.PeerClient {
	key := svc.getCacheKey(peerId)
	ptr, found := MemCache.Get(key)
	if found {
		return ptr.(*entity.PeerClient)
	}
	svc.Mutex.Lock()
	defer svc.Mutex.Unlock()
	ptr, found = MemCache.Get(key)
	if !found {
		peerClient := entity.PeerClient{}
		peerClient.PeerId = peerId
		found, _ = svc.Get(&peerClient, false, "", "")
		if found {
			ptr = &peerClient
		} else {
			ptr = &entity.PeerClient{}
		}

		MemCache.SetDefault(key, ptr)
	}

	return ptr.(*entity.PeerClient)
}

func (svc *PeerClientService) Validate(peerClient *entity.PeerClient) error {
	//expireDate := peerClient.ExpireDate
	//if expireDate == 0 {
	//	return errors.New("Invalid expireDate")
	//}
	return nil
}

// GetLocals 根据peerclient的peerid和clientid查找匹配的本地peerclient
func (svc *PeerClientService) GetLocals(key string, clientId string) ([]*entity.PeerClient, error) {
	rec, err := dht.PeerEndpointDHT.GetLocal(key)
	if err != nil {
		logger.Sugar.Errorf("failed to GetLocal by key: %v, err: %v", key, err)
		return nil, err
	}
	if rec != nil {
		peerClients := make([]*entity.PeerClient, 0)
		err = message.Unmarshal(rec.GetValue(), &peerClients)
		if err != nil {
			logger.Sugar.Errorf("failed to Unmarshal record value with key: %v, err: %v", key, err)
			return nil, err
		}
		if len(clientId) > 0 {
			pcs := make([]*entity.PeerClient, 0)
			for _, peerClient := range peerClients {
				if peerClient.ClientId == clientId {
					pcs = append(pcs, peerClient)
				}
			}
			if len(pcs) == 0 {
				logger.Sugar.Errorf("failed to find key: %v, clientId: %v peer client", key, clientId)
			}
			return pcs, nil
		} else {
			return peerClients, nil
		}
	}

	return nil, nil
}

func (svc *PeerClientService) PutLocals(peerClients []*entity.PeerClient) error {
	for _, peerClient := range peerClients {
		key := ns.GetPeerClientKey(peerClient.PeerId)
		bytePeerClient, err := message.Marshal(peerClient)
		if err != nil {
			return err
		}
		err = dht.PeerEndpointDHT.PutLocal(key, bytePeerClient)
		if err != nil {
			return err
		}
	}

	return nil
}

/*
*
根据peerId，mobile，email，name分布式查询PeerClient
*/
func (svc *PeerClientService) GetValues(peerId string, mobile string, email string, name string) ([]*entity.PeerClient, error) {
	if len(peerId) == 0 && len(mobile) == 0 && len(name) == 0 {
		logger.Sugar.Errorf("InvalidPeerClientKey")
		return nil, errors.New("InvalidPeerClientKey")
	}
	peerClients := make([]*entity.PeerClient, 0)
	var key string
	if len(peerId) > 0 {
		key = ns.GetPeerClientKey(peerId)
		pcs, err := svc.GetKeyValues(key)
		if err != nil {
			return nil, err
		}
		for _, pc := range pcs {
			peerClients = append(peerClients, pc)
		}
	}
	if len(mobile) > 0 {
		key = ns.GetPeerClientMobileKey(mobile, true)
		pcs, err := svc.GetKeyValues(key)
		if err != nil {
			return nil, err
		}
		for _, pc := range pcs {
			peerClients = append(peerClients, pc)
		}
	}
	if len(email) > 0 {
		key = ns.GetPeerClientEmailKey(email, true)
		pcs, err := svc.GetKeyValues(key)
		if err != nil {
			return nil, err
		}
		for _, pc := range pcs {
			peerClients = append(peerClients, pc)
		}
	}
	if len(name) > 0 {
		key = ns.GetPeerClientNameKey(name, true)
		pcs, err := svc.GetKeyValues(key)
		if err != nil {
			return nil, err
		}
		for _, pc := range pcs {
			peerClients = append(peerClients, pc)
		}
	}
	return peerClients, nil
}

func (svc *PeerClientService) GetKeyValues(key string) ([]*entity.PeerClient, error) {
	peerClients := make([]*entity.PeerClient, 0)
	if config.Libp2pParams.FaultTolerantLevel == 0 {
		recvdVals, err := dht.PeerEndpointDHT.GetValues(key)
		if err != nil {
			return nil, err
		}
		for _, recvdVal := range recvdVals {
			pcs := make([]*entity.PeerClient, 0)
			err = message.TextUnmarshal(string(recvdVal), &pcs)
			if err != nil {
				logger.Sugar.Errorf("failed to TextUnmarshal PeerClient value: %v, err: %v", recvdVal, err)
				return nil, err
			}
			for _, pc := range pcs {
				peerClients = append(peerClients, pc)
			}
		}
	} else if config.Libp2pParams.FaultTolerantLevel == 1 {
		buf, err := dht.PeerEndpointDHT.GetValue(key)
		if err != nil {
			return nil, err
		}
		err = message.Unmarshal(buf, &peerClients)
		if err != nil {
			return nil, err
		}
	} else if config.Libp2pParams.FaultTolerantLevel == 2 {
		// 查询删除local记录
		locals, err := svc.GetLocals(key, "")
		if err != nil {
			return nil, err
		}
		if len(locals) > 0 {
			for _, local := range locals {
				peerClients = append(peerClients, local)
			}
			_, _ = svc.Delete(locals, "")
		}
		// 查询non-local记录
		recvdVals, err := dht.PeerEndpointDHT.GetValues(key)
		if err != nil {
			return nil, err
		}
		// 恢复local记录
		err = svc.PutLocals(locals)
		if err != nil {
			return nil, err
		}
		// 整合记录
		for _, recvdVal := range recvdVals {
			pcs := make([]*entity.PeerClient, 0)
			err = message.TextUnmarshal(string(recvdVal), &pcs)
			if err != nil {
				logger.Sugar.Errorf("failed to TextUnmarshal PeerClient value: %v, err: %v", recvdVal, err)
				return nil, err
			}
			for _, pc := range pcs {
				peerClients = append(peerClients, pc)
			}
		}
	}

	return peerClients, nil
}

func (svc *PeerClientService) PutValues(peerClient *entity.PeerClient) error {
	err := svc.PutValue(peerClient, ns.PeerClient_KeyKind)
	if err != nil {
		return err
	}
	//err = svc.PutValue(peerClient, ns.PeerClient_Mobile_KeyKind)
	//if err != nil {
	//	return err
	//}
	//err = svc.PutValue(peerClient, ns.PeerClient_Email_KeyKind)
	//if err != nil {
	//	return err
	//}
	//err = svc.PutValue(peerClient, ns.PeerClient_Name_KeyKind)
	//if err != nil {
	//	return err
	//}
	return nil
}

func (svc *PeerClientService) PutValue(peerClient *entity.PeerClient, keyKind string) error {
	bytePeerClient, err := message.Marshal(peerClient)
	if err != nil {
		return err
	}
	var key string
	if keyKind == ns.PeerClient_KeyKind {
		key = ns.GetPeerClientKey(peerClient.PeerId)
	} else if keyKind == ns.PeerClient_Mobile_KeyKind {
		key = ns.GetPeerClientMobileKey(peerClient.Mobile, true)
	} else if keyKind == ns.PeerClient_Email_KeyKind {
		key = ns.GetPeerClientEmailKey(peerClient.Email, true)
	} else if keyKind == ns.PeerClient_Name_KeyKind {
		key = ns.GetPeerClientNameKey(peerClient.Name, true)
	} else {
		logger.Sugar.Errorf("InvalidPeerClientKeyKind: %v", keyKind)
		return errors.New("InvalidPeerClientKeyKind")
	}

	return dht.PeerEndpointDHT.PutValue(key, bytePeerClient)
}
