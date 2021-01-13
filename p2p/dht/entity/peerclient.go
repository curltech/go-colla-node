package entity

import (
	baseentity "github.com/curltech/go-colla-core/entity"
	"time"
)

/**
新的设计考虑成为peerendpoint的从表，存放特定的信息，而且peerId不是唯一的主键了，成为外键
*/
type PeerClient struct {
	//entity.StatusEntity `xorm:"extends"`
	Id           uint64     `xorm:"pk" json:"-,omitempty"`
	CreateDate   *time.Time `xorm:"created" json:"createDate,omitempty"`
	UpdateDate   *time.Time `xorm:"updated" json:"updateDate,omitempty"`
	EntityId     string     `xorm:"-" json:"entityId,omitempty"`
	State        string     `xorm:"-" json:"state,omitempty"`
	Status       string     `xorm:"varchar(16)" json:"status,omitempty"`
	StatusReason string     `xorm:"varchar(255)" json:"statusReason,omitempty"`
	StatusDate   *time.Time `json:"statusDate,omitempty"`

	PeerId     string `xorm:"varchar(255)" json:"peerId,omitempty"`
	ClientType string `xorm:"varchar(255)" json:"clientType,omitempty"`
	/**
	 * 客户连接到节点的位置
	 */
	ConnectPeerId string `xorm:"varchar(255)" json:"connectPeerId,omitempty"`
	/**
	 * 客户连接到节点的地址
	 */
	ConnectAddress string `xorm:"varchar(255)" json:"connectAddress,omitempty"`
	/**
	 * 客户连接到节点的公钥
	 */
	ConnectPublicKey string `xorm:"varchar(1024)" json:"connectPublicKey,omitempty"`
	// 对应的用户编号
	UserId string `xorm:"varchar(255)" json:"userId,omitempty"`
	// 用户名
	Name string `xorm:"varchar(255)" json:"name,omitempty"`
	// 用户头像（base64字符串）
	Avatar           string `xorm:"varchar(10485760)" json:"avatar,omitempty"`
	ConnectSessionId string `xorm:"varchar(255)" json:"connectSessionId,omitempty"`
	ClientId         string `xorm:"varchar(255)" json:"clientId,omitempty"`
	ClientDevice     string `xorm:"varchar(255)" json:"clientDevice,omitempty"`
	MobileVerified   string `xorm:"varchar(255)" json:"mobileVerified,omitempty"`
	// 可见性YYYYY (peerId、mobileNumber、groupChat、qrCode、contactCard）
	VisibilitySetting string `xorm:"varchar(255)" json:"visibilitySetting,omitempty"`

	LastUpdateTime             *time.Time `json:"lastUpdateTime,omitempty"`
	LastAccessTime             *time.Time `json:"lastAccessTime,omitempty"`
	StartDate                  *time.Time `json:"startDate,omitempty"`
	EndDate                    *time.Time `json:"endDate,omitempty"`
	Mobile                     string     `xorm:"varchar(255)" json:"mobile,omitempty"`
	Address                    string     `xorm:"varchar(255)" json:"address,omitempty"`
	PeerPublicKey              string     `xorm:"varchar(255)" json:"peerPublicKey,omitempty"`
	PublicKey                  string     `xorm:"varchar(1024)" json:"publicKey,omitempty"`
	PreviousPublicKeySignature string     `xorm:"-" json:"previousPublicKeySignature,omitempty"`
	Signature                  string     `xorm:"-" json:"signature,omitempty"`
	SignatureData              string     `xorm:"-" json:"signatureData,omitempty"`
	ExpireDate                 int64      `xorm:"-" json:"expireDate,omitempty"`

	ActiveStatus        string     `xorm:"varchar(255)" json:"activeStatus,omitempty"`
	BlockId             string     `xorm:"varchar(255)" json:"blockId,omitempty"`
	Balance             float64    `json:"balance,omitempty"`
	Currency            string     `xorm:"varchar(32)" json:"currency,omitempty"`
	LastTransactionTime *time.Time `json:"lastTransactionTime,omitempty"`
}

func (PeerClient) TableName() string {
	return "blc_peerclient"
}

func (PeerClient) KeyName() string {
	return "PeerId"
}

func (PeerClient) IdName() string {
	return baseentity.FieldName_Id
}
