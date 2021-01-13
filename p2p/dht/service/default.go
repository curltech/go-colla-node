package service

import (
	"github.com/curltech/go-colla-core/service"
)

/**
同步表结构，服务继承基本服务的方法
*/
type PeerEntityService struct {
	service.OrmBaseService
}

var peerEntityService = &PeerEntityService{}

func GetPeerEntityService() *PeerEntityService {
	return peerEntityService
}

var seqname = "seq_block"

func (this *PeerEntityService) GetSeqName() string {
	return seqname
}

func init() {
	service.RegistSeq(seqname, 0)
}
