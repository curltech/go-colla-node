package entity

import baseentity "github.com/curltech/go-colla-core/entity"

type PeerEndpoint struct {
	PeerEntity       `xorm:"extends"`
	EndpointType     string `xorm:"varchar(255)" json:"endpointType,omitempty"`
	DiscoveryAddress string `xorm:"varchar(255)" json:"discoveryAddress,omitempty"`
	Sdp              string `xorm:"varchar(3096)" json:"sdp,omitempty"`
}

func (PeerEndpoint) TableName() string {
	return "blc_peerendpoint"
}

func (PeerEndpoint) KeyName() string {
	return "PeerId"
}

func (PeerEndpoint) IdName() string {
	return baseentity.FieldName_Id
}
