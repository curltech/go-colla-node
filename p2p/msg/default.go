package msg

import (
	"github.com/curltech/go-colla-core/crypto"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/msgtype"
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
	UUID string `json:",omitempty"`
	/**
	 * 最终的消息接收目标，如果当前节点不是最终的目标，可以进行转发
	 * 如果目标是服务器节点，直接转发，
	 * 如果目标是客户机节点，先找到客户机目前连接的服务器节点，也许就是自己，然后转发
	 */
	TargetPeerId string `json:",omitempty"`
	// 如果目标TargetPeerId是peerclient，在转发的时候，填充相应的peerclient的连接会话编号
	TargetConnectSessionId string `json:",omitempty"`
	TargetConnectPeerId    string `json:",omitempty"`
	/**
	以下三个字段方便对消息处理时寻找目的节点，topic是发送给主题的，TargetPeerId是发送给目标peer的
	*/
	Topic string `json:",omitempty"`
	/**
	src字段,ConnectSessionId在发送的时候不填，到接收端自动填充,第一个连接节点
	*/
	SrcConnectSessionId string `json:",omitempty"`
	SrcConnectPeerId    string `json:",omitempty"`
	// 本次的源连接节点
	LocalConnectPeerId  string `json:",omitempty"`
	LocalConnectAddress string `json:",omitempty"`
	// 源节点，在最初连接节点接收的时候填写
	SrcPeerId  string `json:",omitempty"`
	SrcAddress string `json:",omitempty"`
	// 本次的连接会话，在接收节点处填写
	ConnectSessionId string `json:",omitempty"`
	// 本次准备连接的节点，本次信息发送目标由此得出
	ConnectPeerId  string            `json:",omitempty"`
	ConnectAddress string            `json:",omitempty"`
	MessageType    msgtype.MsgType   `json:",omitempty"`
	MessageDirect  msgtype.MsgDirect `json:",omitempty"`
	/**
	 * 经过目标peer的公钥加密过的对称秘钥，这个对称秘钥是随机生成，每次不同，用于加密payload
	 */
	PayloadKey      string                  `json:",omitempty"`
	NeedCompress    bool                    `json:",omitempty"`
	NeedEncrypt     bool                    `json:",omitempty"`
	SecurityContext *crypto.SecurityContext `json:",omitempty"`
	/**
	 * 消息负载序列化后的寄送格式，再经过客户端自己的加密方式比如openpgp（更安全）加密，签名，压缩，base64处理后的字符串
	 */
	TransportPayload string `json:",omitempty"`
	/**
	 * 不跨网络传输，是transportPayload检验过后还原的对象，传输时通过转换成transportPayload传输
	 */
	Payload interface{} `json:"-,omitempty"`
	Tip     string      `json:",omitempty"`
	/**
	 * 负载json的源peer的签名
	 */
	PayloadSignature string `json:",omitempty"`
	/**
	 * 根据此字段来把TransportPayload对应的字节还原成Payload的对象，最简单的就是字符串
	 * 也可以是一个复杂的结构，但是dht的数据结构（peerendpoint），通用网络块存储（datablock）一般不用这种方式操作
	 * 而采用getvalue和putvalue的方式操作
	 */
	PayloadType     string     `json:",omitempty"`
	CreateTimestamp *time.Time `json:",omitempty"`
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

func Receive(chainMessage *ChainMessage) {

}

func Send(chainMessage *ChainMessage) {

}

////////////////////

/**
与peerclient通信使用原chainmessage消息格式
*/
type PCChainMessage struct {
	/**
	 * 双方的公钥不能被加密传输，因为需要根据公钥决定配对的是哪一个版本的私钥
	 *
	 * 对方的公钥有可能不存在，这时候数据没有加密，对称密钥也不存在
	 *
	 * 自己的公钥始终存在，因此签名始终可以验证
	 */
	SrcPublicKey    string `json:"srcPublicKey,omitempty"`
	TargetPublicKey string `json:"targetPublicKey,omitempty"`
	/**
	 * 消息负载的寄送格式，经过加密，签名，压缩，base64处理后的字符串
	 */
	TransportMessagePayload string `json:"transportMessagePayload,omitempty"`
	NeedCompress            string `json:"needCompress,omitempty"`
	NeedEncrypt             bool   `json:"needEncrypt,omitempty"`
	/**
	 * 负载json的源peer的签名
	 */
	PayloadSignature string `json:"payloadSignature,omitempty"`
	/**
	 * 经过目标peer的公钥加密过的对称秘钥，这个对称秘钥是随机生成，每次不同，用于加密payload
	 */
	PayloadKey            string                  `json:"payloadKey,omitempty"`
	SecurityContextString string                  `json:"securityContext,omitempty"`
	SecurityContext       *crypto.SecurityContext `json:"-,omitempty"`
	/**
	 * 不跨网络传输，是transportPayload检验过后还原的对象，传输时通过转换成transportPayload传输
	 */
	MessagePayload *MessagePayload `json:"-,omitempty"`
}

type MessagePayload struct {
	UUID string `json:"uuid,omitempty"`
	/**
	以下三个字段方便对消息处理时寻找目的节点，topic是发送给主题的，TargetPeerId是发送给目标peer的
	*/
	//Topic         string            `json:",omitempty"`
	TargetPeerId  string `json:"targetPeerId,omitempty"`
	TargetAddress string `json:"targetAddress,omitempty"`
	////TargetPeerType peertype.PeerType `json:",omitempty"`
	//ConnectionId  string            `json:",omitempty"`
	MessageType msgtype.MsgType `json:"messageType,omitempty"`
	//MessageDirect msgtype.MsgDirect `json:",omitempty"`
	/**
	 * 经过目标peer的公钥加密过的对称秘钥，这个对称秘钥是随机生成，每次不同，用于加密payload
	 */
	//PayloadKey      string                  `json:",omitempty"`
	//NeedCompress    bool                    `json:",omitempty"`
	//NeedEncrypt     bool                    `json:",omitempty"`
	//SecurityContext *crypto.SecurityContext `json:",omitempty"`
	/**
	 * 消息负载序列化后的寄送格式，再经过客户端自己的加密方式比如openpgp（更安全）加密，签名，压缩，base64处理后的字符串
	 */
	TransportPayload string `json:"transportPayload,omitempty"`
	/**
	 * 不跨网络传输，是transportPayload检验过后还原的对象，传输时通过转换成transportPayload传输
	 */
	Payload interface{} `json:"-,omitempty"`
	//Tip     string      `json:",omitempty"`
	/**
	 * 负载json的源peer的签名
	 */
	//PayloadSignature string `json:",omitempty"`
	/**
	 * 根据此字段来把TransportPayload对应的字节还原成Payload的对象，最简单的就是字符串
	 * 也可以是一个复杂的结构，但是dht的数据结构（peerendpoint），通用网络块存储（datablock）一般不用这种方式操作
	 * 而采用getvalue和putvalue的方式操作
	 */
	//PayloadType     string     `json:",omitempty"`
	PayloadClass string `json:"payloadClass,omitempty"`
	////PeerKeyType
	////payloadClass
	CreateTimestamp *time.Time `json:"createTimestamp,omitempty"`
	////SrcPeerType peertype.PeerType json:",omitempty"`
	SrcPeer interface{} `json:"srcPeer,omitempty"`
}

type WebsocketChainMessage struct {
	MessageType      string                  `json:"messageType,omitempty"`
	MessageSubType   string                  `json:"messageSubType,omitempty"`
	SrcPeerClient    *entity.PeerClient      `json:"srcPeerClient,omitempty"`
	TargetPeerClient *entity.PeerClient      `json:"targetPeerClient,omitempty"`
	Payload          *map[string]interface{} `json:"payload,omitempty"`
}

func (this *PCChainMessage) Marshal() ([]byte, error) {
	return message.Marshal(this)
}

func (this *PCChainMessage) Unmarshal(data []byte) {
	message.Unmarshal(data, this)
}

func (this *PCChainMessage) TextMarshal() (string, error) {
	return message.TextMarshal(this)
}

func (this *PCChainMessage) TextUnmarshal(data string) {
	message.TextUnmarshal(data, this)
}

func ReceivePC(chainMessage *PCChainMessage) {

}

func SendPC(chainMessage *PCChainMessage) {

}
