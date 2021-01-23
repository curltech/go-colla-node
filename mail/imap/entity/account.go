package entity

import (
	"github.com/curltech/go-colla-core/entity"
	"time"
)

/**
邮件系统有多个账户account，每个账户有多个邮箱，mailbox，每个邮箱有多个消息，mailmessage
*/
type MailAccount struct {
	entity.StatusEntity `xorm:"extends"`
	Name                string     `xorm:"varchar(255) notnull" json:"name,omitempty"`
	Password            string     `xorm:"varchar(512)" json:"password,omitempty"`
	PeerId              string     `xorm:"varchar(512)" json:"peerId,omitempty"`
	StartDate           *time.Time `json:"startDate,omitempty"`
	EndDate             *time.Time `json:"endDate,omitempty"`
}

func (MailAccount) TableName() string {
	return "mail_account"
}

func (MailAccount) KeyName() string {
	return "Name"
}

func (MailAccount) IdName() string {
	return entity.FieldName_Id
}
