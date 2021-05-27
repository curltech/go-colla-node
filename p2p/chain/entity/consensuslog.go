package entity

import (
	"github.com/curltech/go-colla-core/entity"
)

type ConsensusLog struct {
	entity.StatusEntity `xorm:"extends"`
	// 主节点
	PrimaryPeerId string `xorm:"varchar(255)" json:"primaryPeerId,omitempty"`
	// 请求的排好序的序号
	PrimarySequenceId uint64 `json:"primarySequenceId,omitempty"`
	// 主节点的地址
	PrimaryAddress   string `xorm:"varchar(255)" json:"primaryAddress,omitempty"`
	PrimaryPublicKey string `xorm:"varchar(1024)" json:"primaryPublicKey,omitempty"`
	/**
	 * 发起协议的源节点peerClient
	 */
	ClientPeerId    string `xorm:"varchar(255)" json:"clientPeerId,omitempty"`
	ClientPublicKey string `xorm:"varchar(1024)" json:"clientPublicKey,omitempty"`
	ClientAddress   string `xorm:"varchar(255)" json:"clientAddress,omitempty"`
	// 消息源节点的序号
	PeerId      string `xorm:"varchar(255)" json:"peerId,omitempty"`
	Address     string `xorm:"varchar(255)" json:"address,omitempty"`
	PublicKey   string `xorm:"varchar(1024)" json:"publicKey,omitempty"`
	BlockId     string `xorm:"varchar(255)" json:"blockId,omitempty"`
	BlockType   BlockType `xorm:"varchar(255)" json:"blockType,omitempty"`
	SliceNumber uint64 `json:"sliceNumber,omitempty"`
	// 交易请求的payloadhash
	PayloadHash string `xorm:"varchar(255)" json:"payloadHash,omitempty"`
	// 请求的结果状态
	ResponseStatus string `xorm:"varchar(255)" json:"responseStatus,omitempty"`

	TransactionAmount float64 `json:"transactionAmount,omitempty"`
	PeerIds           string `xorm:"varchar(2048)" json:"peerIds,omitempty"`
}

func (ConsensusLog) TableName() string {
	return "blc_consensuslog"
}

func (ConsensusLog) IdName() string {
	return entity.FieldName_Id
}
