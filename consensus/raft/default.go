package raft

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

var MemCache = cache.NewMemCache("raft", 0, 0)

type RaftConsensus struct {
	consensus.Consensus
}

var raftConsensus *RaftConsensus

func GetRaftConsensus() *RaftConsensus {

	return raftConsensus
}

/**
 * leader 收到消息请求，向每个follow发送preprepared消息，实现与pbft一样
 *
 * @param chainMessage
 * @return
 */
func (this *RaftConsensus) ReceiveConsensus(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Infof("ReceiveConsensus")
	dataBlock, err := this.GetDataBlock(chainMessage)
	if err != nil {
		return nil, err
	}
	/**
	 * 填充主节点的属性
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
	log := this.CreateConsensusLog(chainMessage, dataBlock, myselfPeer, msgtype.CONSENSUS_PREPREPARED)
	key := this.GetLogCacheKey(log)
	MemCache.SetDefault(key, log)
	key = this.GetDataBlockCacheKey(dataBlock.BlockId, dataBlock.SliceNumber)
	MemCache.SetDefault(key, dataBlock)

	peerIds := this.ChooseConsensusPeer()
	if peerIds != nil && len(peerIds) > 2 {
		log.PeerIds = strings.Join(peerIds, ",")
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
		logger.Errorf("LessPeerLocation")

		return nil, errors.New("LessPeerLocation")
	}

	return nil, nil
}

/**
follow收到Preprepared消息，准备完成后向leader发送prepared消息
*/
func (this *RaftConsensus) ReceivePreprepared(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Infof("receive ReceivePreprepared")
	dataBlock, err := this.GetDataBlock(chainMessage)
	if err != nil {
		return nil, err
	}
	/**
	 * 校验自己
	 */
	myselfPeer := service.GetMyselfPeerService().GetFromCache()
	if myselfPeer.Id == 0 {
		return nil, errors.New("NoMyselfPeer")
	}
	/**
	 * 通过检查ConsensusLog日志判断是否该接受还是拒绝
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
	//service2.GetDataBlockService().Save(dataBlock)
	// 每个副节点记录自己的Preprepared消息
	log = this.CreateConsensusLog(chainMessage, dataBlock, myselfPeer, msgtype.CONSENSUS_PREPREPARED)
	/**
	 * 准备CONSENSUS_PREPARED状态的消息
	 */
	log.Status = msgtype.CONSENSUS_PREPARED
	key = this.GetLogCacheKey(log)
	MemCache.SetDefault(key, log)
	key = this.GetDataBlockCacheKey(dataBlock.BlockId, dataBlock.SliceNumber)
	MemCache.SetDefault(key, dataBlock)
	/**
	 * 发送CONSENSUS_PREPARED给leader，表示准备完成，这里与pbft的差异是只发给leader，不需要发给每个follow
	 */
	go action.ConsensusAction.ConsensusLog(log.PrimaryPeerId, msgtype.CONSENSUS_PREPARED, log, "")

	return nil, nil
}

/**
leader收到prepared消息，计算是否到达提交标准，向follow发送commited消息
与pbft的差异是只有leader能收到这种消息
*/
func (this *RaftConsensus) ReceivePrepared(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	// 本节点是主副节点都会收到
	logger.Infof("receive ReceivePrepared")
	messageLog, err := this.GetConsensusLog(chainMessage)
	if err != nil {
		return nil, err
	}
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
	if primaryPeerId != myPeerId {
		logger.Errorf("%v", messageLog)
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
	 * 通过检查ConsensusLog日志判断是否所有的节点包括自己都处于prepared状态
	 * 也就是说本节点已经知道所有的节点都发出了prepared通知 如果是则本节点处于prepared certificate状态，
	 * 最终，每个节点都会收到其他节点的prepared状态通知
	 */
	key = this.GetDataBlockCacheKey(messageLog.BlockId, messageLog.SliceNumber)
	var dataBlock *entity.DataBlock
	d, ok := MemCache.Get(key)
	if ok {
		dataBlock = d.(*entity.DataBlock)
	}
	peerIds := strings.Split(dataBlock.PeerIds, ",")
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
		logger.Infof("findCountBy current status:%v;count:%v", msgtype.CONSENSUS_PREPARED, count)
		// 收到足够的数目
		if count > 2*f {
			log.PeerId = myPeerId
			log.Status = msgtype.CONSENSUS_COMMITED
			log.Address = myselfPeer.Address
			log.PublicKey = myselfPeer.PublicKey
			MemCache.SetDefault(key, log)
			for _, id := range peerIds {
				if myPeerId == id {
					continue
				}
				go action.ConsensusAction.ConsensusLog(id, msgtype.CONSENSUS_COMMITED, log, "")
			}
			//保存dataBlock
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
func (this *RaftConsensus) ReceiveCommited(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	// 本节点是主副节点都会收到
	logger.Infof("receive ReceiveCommited")
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
	 * 异步返回leader reply
	 */
	log.Status = msgtype.CONSENSUS_REPLY
	if dataBlock.PeerId != myPeerId {
		//go service.GetPeerEndpointService().modifyBadCount(-1)
		log.PublicKey = myselfPeer.PublicKey
		log.Address = myselfPeer.Address
		log.ClientPeerId = messageLog.ClientPeerId
		go action.ConsensusAction.ConsensusLog(dataBlock.PrimaryPeerId, msgtype.CONSENSUS_REPLY, log, "")
	} else {
		logger.Warnf("SameSrcAndTargetPeer")
	}

	return nil, nil
}

/**
leader收到reply消息，判断完成后向客户就返回reply消息
与pbft的差异是pbft在上一步就完成了向客户端发送reply
*/
func (this *RaftConsensus) ReceiveReply(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	// 本节点是主副节点都会收到
	logger.Infof("receive ReceivePrepared")
	messageLog, err := this.GetConsensusLog(chainMessage)
	if err != nil {
		return nil, err
	}
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
	if primaryPeerId != myPeerId {
		logger.Errorf("%v", messageLog)
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
	 * 通过检查ConsensusLog日志判断是否所有的节点包括自己都处于prepared状态
	 * 也就是说本节点已经知道所有的节点都发出了prepared通知 如果是则本节点处于prepared certificate状态，
	 * 最终，每个节点都会收到其他节点的prepared状态通知
	 */
	key = this.GetDataBlockCacheKey(messageLog.BlockId, messageLog.SliceNumber)
	var dataBlock *entity.DataBlock
	d, ok := MemCache.Get(key)
	if ok {
		dataBlock = d.(*entity.DataBlock)
	}
	peerIds := strings.Split(dataBlock.PeerIds, ",")
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
		logger.Infof("findCountBy current status:%v;count:%v", msgtype.CONSENSUS_PREPARED, count)
		// 收到足够的数目
		if count > 2*f {
			log.PeerId = myPeerId
			log.Status = msgtype.CONSENSUS_COMMITED
			log.Address = myselfPeer.Address
			log.PublicKey = myselfPeer.PublicKey
			MemCache.SetDefault(key, log)
			go action.ConsensusAction.ConsensusLog(dataBlock.PeerId, msgtype.CONSENSUS_COMMITED, log, "")
			//保存dataBlock
		}
	} else {
		logger.Errorf("LessPeerLocation")
		return nil, errors.New("LessPeerLocation")
	}

	return nil, nil
}

func init() {
	raftConsensus = &RaftConsensus{}
	raftConsensus.MemCache = MemCache
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_RAFT, action.ConsensusAction.Send, raftConsensus.ReceiveConsensus, action.ConsensusAction.Response)
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_RAFT_PREPREPARED, action.ConsensusAction.Send, raftConsensus.ReceivePreprepared, action.ConsensusAction.Response)
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_RAFT_PREPARED, action.ConsensusAction.Send, raftConsensus.ReceivePrepared, action.ConsensusAction.Response)
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_RAFT_COMMITED, action.ConsensusAction.Send, raftConsensus.ReceiveCommited, action.ConsensusAction.Response)
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_RAFT_REPLY, action.ConsensusAction.Send, raftConsensus.ReceiveCommited, action.ConsensusAction.Response)
}
