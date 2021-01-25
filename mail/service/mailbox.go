package service

import (
	"github.com/curltech/go-colla-core/service"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/mail/entity"
)

/**
同步表结构，服务继承基本服务的方法
*/
type MailBoxService struct {
	service.OrmBaseService
}

var mailBoxService = &MailBoxService{}

func GetMailBoxService() *MailBoxService {
	return mailBoxService
}

func (this *MailBoxService) GetSeqName() string {
	return seqname
}

func (this *MailBoxService) NewEntity(data []byte) (interface{}, error) {
	entity := &entity.MailBox{}
	if data == nil {
		return entity, nil
	}
	err := message.Unmarshal(data, entity)
	if err != nil {
		return nil, err
	}

	return entity, err
}

func (this *MailBoxService) NewEntities(data []byte) (interface{}, error) {
	entities := make([]*entity.MailBox, 0)
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
	service.GetSession().Sync(new(entity.MailBox))

	mailBoxService.OrmBaseService.GetSeqName = mailBoxService.GetSeqName
	mailBoxService.OrmBaseService.FactNewEntity = mailBoxService.NewEntity
	mailBoxService.OrmBaseService.FactNewEntities = mailBoxService.NewEntities
}
