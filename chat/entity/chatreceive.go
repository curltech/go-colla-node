package entity

import (
	"github.com/curltech/go-colla-core/entity"
	"time"
)

/**
定义了两个实体和他们对应的表名
*/
/**
会话历史记录
*/
type ChatReceive struct {
	entity.BaseEntity `xorm:"extends"`
	MessageId         string `xorm:"varchar(255) notnull" json:",omitempty"`
	ReceivedTime      time.Time
}

func (ChatReceive) TableName() string {
	return "chat_receive"
}

func (ChatReceive) IdName() string {
	return entity.FieldName_Id
}
