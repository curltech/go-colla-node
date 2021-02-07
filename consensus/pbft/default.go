package pbft

import (
	"errors"
	"github.com/curltech/go-colla-core/cache"
	entity2 "github.com/curltech/go-colla-core/entity"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/consensus"
	"github.com/curltech/go-colla-node/consensus/action"
	"github.com/curltech/go-colla-node/p2p/chain/entity"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	service2 "github.com/curltech/go-colla-node/p2p/chain/service"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"strings"
)

var MemCache = cache.NewMemCache("pbft", 0, 0)

type PbftConsensus struct {
	consensus.Consensus
}

var pbftConsensus *PbftConsensus

func GetPbftConsensus() *PbftConsensus {

	return pbftConsensus
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
func (this *PbftConsensus) ReceiveConsensus(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Sugar.Infof("ReceiveConsensus")
	dataBlock, err := this.GetDataBlock(chainMessage)
	if err != nil {
		return nil, err
	}
	/**
	 * 主节点的属性
	 */
	myselfPeer := service.GetMyselfPeerService().GetFromCache()
	if myselfPeer.Id == 0 {
		return nil, errors.New("NoMyselfPeer")
	}
	if dataBlock.PrimaryPeerId == "" {
		dataBlock.PrimaryPeerId = myselfPeer.PeerId
		dataBlock.PrimaryAddress = myselfPeer.Address
		dataBlock.PrimaryPublicKey = myselfPeer.PublicKey
	} else {
		if dataBlock.PrimaryPeerId != myselfPeer.PeerId {
			return nil, errors.New("MustPrimaryPeer")
		}
	}

	/**
	 * 交易校验通过，主节点进入预准备状态，记录日志
	 */
	this.CreateConsensusLog(chainMessage, dataBlock, myselfPeer, msgtype.CONSENSUS_PREPREPARED)

	peerIds := this.ChooseConsensusPeer()
	if peerIds != nil && len(peerIds) > 2 {
		dataBlock.PeerIds = strings.Join(peerIds, ",")
		/**
		 * 发送CONSENSUS_PREPREPARED给副节点，告知主节点的状态
		 */
		for _, peerId := range peerIds {
			if myselfPeer.PeerId == peerId {
				continue
			}
			// 封装消息，异步发送
			go action.ConsensusAction.ConsensusDataBlock(peerId, msgtype.CONSENSUS_PREPREPARED, dataBlock, "")
		}
	} else {
		logger.Sugar.Errorf("LessPeerLocation")

		return nil, errors.New("LessPeerLocation")
	}

	return nil, nil
}

func (this *PbftConsensus) ReceivePreprepared(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Sugar.Infof("receive ReceivePreprepared")
	dataBlock, err := this.GetDataBlock(chainMessage)
	if err != nil {
		return nil, err
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
	sliceNumber := dataBlock.SliceNumber

	log := &entity.ConsensusLog{}
	log.PrimaryPeerId = primaryPeerId
	log.BlockId = blockId
	log.SliceNumber = sliceNumber
	log.PrimarySequenceId = primarySequenceId
	log.PeerId = myselfPeer.PeerId
	log.Status = msgtype.CONSENSUS_PREPREPARED

	key := this.GetLogCacheKey(log)
	l, found := MemCache.Get(key)
	if found {
		cacheLog, _ := l.(*entity.ConsensusLog)
		existPayloadHash := cacheLog.PayloadHash
		if payloadHash != existPayloadHash {
			// 记录坏行为的次数
			// go service.GetPeerEndpointService().Update()
			return nil, errors.New("ErrorPayloadHash")
		} else {
			return nil, nil
		}
	}
	service2.GetDataBlockService().Save(dataBlock)
	// 每个副节点记录自己的Preprepared消息
	log = this.CreateConsensusLog(chainMessage, dataBlock, myselfPeer, msgtype.CONSENSUS_PREPREPARED)
	/**
	 * 准备CONSENSUS_PREPARED状态的消息
	 */
	log.Status = msgtype.CONSENSUS_PREPARED
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
	peerIds := strings.Split(dataBlock.PeerIds, ",")
	if peerIds != nil && len(peerIds) > 2 {
		for _, id := range peerIds {
			if myselfPeer.PeerId == id {
				continue
			}
			go action.ConsensusAction.ConsensusLog(id, msgtype.CONSENSUS_PREPARED, log, "")
		}
	} else {
		logger.Sugar.Errorf("LessPeerLocation")

		return nil, errors.New("LessPeerLocation")
	}

	return nil, nil
}

/**
 * 节点接收其他节点的prepared消息，消息的payload为PbftConsensusLogEO
 *
 * 主节点会收到副节点发的prepared消息，等于commitNumber即可
 *
 * 副节点只会收到副节点的prepared消息，等于commitNumber-1即可
 *
 * 计算准备好的节点数目，满足以上条件，表示准备好，向其他节点发送提交消息
 *
 * 主节点和副本节点收到PREPARE消息，需要进行以下交验：
 *
 * a. 副本节点PREPARE消息签名是否正确。
 *
 * b. 当前副本节点是否已经收到了同一视图v下的n。
 *
 * c. n是否在区间[h, H]内。
 *
 * d. d是否和当前已收到PRE-PPREPARE中的d相同
 *
 * 非法请求丢弃。如果副本节点i收到了2f+1个验证通过的PREPARE消息，则向其他节点包括主节点发送一条<COMMIT, v, n, d, i>消息，v,
 * n, d, i与上述PREPARE消息内容相同。<COMMIT, v, n, d, i>进行副本节点i的签名。记录COMMIT消息到日志中，用于View
 * Change过程中恢复未完成的请求操作。记录其他副本节点发送的PREPARE消息到log中。
 */
func (this *PbftConsensus) ReceivePrepared(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	// 本节点是主副节点都会收到
	logger.Sugar.Infof("receive ReceivePrepared")
	messageLog, err := this.GetConsensusLog(chainMessage)
	if err != nil {
		return nil, err
	}
	/**
	 * 主节点的属性
	 */
	myselfPeer := service.GetMyselfPeerService().GetFromCache()
	if myselfPeer.Id == 0 {
		return nil, errors.New("NoMyselfPeer")
	}
	myPeerId := myselfPeer.PeerId
	/**
	* 通过检查PbftConsensusLogEO日志判断是否该接受还是拒绝
	 */
	primaryPeerId := messageLog.PrimaryPeerId
	payloadHash := messageLog.PayloadHash
	/**
	* 检查准备消息来源
	 */
	peerId := messageLog.PeerId
	if peerId == primaryPeerId {
		logger.Sugar.Errorf("%v", messageLog)
		return nil, errors.New("SendPrimaryPreparedMessage")
	}
	if peerId == myPeerId {
		logger.Sugar.Errorf("%v", messageLog)
		return nil, errors.New("SendMyselfMessage")
	}
	key := this.GetLogCacheKey(messageLog)
	var cacheLog *entity.ConsensusLog
	l, found := MemCache.Get(key)
	if found {
		cacheLog = l.(*entity.ConsensusLog)
	}
	// 已经记录了准备消息，重复收到，检查hash
	if cacheLog != nil {
		existPayloadHash := cacheLog.PayloadHash
		if payloadHash != existPayloadHash {
			//go service.GetPeerEndpointService().modifyBadCount()
			return nil, errors.New("ErrorPayloadHash")
		}
	} else {
		// 记录其他节点发来了Prepared消息，每个节点不会记录自己的Prepared消息
		messageLog.Id = 0
		messageLog.Status = msgtype.CONSENSUS_PREPARED
		service2.GetConsensusLogService().Insert(messageLog)
		MemCache.SetDefault(key, messageLog)
	}
	/**
	 * 通过检查PbftConsensusLogEO日志判断是否所有的节点包括自己都处于prepared状态
	 * 也就是说本节点已经知道所有的节点都发出了prepared通知 如果是则本节点处于prepared certificate状态，
	 * 最终，每个节点都会收到其他节点的prepared状态通知
	 */
	peerIds := strings.Split(messageLog.PeerIds, ",")
	if peerIds != nil && len(peerIds) > 2 {
		log := &entity.ConsensusLog{}
		log.PrimaryPeerId = primaryPeerId
		log.BlockId = messageLog.BlockId
		log.SliceNumber = messageLog.SliceNumber
		log.PrimarySequenceId = messageLog.PrimarySequenceId
		log.PeerId = peerId
		log.Status = msgtype.CONSENSUS_PREPARED
		count := 1
		for _, id := range peerIds {
			log.PeerId = id
			key = this.GetLogCacheKey(log)
			l, found = MemCache.Get(key)
			if found {
				cacheLog = l.(*entity.ConsensusLog)
			}
			if cacheLog != nil {
				count++
			}
		}
		f := len(peerIds) / 3
		logger.Sugar.Infof("findCountBy current status:%v;count:%v", msgtype.CONSENSUS_PREPARED, count)
		// 收到足够的数目
		if count > 2*f {
			log.PeerId = myPeerId
			log.Status = msgtype.CONSENSUS_COMMITED
			key = this.GetLogCacheKey(log)
			l, found = MemCache.Get(key)
			if found {
				cacheLog = l.(*entity.ConsensusLog)
			}
			if cacheLog == nil {
				log.Address = myselfPeer.Address
				log.PublicKey = myselfPeer.PublicKey
				MemCache.SetDefault(key, log)

				for _, id := range peerIds {
					/**
					 * 如果接受，则生成prepare消息进行广播
					 *
					 * 备份节点发出PREPARE信息表示该节点同意主节点在view v中将编号n分配给请求m，
					 *
					 * 不发即表示不同意
					 */
					if myPeerId == id {
						continue
					}
					go action.ConsensusAction.ConsensusLog(id, msgtype.CONSENSUS_COMMITED, log, "")
				}
			}
		}
	} else {
		logger.Sugar.Errorf("LessPeerLocation")
		return nil, errors.New("LessPeerLocation")
	}

	return nil, nil
}

/**
 * 节点接收其他节点的commit消息，消息的payload为PbftConsensusLogEO
 *
 * 节点会收到节点发的commit消息，等于commitNumber即可
 *
 * 计算commit的节点数目，满足以上条件，表示可以commit，直接向客户端发送提交成功消息
 *
 * 主节点和副本节点收到COMMIT消息，需要进行以下交验：
 *
 * a. 副本节点COMMIT消息签名是否正确。
 *
 * b. 当前副本节点是否已经收到了同一视图v下的n。
 *
 * c. d与m的摘要是否一致。
 *
 * d. n是否在区间[h, H]内。
 *
 * 非法请求丢弃。如果副本节点i收到了2f+1个验证通过的COMMIT消息，说明当前网络中的大部分节点已经达成共识，运行客户端的请求操作o，并返回<REPLY,
 * v, t, c, i,
 * r>给客户端，r：是请求操作结果，客户端如果收到f+1个相同的REPLY消息，说明客户端发起的请求已经达成全网共识，否则客户端需要判断是否重新发送请求给主节点。记录其他副本节点发送的COMMIT消息到log中。
 */
func (this *PbftConsensus) ReceiveCommited(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	// 本节点是主副节点都会收到
	logger.Sugar.Infof("receive ReceiveCommited")
	messageLog, err := this.GetConsensusLog(chainMessage)
	if err != nil {
		return nil, err
	}
	/**
	 * 主节点的属性
	 */
	myselfPeer := service.GetMyselfPeerService().GetFromCache()
	if myselfPeer.Id == 0 {
		return nil, errors.New("NoMyselfPeer")
	}
	myPeerId := myselfPeer.PeerId
	/**
	 * 通过检查PbftConsensusLogEO日志判断是否该接受还是拒绝
	 */
	peerId := messageLog.PeerId
	if peerId == myPeerId {
		logger.Sugar.Errorf("%v", messageLog)
		return nil, errors.New("SendMyselfMessage")
	}

	/**
	 * 检查自己是否重复收到CONSENSUS_COMMITED消息
	 */
	log := &entity.ConsensusLog{}
	log.PrimaryPeerId = messageLog.PrimaryPeerId
	log.BlockId = messageLog.BlockId
	log.SliceNumber = messageLog.SliceNumber
	log.PrimarySequenceId = messageLog.PrimarySequenceId
	log.PeerId = peerId
	log.Status = msgtype.CONSENSUS_COMMITED
	key := this.GetLogCacheKey(log)
	var cacheLog *entity.ConsensusLog
	l, found := MemCache.Get(key)
	if found {
		cacheLog = l.(*entity.ConsensusLog)
	}
	// 已经记录了提交消息，重复收到，检查hash
	if cacheLog != nil {
		existPayloadHash := cacheLog.PayloadHash
		if messageLog.PayloadHash != existPayloadHash {
			//go service.GetPeerEndpointService().modifyBadCount()

			return nil, errors.New("ErrorPayloadHash")
		}
	} else {
		// 记录其他节点发来了Commited消息，每个节点不会记录自己的Commited消息
		messageLog.Id = 0
		messageLog.Status = msgtype.CONSENSUS_COMMITED
		service2.GetConsensusLogService().Insert(messageLog)
		MemCache.SetDefault(key, messageLog)
	}

	/**
	 * 通过检查PbftConsensusLogEO日志判断是否所有的节点包括自己都处于Commited状态
	 * 也就是说本节点已经知道所有的节点都发出了Commited通知 如果是则本节点处于Commited certificate状态，
	 * 最终，每个节点都会收到其他节点的Commited状态通知
	 */
	peerIds := strings.Split(messageLog.PeerIds, ",")
	if peerIds != nil && len(peerIds) > 2 {
		count := 1
		for _, id := range peerIds {
			log.PeerId = id
			key = this.GetLogCacheKey(log)
			l, found = MemCache.Get(key)
			if found {
				cacheLog = l.(*entity.ConsensusLog)
			}
			if cacheLog != nil {
				count++
			}
		}
		f := len(peerIds) / 3
		logger.Sugar.Infof("findCountBy current status:%v;count:%v", msgtype.CONSENSUS_COMMITED, count)
		if count > 2*f {
			/**
			 * 数据块记录有效
			 */
			var dataBlock *entity.DataBlock
			//dataBlock = service2.GetDataBlockService().GetCachedDataBlock(blockId, txSequenceId, sliceNumber)
			if dataBlock != nil {
				status := dataBlock.Status
				if entity2.EntityStatus_Effective != status {
					dataBlock.Status = entity2.EntityStatus_Effective
					service2.GetDataBlockService().Update(dataBlock, nil, "")
				}
			}

			/**
			 * 异步返回客户端reply
			 */
			log.Status = msgtype.CONSENSUS_REPLY
			if dataBlock.PeerId != myPeerId {
				//go service.GetPeerEndpointService().modifyBadCount(-1)
				log.PublicKey = myselfPeer.PublicKey
				log.Address = myselfPeer.Address
				log.ClientPeerId = messageLog.ClientPeerId
				go action.ConsensusAction.ConsensusLog(dataBlock.PeerId, msgtype.CONSENSUS_REPLY, log, "")
			} else {
				logger.Sugar.Warnf("SameSrcAndTargetPeer")
			}
		} else {
			return nil, errors.New("NoDataBlockInCache")
		}
	} else {
		logger.Sugar.Errorf("LessPeerLocation")
		return nil, errors.New("LessPeerLocation")
	}

	return nil, nil
}

func init() {
	pbftConsensus = &PbftConsensus{}
	pbftConsensus.MemCache = MemCache
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_PBFT, action.ConsensusAction.Send, pbftConsensus.ReceiveConsensus, action.ConsensusAction.Response)
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_PBFT_PREPREPARED, action.ConsensusAction.Send, pbftConsensus.ReceivePreprepared, action.ConsensusAction.Response)
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_PBFT_PREPARED, action.ConsensusAction.Send, pbftConsensus.ReceivePrepared, action.ConsensusAction.Response)
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_PBFT_COMMITED, action.ConsensusAction.Send, pbftConsensus.ReceiveCommited, action.ConsensusAction.Response)
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_PBFT_REPLY, action.ConsensusAction.Send, pbftConsensus.ReceiveCommited, action.ConsensusAction.Response)
}
