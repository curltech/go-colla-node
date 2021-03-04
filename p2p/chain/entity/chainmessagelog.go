package entity

import (
	"github.com/curltech/go-colla-core/entity"
	"time"
)

type ChainMessageLog struct {
	entity.StatusEntity `xorm:"extends"`
	UUID                string `xorm:"varchar(255)" json:"uuid,omitempty"`
	MessageType         string `xorm:"varchar(255)" json:"messageType,omitempty"`
	Action              string `xorm:"varchar(255)" json:"action,omitempty"`
	// 消息发送的源地址
	SrcPeerId string `xorm:"varchar(255)" json:"srcPeerId,omitempty"`
	// 消息发送的目标地址
	TargetPeerId string `xorm:"varchar(255)" json:"targetPeerId,omitempty"`
	// 消息发送的源地址
	SrcAddress string `xorm:"varchar(255)" json:"srcAddress,omitempty"`
	// 消息发送的目标地址
	TargetAddress string `xorm:"varchar(255)" json:"targetAddress,omitempty"`
	Payload       string `xorm:"varchar(10485760)" json:"payload,omitempty"`
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
	PayloadClass string `xorm:"varchar(255)" json:"payloadClass,omitempty"`
	// 请求的时间戳
	CreateTimestamp      time.Time `json:"createTimestamp,omitempty"`
	// 返回消息
	ResponseType    string `xorm:"varchar(255)" json:"responseType,omitempty"`
	ResponseCode    string `xorm:"varchar(255)" json:"responseCode,omitempty"`
	ResponseMessage string `xorm:"varchar(10485760)" json:"responseMessage,omitempty"`
	ResponsePayload string `xorm:"varchar(10485760)" json:"responsePayload,omitempty"`
}

func (ChainMessageLog) TableName() string {
	return "blc_chainmessagelog"
}

func (ChainMessageLog) IdName() string {
	return entity.FieldName_Id
}
