package pbft

import (
	"errors"
	"fmt"
	"github.com/curltech/go-colla-core/cache"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/consensus/pbft/action"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/p2p/chain/entity"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	service2 "github.com/curltech/go-colla-node/p2p/chain/service"
	entity1 "github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/libp2p/go-libp2p-core/peer"
	kb "github.com/libp2p/go-libp2p-kbucket"
	"time"
)

var MemCache = cache.NewMemCache("pbft", 0, 0)

func getLogCacheKey(log *entity.PbftConsensusLog) string {
	key := fmt.Sprintf("%v:%v:%v:%v:%v:%v:%v", log.PrimaryPeerId, log.BlockId, log.TxSequenceId, log.SliceNumber, log.PrimarySequenceId, log.PeerId, log.Status)
	return key
}

/**
 * 只有主节点才能收到客户端的共识请求，这个请求一般是交易请求，消息payload为BlockEO
 *
 *
 * Block包含交易列表和参与者的秘钥，交易列表的负载时加密过的String，
 *
 * 参与者列表的第一条记录就是客户端的记录
 *
 * 交易列表包含客户端的签名
 *
 * 加密的秘钥是客户端和交易参与者的公钥加密共享临时秘钥
 *
 * 本方法是Pbft共识算法的起点，传来的交易消息被转换成共识日志信息
 *
 *
 * 客户端c向主节点p发送<REQUEST, o, t, c>请求。o: 请求的具体操作，t: 请求时客户端追加的时间戳，c：客户端标识。REQUEST:
 * 包含消息内容m，以及消息摘要d(m)。客户端对请求进行签名
 *
 * a. 客户端请求消息签名是否正确。非法请求丢弃。正确请求，分配一个编号n，编号n主要用于对客户端的请求进行排序。
 *
 * 然后广播一条<<PRE-PREPARE, v, n, d>, m>消息给其他副本节点。v：视图编号，d客户端消息摘要，m消息内容。
 *
 * <PRE-PREPARE, v, n, d>进行主节点签名。n是要在某一个范围区间内的[h, H]，具体原因参见垃圾回收章节。
 *
 * @param chainMessage
 * @return
 */
func ReceiveConsensus(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Infof("receive:%v", chainMessage)
	dataBlock := chainMessage.Payload.(*entity.DataBlock)
	var response *msg.ChainMessage = nil
	if dataBlock == nil {
		return nil, errors.New("NoPayload")
	}
	/**
	 * 主节点的属性
	 */
	myselfPeer := service.GetMyselfPeerService().GetFromCache()
	if myselfPeer.Id == 0 {
		return nil, errors.New("NoMyselfPeer")
	}

	/**
	 * 交易校验通过，主节点进入预准备状态，记录日志
	 */
	log := &entity.PbftConsensusLog{}
	log.BlockId = dataBlock.BlockId
	log.TxSequenceId = dataBlock.TxSequenceId
	log.SliceNumber = dataBlock.SliceNumber
	log.PrimarySequenceId = dataBlock.PrimarySequenceId
	log.PayloadHash = dataBlock.PayloadHash
	log.ClientPeerId = chainMessage.SrcPeerId
	log.ClientAddress = chainMessage.SrcAddress
	log.PeerId = myselfPeer.PeerId
	log.Address = myselfPeer.Address
	log.PublicKey = myselfPeer.PublicKey
	log.PrimaryPeerId = dataBlock.PrimaryPeerId
	log.PrimaryAddress = dataBlock.PrimaryAddress
	log.PrimaryPublicKey = dataBlock.PrimaryPublicKey
	log.Status = msgtype.CONSENSUS_PBFT_PREPREPARED
	t := time.Now()
	log.StatusDate = &t
	log.CreateTimestamp = time.Now()
	log.TransactionAmount = dataBlock.TransactionAmount
	service2.GetPbftConsensusLogService().Insert(log)
	key := getLogCacheKey(log)
	MemCache.SetDefault(key, log)

	consensusPeers := chooseConsensusPeer()
	/**
	 * 发送CONSENSUS_PREPREPARED给副节点，告知主节点的状态
	 */
	for _, consensusPeer := range consensusPeers {
		if myselfPeer.PeerId == consensusPeer.PeerId {
			continue
		}
		// 封装消息，异步发送
		go action.PrepreparedAction.Preprepared(consensusPeer.PeerId, dataBlock, "")
	}
	response = &msg.ChainMessage{MessageType: msgtype.CONSENSUS_PBFT, TargetPeerId: chainMessage.SrcPeerId,
		Payload: dataBlock, PayloadType: handler.PayloadType_DataBlock}

	return response, nil
}

func nearestConsensusPeer() []peer.ID {
	id := kb.ConvertKey(global.Global.PeerId.String())
	ids := dht.PeerEndpointDHT.RoutingTable.NearestPeers(id, 10)

	return ids
}

func chooseConsensusPeer() []*entity1.PeerEndpoint {
	return service.GetPeerEndpointService().GetRand(10)
}
