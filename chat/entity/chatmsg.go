package entity

import (
	"github.com/curltech/go-colla-core/entity"
)

/**
定义了两个实体和他们对应的表名
*/
/**
会话历史记录
*/
type ChatMessage struct {
	entity.BaseEntity `xorm:"extends"`
	MessageId         string `xorm:"varchar(255) notnull" json:",omitempty"`
	Content           string `xorm:"varchar(32)" json:",omitempty"`
}

func (ChatMessage) TableName() string {
	return "chat_message"
}

func (ChatMessage) IdName() string {
	return entity.FieldName_Id
}
