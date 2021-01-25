package service

import (
	"github.com/curltech/go-colla-core/service"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/mail/entity"
)

/**
同步表结构，服务继承基本服务的方法
*/
type MailMessageService struct {
	service.OrmBaseService
}

var mailMessageService = &MailMessageService{}

func GetMailMessageService() *MailMessageService {
	return mailMessageService
}

func (this *MailMessageService) GetSeqName() string {
	return seqname
}

func (this *MailMessageService) NewEntity(data []byte) (interface{}, error) {
	entity := &entity.MailMessage{}
	if data == nil {
		return entity, nil
	}
	err := message.Unmarshal(data, entity)
	if err != nil {
		return nil, err
	}

	return entity, err
}

func (this *MailMessageService) NewEntities(data []byte) (interface{}, error) {
	entities := make([]*entity.MailMessage, 0)
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
	service.GetSession().Sync(new(entity.MailMessage))

	mailMessageService.OrmBaseService.GetSeqName = mailMessageService.GetSeqName
	mailMessageService.OrmBaseService.FactNewEntity = mailMessageService.NewEntity
	mailMessageService.OrmBaseService.FactNewEntities = mailMessageService.NewEntities
}
