package service

import (
	"github.com/curltech/go-colla-core/service"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/mail/imap/entity"
)

/**
同步表结构，服务继承基本服务的方法
*/
type MailAccountService struct {
	service.OrmBaseService
}

var mailAccountService = &MailAccountService{}

func GetMailAccountService() *MailAccountService {
	return mailAccountService
}

var seqname = "seq_mail"

func (this *MailAccountService) GetSeqName() string {
	return seqname
}

func (this *MailAccountService) NewEntity(data []byte) (interface{}, error) {
	entity := &entity.MailAccount{}
	if data == nil {
		return entity, nil
	}
	err := message.Unmarshal(data, entity)
	if err != nil {
		return nil, err
	}

	return entity, err
}

func (this *MailAccountService) NewEntities(data []byte) (interface{}, error) {
	entities := make([]*entity.MailAccount, 0)
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
	service.GetSession().Sync(new(entity.MailAccount))

	mailAccountService.OrmBaseService.GetSeqName = mailAccountService.GetSeqName
	mailAccountService.OrmBaseService.FactNewEntity = mailAccountService.NewEntity
	mailAccountService.OrmBaseService.FactNewEntities = mailAccountService.NewEntities
}
