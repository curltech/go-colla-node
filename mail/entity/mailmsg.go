package entity

import (
	"github.com/curltech/go-colla-core/entity"
)

type MailMessage struct {
	entity.StatusEntity `xorm:"extends"`
	BoxName             string `xorm:"varchar(255) notnull" json:"boxName,omitempty"`
	Hash                string `xorm:"varchar(255) notnull" json:"hash,omitempty"`
	Size                uint64 `json:"size,omitempty"`
	SeenFlag            string `xorm:"varchar(255) notnull" json:"seenFlag,omitempty"`
	AnsweredFlag        string `xorm:"varchar(255) notnull" json:"answeredFlag,omitempty"`
	FlaggedFlag         string `xorm:"varchar(255) notnull" json:"flaggedFlag,omitempty"`
	DeletedFlag         string `xorm:"varchar(255) notnull" json:"deletedFlag,omitempty"`
	DraftFlag           string `xorm:"varchar(255) notnull" json:"draftFlag,omitempty"`
	RecentFlag          string `xorm:"varchar(255) notnull" json:"recentFlag,omitempty"`
	Body                []byte `json:"body,omitempty"`
}

func (MailMessage) TableName() string {
	return "mail_message"
}

func (MailMessage) KeyName() string {
	return "Hash"
}

func (MailMessage) IdName() string {
	return entity.FieldName_Id
}
