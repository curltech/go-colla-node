package entity

import (
	baseentity "github.com/curltech/go-colla-core/entity"
	"time"
)

type MyselfPeer struct {
	PeerEntity     `xorm:"extends"`
	PeerPrivateKey string `xorm:"varchar(128)" json:"-,omitempty"`
	PrivateKey     string `xorm:"varchar(1024)" json:"-,omitempty"`
	/**
	 * 以下的字段是和证书相关，不是必须的
	 */
	CertType   string `xorm:"varchar(255)" json:"certType,omitempty"`
	CertFormat string `xorm:"varchar(255)" json:"certFormat,omitempty"`
	// peer的保护密码，不保存到数据库，hash后成为password
	OldPassword string `xorm:"-" json:"oldPassword,omitempty"`
	// peer的证书的原密码，申请新证书的时候必须提供，不保存数据库
	OldCertPassword string `xorm:"-" json:"oldCertPassword,omitempty"`
	// peer的新证书的密码，申请新证书的时候必须提供，不保存数据库
	NewCertPassword string `xorm:"-" json:"newCertPassword,omitempty"`
	// peer的hash后密码
	//Password    string `xorm:"varchar(255)" json:"-,omitempty"`
	CertContent string `xorm:"varchar(255)" json:"-,omitempty"`
	/**
	 * 主发现地址，表示可信的，可以推荐你的peer地址
	 */
	DiscoveryAddress string     `xorm:"varchar(255)" json:"discoveryAddress,omitempty"`
	LastFindNodeTime *time.Time `json:"lastFindNodeTime,omitempty"`
}

func (MyselfPeer) TableName() string {
	return "blc_myselfpeer"
}

func (MyselfPeer) KeyName() string {
	return "PeerId"
}

func (MyselfPeer) IdName() string {
	return baseentity.FieldName_Id
}
