package raft

import (
	"errors"
	"fmt"
	"github.com/curltech/go-colla-core/cache"
	entity2 "github.com/curltech/go-colla-core/entity"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/consensus"
	"github.com/curltech/go-colla-node/consensus/pbft/action"
	"github.com/curltech/go-colla-node/p2p/chain/entity"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	service2 "github.com/curltech/go-colla-node/p2p/chain/service"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"strings"
)

var MemCache = cache.NewMemCache("raft", 0, 0)

func getLogCacheKey(log *entity.ConsensusLog) string {
	key := fmt.Sprintf("%v:%v:%v:%v:%v:%v", log.PrimaryPeerId, log.BlockId, log.SliceNumber, log.PrimarySequenceId, log.PeerId, log.Status)
	return key
}

/**
 * leader 收到消息请求，向每个follow发送preprepared消息，实现与pbft一样
 *
 * @param chainMessage
 * @return
 */
func ReceiveConsensus(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Infof("ReceiveConsensus")
	dataBlock, err := consensus.GetDataBlock(chainMessage)
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
	log := consensus.CreateConsensusLog(chainMessage, dataBlock, myselfPeer, msgtype.CONSENSUS_PBFT_PREPREPARED)
	key := getLogCacheKey(log)
	MemCache.SetDefault(key, log)

	peerIds := consensus.ChooseConsensusPeer()
	if peerIds != nil && len(peerIds) > 2 {
		/**
		 * 发送CONSENSUS_PREPREPARED给副节点，告知主节点的状态
		 */
		for _, peerId := range peerIds {
			if myselfPeer.PeerId == peerId {
				continue
			}
			// 封装消息，异步发送
			go action.PrepreparedAction.Preprepared(peerId, dataBlock, "")
		}
	} else {
		logger.Errorf("LessPeerLocation")

		return nil, errors.New("LessPeerLocation")
	}

	return nil, nil
}

/**
follow收到Preprepared消息，准备完成后向leader发送prepared消息
*/
func ReceivePreprepared(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Infof("receive ReceivePreprepared")
	dataBlock, err := consensus.GetDataBlock(chainMessage)
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
	log.Status = msgtype.CONSENSUS_PBFT_PREPREPARED

	key := getLogCacheKey(log)
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
	log = consensus.CreateConsensusLog(chainMessage, dataBlock, myselfPeer, msgtype.CONSENSUS_PBFT_PREPREPARED)
	/**
	 * 准备CONSENSUS_PREPARED状态的消息
	 */
	log.Status = msgtype.CONSENSUS_PBFT_PREPARED
	/**
	 * 发送CONSENSUS_PREPARED给leader，表示准备完成，这里与pbft的差异是只发给leader，不需要发给每个follow
	 */
	peerIds := strings.Split(dataBlock.PeerIds, ",")
	if peerIds != nil && len(peerIds) > 2 {
		for _, id := range peerIds {
			if myselfPeer.PeerId == id {
				continue
			}
			go action.PreparedAction.Prepared(id, log, "")
		}
	} else {
		logger.Errorf("LessPeerLocation")

		return nil, errors.New("LessPeerLocation")
	}

	return nil, nil
}

/**
leader收到prepared消息，计算是否到达提交标准，向follow发送commited消息
与pbft的差异是只有leader能收到这种消息
*/
func ReceivePrepared(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	// 本节点是主副节点都会收到
	logger.Infof("receive ReceivePrepared")
	messageLog, err := consensus.GetConsensusLog(chainMessage)
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
		logger.Errorf("%v", messageLog)
		return nil, errors.New("SendPrimaryPreparedMessage")
	}
	if peerId == myPeerId {
		logger.Errorf("%v", messageLog)
		return nil, errors.New("SendMyselfMessage")
	}
	key := getLogCacheKey(messageLog)
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
		messageLog.Status = msgtype.CONSENSUS_PBFT_PREPARED
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
		log.Status = msgtype.CONSENSUS_PBFT_PREPARED
		count := 1
		for _, id := range peerIds {
			log.PeerId = id
			key = getLogCacheKey(log)
			l, found = MemCache.Get(key)
			if found {
				cacheLog = l.(*entity.ConsensusLog)
			}
			if cacheLog != nil {
				count++
			}
		}
		f := len(peerIds) / 3
		logger.Infof("findCountBy current status:%v;count:%v", msgtype.CONSENSUS_PBFT_PREPARED, count)
		// 收到足够的数目
		if count > 2*f {
			log.PeerId = myPeerId
			log.Status = msgtype.CONSENSUS_PBFT_COMMITED
			key = getLogCacheKey(log)
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
					go action.CommitedAction.Commited(id, log, "")
				}
			}
		}
	} else {
		logger.Errorf("LessPeerLocation")
		return nil, errors.New("LessPeerLocation")
	}

	return nil, nil
}

/**
 * follow收到commited消息，完成后，向leader发送reply消息，
 * 这里与pbft的差异是不用判断，完成后reply消息发送给leader，而不是发给客户端
 */
func ReceiveCommited(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	// 本节点是主副节点都会收到
	logger.Infof("receive ReceiveCommited")
	messageLog, err := consensus.GetConsensusLog(chainMessage)
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
		logger.Errorf("%v", messageLog)
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
	log.Status = msgtype.CONSENSUS_PBFT_COMMITED
	key := getLogCacheKey(log)
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
		messageLog.Status = msgtype.CONSENSUS_PBFT_COMMITED
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
			key = getLogCacheKey(log)
			l, found = MemCache.Get(key)
			if found {
				cacheLog = l.(*entity.ConsensusLog)
			}
			if cacheLog != nil {
				count++
			}
		}
		f := len(peerIds) / 3
		logger.Infof("findCountBy current status:%v;count:%v", msgtype.CONSENSUS_PBFT_COMMITED, count)
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
			log.Status = msgtype.CONSENSUS_PBFT_REPLY
			if dataBlock.PeerId != myPeerId {
				//go service.GetPeerEndpointService().modifyBadCount(-1)
				log.PublicKey = myselfPeer.PublicKey
				log.Address = myselfPeer.Address
				log.ClientPeerId = messageLog.ClientPeerId
				go action.ReplyAction.Reply(dataBlock.PeerId, log, "")
			} else {
				logger.Warnf("SameSrcAndTargetPeer")
			}
		} else {
			return nil, errors.New("NoDataBlockInCache")
		}
	} else {
		logger.Errorf("LessPeerLocation")
		return nil, errors.New("LessPeerLocation")
	}

	return nil, nil
}

/**
leader收到reply消息，判断完成后向客户就返回reply消息
与pbft的差异是pbft在上一步就完成了向客户端发送reply
*/
func ReceiveReply(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	return nil, nil
}

func init() {
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_PBFT, action.PbftAction.Send, ReceiveConsensus, action.PbftAction.Response)
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_PBFT_PREPREPARED, action.PrepreparedAction.Send, ReceivePreprepared, action.PrepreparedAction.Response)
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_PBFT_PREPARED, action.PreparedAction.Send, ReceivePrepared, action.PreparedAction.Response)
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_PBFT_COMMITED, action.CommitedAction.Send, ReceiveCommited, action.CommitedAction.Response)
}
