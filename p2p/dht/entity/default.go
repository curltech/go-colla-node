package entity

import (
	"github.com/curltech/go-colla-core/entity"
	"time"
)

/**
节点的位置，
*/
type PeerLocation struct {
	PeerId          string `xorm:"varchar(255) unique" json:"peerId,omitempty"`
	Kind            string `xorm:"varchar(255) notnull" json:"kind,omitempty"`
	Name            string `xorm:"varchar(255) notnull" json:"name,omitempty"`
	SecurityContext string `xorm:"varchar(1024)" json:"securityContext,omitempty"`
	/**
	libp2p的公私钥
	*/
	PeerPublicKey string `xorm:"varchar(128)" json:"peerPublicKey,omitempty"`
	/**
	openpgp的公私钥
	*/
	PublicKey      string     `xorm:"varchar(1024)" json:"publicKey,omitempty"`
	Address        string     `xorm:"varchar(1024) notnull" json:"address,omitempty"`
	LastUpdateTime *time.Time `json:"lastUpdateTime,omitempty"`
}

type PeerEntity struct {
	entity.StatusEntity `xorm:"extends"`
	PeerLocation        `xorm:"extends"`
	Mobile              string     `xorm:"varchar(255) notnull" json:"mobile,omitempty"`
	Email               string     `xorm:"varchar(512)" json:"email,omitempty"`
	StartDate           *time.Time `json:"startDate,omitempty"`
	EndDate             *time.Time `json:"endDate,omitempty"`
	ConnectSessionId    string     `xorm:"varchar(255)" json:"connectSessionId,omitempty"`
	CreditScore         uint64
	PreferenceScore     uint64
	BadCount            uint64
	StaleCount          uint64
	LastAccessMillis    int64
	LastAccessTime      *time.Time `json:"lastAccessTime,omitempty"`
	ActiveStatus        string     `xorm:"varchar(32)" json:"activeStatus,omitempty"`
	BlockId             string     `xorm:"varchar(255)" json:"blockId,omitempty"`
	Balance             float64    `json:"balance,omitempty"`
	Currency            string     `xorm:"varchar(32)" json:"currency,omitempty"`
	LastTransactionTime *time.Time `json:"lastTransactionTime,omitempty"`

	PreviousPublicKeySignature string `xorm:"-" json:"previousPublicKeySignature,omitempty"`
	Signature                  string `xorm:"-" json:"signature,omitempty"`
	SignatureData              string `xorm:"-" json:"signatureData,omitempty"`
	ExpireDate                 int64  `xorm:"-" json:"expireDate,omitempty"`
	//Version                    int        `xorm:"version"`
}

const (
	PeerType_MyselfPeer   string = "MyselfPeer"
	PeerType_PeerEndpoint string = "PeerEndpoint"
	PeerType_PeerClient   string = "PeerClient"
	PeerType_ChainApp     string = "ChainApp"
)

const (
	ActiveStatus_Up   string = "Up"
	ActiveStatus_Down string = "Down"
)

const (
	TransactionType_DataBlock string = "DataBlock"
)
