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
	Flag                string     `xorm:"varchar(255) notnull" json:"flag,omitempty"`
	PermanentFlag       string     `xorm:"varchar(255) notnull" json:"permanentFlag,omitempty"`
	ReadOnly            bool       `json:"readOnly,omitempty"`
	// 第一个未读消息的序列号
	UnseenSeqNum uint32 `json:"unseenSeqNum,omitempty"`
	// 消息的数量
	MessageNum uint32 `json:"messageNum,omitempty"`
	// 最后一次打开时未读消息的数量
	Recent uint32 `json:"recent,omitempty"`
	// 未读消息的数量
	Unseen uint32 `json:"unseen,omitempty"`
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
