package msgtype

type MsgType string

const (
	// 未定义
	UNDEFINED = "UNDEFINED"
	// 消息返回正确
	OK = "OK"
	// 消息返回错误
	ERROR = "ERROR"
	// 发送消息peer对接收者来说不可信任
	UNTRUST = "UNTRUST"
	// 消息超时无响应
	NO_RESPONSE = "NO_RESPONSE"
	// 通用消息返回，表示可能异步返回
	RESPONSE = "RESPONSE"
	// 消息被拒绝
	REJECT = "REJECT"
	// 可做心跳测试
	PING = "PING"
	// 发送聊天报文
	CHAT       = "CHAT"
	FINDPEER   = "FINDPEER"
	GETVALUE   = "GETVALUE"
	PUTVALUE   = "PUTVALUE"
	SIGNAL     = "SIGNAL"
	IONSIGNAL  = "IONSIGNAL"
	FINDCLIENT = "FINDCLIENT"
	// PeerClient连接
	CONNECT = "TRANS_CONNECT"
	// PeerClient断开连接
	DISCONNECT = "TRANS_DISCONNECT"
	// PeerClient查找
	TRANSFINDCLIENT = "TRANS_FINDCLIENT"
	// PeerClient保存
	STORECLIENT = "TRANS_STORECLIENT"
	// DataBlock查找
	QUERYVALUE = "TRANS_QUERYVALUE"
	// DataBlock保存
	STORE = "CONSENSUS"
	// PeerTrans查找
	QUERYPEERTRANS = "TRANS_QUERYPEERTRANS"
	// PeerClient websocket消息
	WEBSOCKET = "TRANS_WEBSOCKET"
	// Websocket P2P
	PEERWEBSOCKET = "PEER_WEBSOCKET"
)

type MsgDirect string

const (
	MsgDirect_Request  = "Request"
	MsgDirect_Response = "Response"
)

// WebsocketChainMessage MessageType

const (
	// 退出登录
	SOCKET_LOGOUT = "LOGOUT"
	// RTC OFFER
	SOCKET_RTC_OFFER = "RTC_OFFER"
	// RTC ANSWER
	SOCKET_RTC_ANSWER = "RTC_ANSWER"
	// RTC CANDIDATE
	SOCKET_RTC_CANDIDATE = "RTC_CANDIDATE"
	// ADD LINKMAN INDIVIDUAL
	SOCKET_ADD_LINKMAN_INDIVIDUAL = "ADD_LINKMAN_INDIVIDUAL"
	// ADD LINKMAN INDIVIDUAL RECEIPT
	SOCKET_ADD_LINKMAN_INDIVIDUAL_RECEIPT = "ADD_LINKMAN_INDIVIDUAL_RECEIPT"
)

func init() {

}
