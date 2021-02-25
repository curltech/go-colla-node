package entity

import (
	"github.com/curltech/go-colla-core/entity"
	"time"
)

type BlockType string

const (
	BlockType_Temp  = "Temp"
	BlockType_Permanent = "Permanent"
)

type DataBlock struct {
	//entity.StatusEntity `xorm:"extends"`
	Id         uint64     `xorm:"pk" json:"-,omitempty"`
	CreateDate *time.Time `xorm:"created" json:"createDate,omitempty"`
	UpdateDate *time.Time `xorm:"updated" json:"updateDate,omitempty"`
	EntityId   string     `xorm:"-" json:"entityId,omitempty"`
	State      string     `xorm:"-" json:"state$,omitempty"`
	/**
	 * 当在一个事务中的一批交易开始执行的时候，首先保存交易，状态是Draft，
	 *
	 * 交易在共识后真正执行完毕状态变为effective，生成blockId
	 *
	 * 交易被取消，状态变成Ineffective
	 */
	Status       string     `xorm:"varchar(16)" json:"status,omitempty"`
	StatusReason string     `xorm:"varchar(255)" json:"statusReason,omitempty"`
	StatusDate   *time.Time `json:"statusDate,omitempty"`

	BlockId        string `xorm:"varchar(255) notnull" json:"blockId,omitempty"`
	BusinessNumber string `xorm:"varchar(255)" json:"businessNumber,omitempty"`
	BlockType      BlockType `xorm:"varchar(255)" json:"blockType,omitempty"`
	/**
	 * 双方的公钥不能被加密传输，因为需要根据公钥决定配对的是哪一个版本的私钥
	 *
	 * 对方的公钥有可能不存在，这时候数据没有加密，对称密钥也不存在
	 *
	 * 自己的公钥始终存在，因此签名始终可以验证
	 */
	PeerId          string `xorm:"varchar(255)" json:"peerId,omitempty"`
	PublicKey       string `xorm:"varchar(1024)" json:"publicKey,omitempty"`
	Address         string `xorm:"varchar(255)" json:"address,omitempty"`
	SecurityContext string `xorm:"varchar(255)" json:"securityContext,omitempty"`
	/**
	 * 经过源peer的公钥加密过的对称秘钥，这个对称秘钥是随机生成，每次不同，用于加密payload
	 *
	 * 如果本字段不为空，表示负载被加密了，至少是只有源peer能够解密
	 *
	 * 这样处理的好处是判断是否加密只需datablock，而且如果只是源peer的收藏，则transactionkey为空
	 */
	PayloadKey string `xorm:"varchar(1024)" json:"payloadKey,omitempty"`
	/**
	 * chainTransactions的寄送格式
	 */
	TransportPayload string `xorm:"varchar(10485760)" json:"transportPayload,omitempty"`
	/**
	 * 块负载的hash，是负载的decode64 hash，然后encode64
	 */
	PayloadHash string `xorm:"varchar(255)" json:"payloadHash,omitempty"`
	/**
	 * 负载源peer的签名
	 */
	Signature string `xorm:"varchar(1024)" json:"signature,omitempty"`
	/**
	 * transactionKeys的寄送格式，每个交易的第一个分片有数据，保证每个交易可以单独被查看
	 */
	TransportKey string `xorm:"varchar(32768)" json:"transportKey,omitempty"`
	/**
	 * 是负载的decode64 signature，然后encode64
	 */
	TransactionKeySignature string `xorm:"varchar(1024)" json:"transactionKeySignature,omitempty"`
	/**
	 * 本数据块的负载被拆成分片的总数，在没有分片前是1，分片后>=1，同一交易负载的交易号相同
	 */
	SliceSize uint64 `xorm:"notnull" json:"sliceSize,omitempty"`
	/**
	 * 数据块的本条分片是交易负载的第几个分片，在没有分片前是0，分片后>=0，但是<sliceSize
	 */
	SliceNumber uint64 `xorm:"notnull" json:"sliceNumber"`
	/**
	 * 分片hash汇总到交易，交易汇总到块hash
	 */
	StateHash       string            `xorm:"varchar(255)" json:"stateHash,omitempty"`
	TransactionKeys []*TransactionKey `xorm:"-" json:"transactionKeys,omitempty"`
	// 共识可能会引入的一些可选的元数据
	Metadata string `xorm:"varchar(255)" json:"metadata,omitempty"`
	MimeType string `xorm:"varchar(255)" json:"mimeType,omitempty"`

	// 由区块提议者填充的时间戳
	CreateTimestamp   uint64  `json:"createTimestamp,omitempty"`
	ExpireDate        int64   `json:"expireDate,omitempty"`
	TransactionAmount float64 `json:"transactionAmount"`

	// 请求的排好序的序号
	PrimarySequenceId uint64 `json:"primarySequenceId,omitempty"`
	PreviousBlockId   string `xorm:"varchar(255)" json:"previousBlockId,omitempty"`
	// 前一个区块的全局hash，也就是0:0的stateHash
	PreviousBlockHash string `xorm:"varchar(255)" json:"previousBlockHash,omitempty"`
	/**
	 * chainApp
	 */
	ChainAppPeerId    string `xorm:"varchar(255)" json:"chainAppPeerId,omitempty"`
	ChainAppPublicKey string `xorm:"varchar(1024)" json:"chainAppPublicKey,omitempty"`
	ChainAppAddress   string `xorm:"varchar(255)" json:"chainAppAddress,omitempty"`
	/**
	 * primary peer
	 */
	PrimaryPeerId    string `xorm:"varchar(255)" json:"primaryPeerId,omitempty"`
	PrimaryPublicKey string `xorm:"varchar(1024)" json:"primaryPublicKey,omitempty"`
	PrimaryAddress   string `xorm:"varchar(255)" json:"primaryAddress,omitempty"`

	PeerIds string `xorm:"varchar(255)" json:",omitempty"`
}

func (DataBlock) TableName() string {
	return "blc_datablock"
}

func (DataBlock) KeyName() string {
	return "BlockId" // PeerId
}

func (DataBlock) IdName() string {
	return entity.FieldName_Id
}
