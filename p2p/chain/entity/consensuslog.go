package entity

import (
	"github.com/curltech/go-colla-core/entity"
)

type ConsensusLog struct {
	entity.StatusEntity `xorm:"extends"`
	// 主节点
	PrimaryPeerId string `xorm:"varchar(255)" json:",omitempty"`
	// 请求的排好序的序号
	PrimarySequenceId uint64
	// 主节点的地址
	PrimaryAddress   string `xorm:"varchar(255)" json:",omitempty"`
	PrimaryPublicKey string `xorm:"varchar(1024)" json:",omitempty"`
	/**
	 * 发起协议的源节点peerClient
	 */
	ClientPeerId    string `xorm:"varchar(255)" json:",omitempty"`
	ClientPublicKey string `xorm:"varchar(1024)" json:",omitempty"`
	ClientAddress   string `xorm:"varchar(255)" json:",omitempty"`
	// 消息源节点的序号
	PeerId      string `xorm:"varchar(255)" json:",omitempty"`
	Address     string `xorm:"varchar(255)" json:",omitempty"`
	PublicKey   string `xorm:"varchar(1024)" json:",omitempty"`
	BlockId     string `xorm:"varchar(255)" json:",omitempty"`
	SliceNumber uint64
	// 交易请求的payloadhash
	PayloadHash string `xorm:"varchar(255)" json:",omitempty"`
	// 请求的结果状态
	ResponseStatus string `xorm:"varchar(255)" json:",omitempty"`

	TransactionAmount float64
	PeerIds           string `xorm:"varchar(255)" json:",omitempty"`
}

func (ConsensusLog) TableName() string {
	return "blc_consensuslog"
}

func (ConsensusLog) IdName() string {
	return entity.FieldName_Id
}