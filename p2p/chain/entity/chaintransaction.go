package entity

import "github.com/curltech/go-colla-core/entity"

type ChainTransaction struct {
	entity.BaseEntity `xorm:"extends"`
	BlockId           string      `json:"transactionId,omitempty"`
	TransactionId     string      `json:"transactionId,omitempty"`
	TxSequenceId      uint64      `json:"txSequenceId,omitempty"`
	TransactionType   string      `json:"transactionType,omitempty"`
	Signature         string      `json:"signature,omitempty"`
	PayloadHash       string      `json:"payloadHash,omitempty"`
	PayloadClass      string      `json:"payloadClass,omitempty"`
	ExpireDate        int64       `json:"expireDate,omitempty"`
	Metadata          string      `json:"metadata,omitempty"`
	MimeType          string      `json:"mimeType,omitempty"`
	TransportPayload  string      `json:"transportPayload,omitempty"`
	Payload           interface{} `json:"payload,omitempty"`
	NeedCompress      string      `json:"needCompress,omitempty"`
}
