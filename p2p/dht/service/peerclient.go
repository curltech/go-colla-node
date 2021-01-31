package service

import (
	"github.com/curltech/go-colla-core/container"
	"github.com/curltech/go-colla-core/service"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/p2p/msg"
	"github.com/curltech/go-colla-core/logger"
	"sync"
	"errors"
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

func (this *PeerClientService) Validate(messagePayload *msg.MessagePayload) error {
	peerClient := messagePayload.Payload.(*entity.PeerClient)
	if peerClient == nil {
		return errors.New("NullPeerClient")
	}
	srcPeer := messagePayload.SrcPeer.(*entity.PeerClient)
	srcPeerId := srcPeer.PeerId
	senderPeerId := peerClient.PeerId
	if senderPeerId != srcPeerId {
		return errors.New("SenderAndSrcPeerPeerIdAreDifferent")
	}
	srcPublicKey := srcPeer.PublicKey
	senderPublicKey := peerClient.PublicKey
	if senderPublicKey != srcPublicKey {
		return errors.New("SenderAndSrcPeerPublicKeyAreDifferent")
	}
	expireDate := peerClient.ExpireDate
	if expireDate == 0 {
		return errors.New("Invalid expireDate")
	}
	return nil
}

func (this *PeerClientService) GetLocals(keyKind string, peerId string, mobile string, clientId string) ([]*entity.PeerClient, error) {
	var key string
	if keyKind == ns.PeerClient_KeyKind {
		if len(peerId) == 0 {
			return nil, errors.New("NullPeerId")
		}
		key = ns.GetPeerClientKey(peerId)
	} else if keyKind == ns.PeerClient_Mobile_KeyKind {
		if len(mobile) == 0 {
			return nil, errors.New("NullMobile")
		}
		key = ns.GetPeerClientMobileKey(mobile)
	} else {
		logger.Errorf("InvalidPeerClientKeyKind: %v", keyKind)
		return nil, errors.New("InvalidPeerClientKeyKind")
	}
	rec, err := dht.PeerEndpointDHT.GetLocal(key)
	if err != nil {
		logger.Errorf("failed to GetLocal by key: %v, err: %v", key, err)
		return nil, err
	}
	if rec != nil {
		peerClients := make([]*entity.PeerClient, 0)
		err = message.Unmarshal(rec.GetValue(), &peerClients)
		if err != nil {
			logger.Errorf("failed to Unmarshal record value with key: %v, err: %v", key, err)
			return nil, err
		}
		if len(clientId) > 0 {
			pcs := make([]*entity.PeerClient, 0)
			for _, peerClient := range peerClients {
				if peerClient.ClientId == clientId {
					pcs = append(pcs, peerClient)
				}
			}
			return pcs, nil
		} else {
			return peerClients, nil
		}
	}

	return nil, nil
}

func (this *PeerClientService) PutLocals(peerClients []*entity.PeerClient) error {
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

func (this *PeerClientService) GetValues(peerId string, mobile string) ([]*entity.PeerClient, error) {
	peerClients := make([]*entity.PeerClient, 0)
	var key string
	if len(peerId) > 0 {
		key = ns.GetPeerClientKey(peerId)
	} else if len(mobile) > 0 {
		key = ns.GetPeerClientMobileKey(mobile)
	} else {
		logger.Errorf("InvalidPeerClientKey")
		return nil, errors.New("InvalidPeerClientKey")
	}
	if config.Libp2pParams.FaultTolerantLevel == 0 {
		recvdVals, err := dht.PeerEndpointDHT.GetValues(key, config.Libp2pParams.Nvals)
		if err != nil {
			return nil, err
		}
		for _, recvdVal := range recvdVals {
			pcs := make([]*entity.PeerClient, 0)
			err = message.TextUnmarshal(string(recvdVal.Val), &pcs)
			if err != nil {
				logger.Errorf("failed to TextUnmarshal PeerClient value: %v, err: %v", recvdVal.Val, err)
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
		var locals []*entity.PeerClient
		var err error
		if len(peerId) > 0 {
			locals, err = this.GetLocals(ns.PeerClient_KeyKind, peerId, "", "")
		} else if len(mobile) > 0 {
			locals, err = this.GetLocals(ns.PeerClient_Mobile_KeyKind, "", mobile, "")
		}
		if err != nil {
			return nil, err
		}
		if len(locals) > 0 {
			for _, local := range locals {
				peerClients = append(peerClients, local)
			}
			this.Delete(locals, "")
		}
		// 查询non-local记录
		recvdVals, err := dht.PeerEndpointDHT.GetValues(key, config.Libp2pParams.Nvals)
		if err != nil {
			return nil, err
		}
		// 恢复local记录
		err = this.PutLocals(locals)
		if err != nil {
			return nil, err
		}
		// 整合记录
		for _, recvdVal := range recvdVals {
			pcs := make([]*entity.PeerClient, 0)
			err = message.TextUnmarshal(string(recvdVal.Val), &pcs)
			if err != nil {
				logger.Errorf("failed to TextUnmarshal PeerClient value: %v, err: %v", recvdVal.Val, err)
				return nil, err
			}
			for _, pc := range pcs {
				peerClients = append(peerClients, pc)
			}
		}
	}

	return peerClients, nil
}

func (this *PeerClientService) PutValues(peerClient *entity.PeerClient) error {
	err := this.PutValue(peerClient, ns.PeerClient_KeyKind)
	if err != nil {
		return err
	}
	return this.PutValue(peerClient, ns.PeerClient_Mobile_KeyKind)
}

func (this *PeerClientService) PutValue(peerClient *entity.PeerClient, keyKind string) error {
	bytePeerClient, err := message.Marshal(peerClient)
	if err != nil {
		return err
	}
	var key string
	if keyKind == ns.PeerClient_KeyKind {
		key = ns.GetPeerClientKey(peerClient.PeerId)
	} else if keyKind == ns.PeerClient_Mobile_KeyKind {
		key = ns.GetPeerClientMobileKey(peerClient.Mobile)
	} else {
		logger.Errorf("InvalidPeerClientKeyKind: %v", keyKind)
		return errors.New("InvalidPeerClientKeyKind")
	}

	return dht.PeerEndpointDHT.PutValue(key, bytePeerClient)
}
