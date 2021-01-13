package entity

import (
	"github.com/curltech/go-colla-core/entity"
	"time"
)

type ChainMessageLog struct {
	entity.StatusEntity `xorm:"extends"`
	UUID                string `xorm:"varchar(255)" json:",omitempty"`
	MessageType         string `xorm:"varchar(255)" json:",omitempty"`
	Action              string `xorm:"varchar(255)" json:",omitempty"`
	// 消息发送的源地址
	SrcPeerId string `xorm:"varchar(255)" json:",omitempty"`
	// 消息发送的目标地址
	TargetPeerId string `xorm:"varchar(255)" json:",omitempty"`
	// 消息发送的源地址
	SrcAddress string `xorm:"varchar(255)" json:",omitempty"`
	// 消息发送的目标地址
	TargetAddress string `xorm:"varchar(255)" json:",omitempty"`
	Payload       string `xorm:"varchar(10485760)" json:",omitempty"`
	/**
	 * 定义消息的负载的类型
	 *
	 * 可以是定义消息类型对应的负载的实体类型，比如PeerEndpointEO
	 *
	 * 也可以是列表List
	 *
	 * 其取值来源于消息构造时的设置，如果没有设置，默认为消息类型的实体类型
	 *
	 */
	PayloadClass string `xorm:"varchar(255)" json:",omitempty"`
	// 请求的时间戳
	CreateTimestamp      time.Time
	CreateTimestampNanos uint64
	// 返回消息
	ResponseType    string `xorm:"varchar(255)" json:",omitempty"`
	ResponseCode    string `xorm:"varchar(255)" json:",omitempty"`
	ResponseMessage string `xorm:"varchar(10485760)" json:",omitempty"`
	ResponsePayload string `xorm:"varchar(10485760)" json:",omitempty"`
}

func (ChainMessageLog) TableName() string {
	return "blc_chainmessagelog"
}

func (ChainMessageLog) IdName() string {
	return entity.FieldName_Id
}
