package service

import (
	"github.com/curltech/go-colla-core/container"
	"github.com/curltech/go-colla-core/service"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/chain/entity"
)

/**
同步表结构，服务继承基本服务的方法
*/
type PeerTransactionService struct {
	service.OrmBaseService
}

var peerTransactionService = &PeerTransactionService{}

func GetPeerTransactionService() *PeerTransactionService {
	return peerTransactionService
}

func (this *PeerTransactionService) GetSeqName() string {
	return seqname
}

func (this *PeerTransactionService) NewEntity(data []byte) (interface{}, error) {
	entity := &entity.PeerTransaction{}
	if data == nil {
		return entity, nil
	}
	err := message.Unmarshal(data, entity)
	if err != nil {
		return nil, err
	}

	return entity, err
}

func (this *PeerTransactionService) NewEntities(data []byte) (interface{}, error) {
	entities := make([]*entity.PeerTransaction, 0)
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
	service.GetSession().Sync(new(entity.PeerTransaction))

	peerTransactionService.OrmBaseService.GetSeqName = peerTransactionService.GetSeqName
	peerTransactionService.OrmBaseService.FactNewEntity = peerTransactionService.NewEntity
	peerTransactionService.OrmBaseService.FactNewEntities = peerTransactionService.NewEntities
	service.RegistSeq(seqname, 0)
	container.RegistService(ns.PeerTransaction_Src_Prefix, peerTransactionService)
	container.RegistService(ns.PeerTransaction_Target_Prefix, peerTransactionService)
}
