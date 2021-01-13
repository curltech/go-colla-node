package entity

import "github.com/curltech/go-colla-core/entity"

/**
新的设计考虑成为peerendpoint的从表，存放特定的信息，而且peerId不是唯一的主键了
*/

type ChainApp struct {
	entity.StatusEntity `xorm:"extends"`
	PeerId              string `xorm:"varchar(255)" json:"peerId,omitempty"`
	AppType             string `xorm:"varchar(255)" json:"appType,omitempty"`
	RegistPeerId        string `xorm:"varchar(255)" json:"registPeerId,omitempty"`
	Path                string `xorm:"varchar(255)" json:"path,omitempty"`
	MainClass           string `xorm:"varchar(255)" json:"mainClass,omitempty"`
	CodePackage         string `xorm:"varchar(255)" json:"codePackage,omitempty"`
	AppHash             string `xorm:"varchar(255)" json:"appHash,omitempty"`
	AppSignature        string `xorm:"varchar(255)" json:"appSignature,omitempty"`
	ConnectSessionId    string `xorm:"varchar(255)" json:"connectSessionId,omitempty"`
	ActiveStatus        string `xorm:"varchar(255)" json:"activeStatus,omitempty"`
}

func (ChainApp) TableName() string {
	return "blc_chainapp"
}

func (ChainApp) KeyName() string {
	return "PeerId"
}

func (ChainApp) IdName() string {
	return entity.FieldName_Id
}
