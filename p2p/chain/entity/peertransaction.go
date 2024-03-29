package entity

import (
	"github.com/curltech/go-colla-core/entity"
	"time"
)

type PeerTransaction struct {
	entity.StatusEntity    `xorm:"extends"`
	TransactionType        string     `xorm:"varchar(255)" json:"transactionType,omitempty"`
	SrcPeerId              string     `xorm:"varchar(255)" json:"srcPeerId,omitempty"`
	SrcPeerType            string     `xorm:"varchar(255)" json:"srcPeerType,omitempty"`
	PrimaryPeerId		   string     `xorm:"varchar(255)" json:"primaryPeerId,omitempty"`
	TargetPeerId           string     `xorm:"varchar(255)" json:"targetPeerId,omitempty"`
	TargetPeerType         string     `xorm:"varchar(255)" json:"targetPeerType,omitempty"`
	BlockId                string     `xorm:"varchar(255) notnull" json:"blockId,omitempty"`
	Amount                 float64    `json:"amount,omitempty"`
	Currency               string     `xorm:"varchar(32)" json:"currency,omitempty"`
	TransactionTime        *time.Time `json:"transactionTime,omitempty"`
	SrcPeerBeginBalance    float64    `json:"srcPeerBeginBalance,omitempty"`
	SrcPeerEndBalance      float64    `json:"srcPeerEndBalance,omitempty"`
	TargetPeerBeginBalance float64    `json:"targetPeerBeginBalance,omitempty"`
	TargetPeerEndBalance   float64    `json:"targetPeerEndBalance,omitempty"`
	ParentBusinessNumber   string     `xorm:"varchar(255)" json:"parentBusinessNumber,omitempty"`
	BusinessNumber         string     `xorm:"varchar(255)" json:"businessNumber,omitempty"`
	SliceNumber            uint64	  `xorm:"notnull" json:"sliceNumber"`
	CreateTimestamp        uint64     `json:"createTimestamp,omitempty"`
	Metadata 			   string     `xorm:"varchar(32768)" json:"metadata,omitempty"`
	Thumbnail 			   string     `xorm:"varchar(32768)" json:"thumbnail,omitempty"`
	Name 			   	   string     `xorm:"varchar(255)" json:"name,omitempty"`
	Description 		   string     `xorm:"varchar(255)" json:"description,omitempty"`
}

func (PeerTransaction) TableName() string {
	return "blc_peertransaction"
}

func (PeerTransaction) KeyName() string {
	return "SrcPeerId" // TargetPeerId
}

func (PeerTransaction) IdName() string {
	return entity.FieldName_Id
}
