package entity

import (
	"github.com/curltech/go-colla-core/crypto"
	"github.com/curltech/go-colla-core/entity"
	"github.com/curltech/go-colla-core/util/message"
	"time"
)

type StreamCloseFlag uint

const (
	StreamCloseFlag_Open  = 0
	StreamCloseFlag_Close = 1
	StreamCloseFlag_Reset = 2
)

/**
采用libp2p实现后，chainmessage只用于自定义协议点对点的数据传输，不涉及网络递归的存放和查找
因此结构需要简化，不需要两层的结构了
*/
type ChainMessage struct {
	Id         uint64     `xorm:"pk" json:"id,omitempty"`
	CreateDate *time.Time `xorm:"created" json:"createDate,omitempty"`
	UpdateDate *time.Time `xorm:"updated" json:"updateDate,omitempty"`
	UUID       string     `xorm:"varchar(255)" json:"uuid,omitempty"`
	Topic      string     `xorm:"varchar(255)" json:"topic,omitempty"`
	/**
	 * 说明的场景是peerclient->peerendpoint->peerendpoint->perclient
	 * 最终的消息接收目标，如果当前节点不是最终的目标，可以进行转发
	 * 如果目标是服务器节点，直接转发，
	 * 如果目标是客户机节点，先找到客户机目前连接的服务器节点，也许就是自己，然后转发
	 */
	/**
	 *	以下四个字段方便对消息处理时寻找目的节点，topic是发送给主题的，TargetPeerId是发送给最终目标peer的
	 */
	TargetPeerId   string `xorm:"varchar(255)" json:"targetPeerId,omitempty"`
	TargetClientId string `xorm:"varchar(255)" json:"targetClientId,omitempty"`
	//目标客户端当前的连接节点和会话需要根据TargetPeerId实时查询得出
	TargetConnectSessionId string `xorm:"varchar(255)" json:"targetConnectSessionId,omitempty"`
	TargetConnectPeerId    string `xorm:"varchar(255)" json:"targetConnectPeerId,omitempty"`
	TargetConnectAddress   string `xorm:"varchar(255)" json:"targetConnectAddress,omitempty"`
	/**
	src字段,SrcConnectSessionId在发送的时候不填，到接收端自动填充，表示src的连接peerendpoint
	最初的发送peerclient信息，在最初连接节点接收的时候填写
	*/
	SrcConnectSessionId string `xorm:"varchar(255)" json:"srcConnectSessionId,omitempty"`
	SrcConnectPeerId    string `xorm:"varchar(255)" json:"srcConnectPeerId,omitempty"`
	SrcPeerId           string `xorm:"varchar(255)" json:"srcPeerId,omitempty"`
	SrcClientId         string `xorm:"varchar(255)" json:"srcClientId,omitempty"`
	SrcConnectAddress   string `xorm:"varchar(255)" json:"srcConnectAddress,omitempty"`
	// 本次连接peerendpoint的信息，ConnectSessionId在接收节点处填写，一般与target的值一致
	// 因为chain message趋向与发送给目标的连接peerendpoint
	ConnectSessionId string `xorm:"varchar(255)" json:"connectSessionId,omitempty"`
	ConnectPeerId    string `xorm:"varchar(255)" json:"connectPeerId,omitempty"`
	ConnectAddress   string `xorm:"varchar(255)" json:"connectAddress,omitempty"`
	// 消息的属性
	MessageType   string `xorm:"varchar(255)" json:"messageType,omitempty"`
	MessageDirect string `xorm:"varchar(255)" json:"messageDirect,omitempty"`
	/**
	 * 经过目标peer的公钥加密过的对称秘钥，这个对称秘钥是随机生成，每次不同，用于加密payload
	 */
	PayloadKey      string                  `xorm:"varchar(255)" json:"payloadKey,omitempty"`
	NeedCompress    bool                    `json:"needCompress,omitempty"`
	NeedEncrypt     bool                    `json:"needEncrypt,omitempty"`
	SecurityContext *crypto.SecurityContext `xorm:"-" json:"securityContext,omitempty"`
	/**
	 * 消息负载序列化后的寄送格式，再经过客户端自己的加密方式比如openpgp（更安全）加密，签名，压缩，base64处理后的字符串
	 */
	TransportPayload string `xorm:"text" json:"transportPayload,omitempty"`
	/**
	 * 不跨网络传输，是transportPayload检验过后还原的对象，传输时通过转换成transportPayload传输
	 */
	Payload interface{} `xorm:"-" json:"-"`
	Tip     string      `json:"tip,omitempty"`
	/**
	 * 负载json的源peer的签名
	 */
	PayloadSignature                  string `json:"payloadSignature,omitempty"`
	PreviousPublicKeyPayloadSignature string `json:"previousPublicKeyPayloadSignature,omitempty"`
	/**
	 * 根据此字段来把TransportPayload对应的字节还原成Payload的对象，最简单的就是字符串
	 * 也可以是一个复杂的结构，但是dht的数据结构（peerendpoint），通用网络块存储（datablock）一般不用这种方式操作
	 * 而采用getvalue和putvalue的方式操作
	 */
	PayloadType     string     `json:"payloadType,omitempty"`
	CreateTimestamp *time.Time `json:"createTimestamp,omitempty"`
	StatusCode      int        `json:"statusCode,omitempty"`
}

func (this *ChainMessage) Marshal() ([]byte, error) {
	return message.Marshal(this)
}

func (this *ChainMessage) Unmarshal(data []byte) {
	message.Unmarshal(data, this)
}

func (this *ChainMessage) TextMarshal() (string, error) {
	return message.TextMarshal(this)
}

func (this *ChainMessage) TextUnmarshal(data string) {
	message.TextUnmarshal(data, this)
}

func (ChainMessage) TableName() string {
	return "blc_chainmessage"
}

func (ChainMessage) KeyName() string {
	return "uuid"
}

func (ChainMessage) IdName() string {
	return entity.FieldName_Id
}
