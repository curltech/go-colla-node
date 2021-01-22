package pbft

import (
	"errors"
	"fmt"
	"github.com/curltech/go-colla-core/cache"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-core/util/reflect"
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

func ReceivePreprepared(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
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
	 * 通过检查PbftConsensusLogEO日志判断是否该接受还是拒绝
	 */
	primaryPeerId := dataBlock.PrimaryPeerId
	/**
	 * 发送节点必须是主节点
	 */
	if primaryPeerId != chainMessage.SrcPeerId {
		return nil, errors.New("SendPrepreparedMustPrimaryPeer")
	}
	/**
	 * 主节点是不会有CONSENSUS_PREPREPARED记录的
	 */
	if primaryPeerId == myselfPeer.PeerId {
		return nil, errors.New("PrepreparedInPrimaryPeer")
	}
	payloadHash := dataBlock.PayloadHash
	primarySequenceId := dataBlock.PrimarySequenceId
	blockId := dataBlock.BlockId
	txSequenceId := dataBlock.TxSequenceId
	sliceNumber := dataBlock.SliceNumber
	status := msgtype.CONSENSUS_PBFT_PREPREPARED
	log := &entity.PbftConsensusLog{}
	log.PrimaryPeerId = primaryPeerId
	log.BlockId = blockId
	log.TxSequenceId = txSequenceId
	log.SliceNumber = sliceNumber
	log.PrimarySequenceId = primarySequenceId
	log.PeerId = myselfPeer.PeerId
	log.Status = status

	key := getLogCacheKey(log)
	l, found := MemCache.Get(key)
	if found {
		cacheLog := l.(*entity.PbftConsensusLog)
		existPayloadHash := cacheLog.PayloadHash
		if payloadHash != existPayloadHash {
			// 记录坏行为的次数
			// go service.GetPeerEndpointService().Update()
			return nil, errors.New("ErrorPayloadHash")
		}
	} else {
		service2.GetDataBlockService().Save(dataBlock)
		// 每个副节点记录自己的Preprepared消息
		log.ClientPeerId = dataBlock.PeerId
		log.ClientAddress = dataBlock.Address
		log.ClientPublicKey = dataBlock.PublicKey
		log.PayloadHash = dataBlock.PayloadHash
		log.Address = myselfPeer.Address
		log.PublicKey = myselfPeer.PeerPublicKey
		log.PrimaryPeerId = dataBlock.PrimaryPeerId
		log.PrimaryAddress = dataBlock.PrimaryAddress
		log.PrimaryPublicKey = dataBlock.PrimaryPublicKey
		t := time.Now()
		log.StatusDate = &t
		log.PeerIds = dataBlock.PeerIds
		log.TransactionAmount = dataBlock.TransactionAmount
		service2.GetPbftConsensusLogService().Insert(log)
		MemCache.SetDefault(key, log)

		/**
		 * 准备CONSENSUS_PREPARED状态的消息
		 */
		l := reflect.New(log)
		log = l.(*entity.PbftConsensusLog)
		log.Status = msgtype.CONSENSUS_PBFT_PREPARED
		/**
		 * 发送CONSENSUS_PREPARED给副节点
		 *
		 * 如果接受，则生成prepare消息进行广播
		 *
		 * 备份节点发出PREPARE信息表示该节点同意主节点在view v中将编号n分配给请求m，
		 *
		 * 不发即表示不同意
		 *
		 */
		consensusPeers := make([]string, 0)
		message.TextUnmarshal(dataBlock.PeerIds, &consensusPeers)
		if consensusPeers != nil && len(consensusPeers) > 2 {
			for _, id := range consensusPeers {
				if myselfPeer.PeerId == id {
					continue
				}
				go action.PreparedAction.Prepared(id, log, "")
			}
		} else {
			logger.Errorf("LessPeerLocation")
			return nil, errors.New("LessPeerLocation")
		}
	}
	response = &msg.ChainMessage{MessageType: msgtype.CONSENSUS_PBFT_PREPREPARED, TargetPeerId: chainMessage.SrcPeerId,
		Payload: dataBlock, PayloadType: handler.PayloadType_DataBlock}

	return response, nil
}
