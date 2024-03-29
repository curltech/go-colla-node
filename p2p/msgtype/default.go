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
	REJECT     = "REJECT"
	ManageRoom = "MANAGEROOM"
	// 可做心跳测试
	PING = "PING"
	// 发送聊天报文
	CHAT      = "CHAT"
	P2PCHAT   = "P2PCHAT"
	FINDPEER  = "FINDPEER"
	GETVALUE  = "GETVALUE"
	PUTVALUE  = "PUTVALUE"
	SIGNAL    = "SIGNAL"
	IONSIGNAL = "IONSIGNAL"
	// PEERENDPOINT更新
	PEERENDPOINT = "PEERENDPOINT"
	// PeerClient连接
	CONNECT = "CONNECT"
	// PeerClient查找
	FINDCLIENT = "FINDCLIENT"
	// DataBlock查找
	QUERYVALUE = "QUERYVALUE"
	// PeerTrans查找
	QUERYPEERTRANS = "QUERYPEERTRANS"
	// DataBlock保存共识消息
	CONSENSUS                  = "CONSENSUS"
	CONSENSUS_REPLY            = "CONSENSUS_REPLY"
	CONSENSUS_PREPREPARED      = "CONSENSUS_PREPREPARED"
	CONSENSUS_PREPARED         = "CONSENSUS_PREPARED"
	CONSENSUS_COMMITED         = "CONSENSUS_COMMITED"
	CONSENSUS_RAFT             = "CONSENSUS_RAFT"
	CONSENSUS_RAFT_REPLY       = "CONSENSUS_RAFT_REPLY"
	CONSENSUS_RAFT_PREPREPARED = "CONSENSUS_RAFT_PREPREPARED"
	CONSENSUS_RAFT_PREPARED    = "CONSENSUS_RAFT_PREPARED"
	CONSENSUS_RAFT_COMMITED    = "CONSENSUS_RAFT_COMMITED"
	CONSENSUS_PBFT             = "CONSENSUS_PBFT"
	CONSENSUS_PBFT_REPLY       = "CONSENSUS_PBFT_REPLY"
	CONSENSUS_PBFT_PREPREPARED = "CONSENSUS_PBFT_PREPREPARED"
	CONSENSUS_PBFT_PREPARED    = "CONSENSUS_PBFT_PREPARED"
	CONSENSUS_PBFT_COMMITED    = "CONSENSUS_PBFT_COMMITED"
)

type MsgDirect string

const (
	MsgDirect_Request  = "Request"
	MsgDirect_Response = "Response"
)

// Chat MessageType

const (
	// 退出登录
	CHAT_LOGOUT = "LOGOUT"
	// ADD LINKMAN INDIVIDUAL
	CHAT_ADD_LINKMAN_INDIVIDUAL = "ADD_LINKMAN_INDIVIDUAL"
	// ADD LINKMAN INDIVIDUAL RECEIPT
	CHAT_ADD_LINKMAN_INDIVIDUAL_RECEIPT = "ADD_LINKMAN_INDIVIDUAL_RECEIPT"
)

func init() {

}
