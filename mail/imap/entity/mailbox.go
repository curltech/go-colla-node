package entity

import (
	"github.com/curltech/go-colla-core/entity"
	"time"
)

type MailBox struct {
	entity.StatusEntity `xorm:"extends"`
	BoxName             string     `xorm:"varchar(255) notnull" json:"name,omitempty"`
	AccountName         string     `xorm:"varchar(255) notnull" json:"accountName,omitempty"`
	Subscribed          bool       `json:"subscribed,omitempty"`
	StartDate           *time.Time `json:"startDate,omitempty"`
	EndDate             *time.Time `json:"endDate,omitempty"`
}

func (MailBox) TableName() string {
	return "mail_box"
}

func (MailBox) KeyName() string {
	return "Name"
}

func (MailBox) IdName() string {
	return entity.FieldName_Id
}
