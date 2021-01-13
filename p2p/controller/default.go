package controller

type P2pParam struct {
	Id           string `json:",omitempty"`
	TargetPeerId string `json:",omitempty"`
	PeerId       string `json:",omitempty"`
	Addr         string `json:",omitempty"`
	Key          string `json:",omitempty"`
	PayloadType  string `json:",omitempty"`
	Value        string `json:",omitempty"`
}
