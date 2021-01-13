package entity

import "github.com/curltech/go-colla-core/entity"

type TransactionKey struct {
	entity.BaseEntity `xorm:"extends"`
	BlockId           string `xorm:"varchar(255)" json:"blockId,omitempty"`
	PeerId            string `xorm:"varchar(255)" json:"peerId,omitempty"`
	/**
	 * 经过目标peer的公钥加密过的对称秘钥，这个对称秘钥是随机生成，每次不同，用于加密payload
	 */
	PayloadKey string `xorm:"varchar(1024)" json:"payloadKey,omitempty"`
	PublicKey  string `xorm:"varchar(1024)" json:"publicKey,omitempty"`
	Address    string `xorm:"varchar(255)" json:"address,omitempty"`
	PeerType   string `xorm:"varchar(255)" json:"peerType,omitempty"`
	/**
	Key是BlockId:PeerId，用于Key检索和存储
	*/
	Key string `xorm:"varchar(255)" json:"key,omitempty"`
}

func (TransactionKey) TableName() string {
	return "blc_transactionkey"
}

func (TransactionKey) KeyName() string {
	return "BlockId" // PeerId
}

func (TransactionKey) IdName() string {
	return entity.FieldName_Id
}
