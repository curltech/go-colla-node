package std

import (
	"errors"
	"github.com/curltech/go-colla-core/cache"
	"github.com/curltech/go-colla-core/config"
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

/**
主节点保存数据，根据配置随机选择副节点，发送数据到副节点保存，成功的节点返回
*/
var MemCache = cache.NewMemCache("std", 0, 0)

type StdConsensus struct {
	consensus.Consensus
}

var stdConsensus *StdConsensus

func GetStdConsensus() *StdConsensus {

	return stdConsensus
}

/**
 * primary 收到消息请求，向每个follow发送commited消息
 *
 * @param chainMessage
 * @return
 */
func (this *StdConsensus) ReceiveConsensus(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
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
	} else {
		if dataBlock.PrimaryPeerId != myselfPeer.PeerId {
			return nil, errors.New("MustPrimaryPeer")
		}
	}

	var peerIds []string
	if config.ConsensusParams.StdMinPeerNum > 0 {
		peerIds = this.ChooseConsensusPeer()
	}
	if peerIds != nil && len(peerIds) > 0 {
		/**
		 * 交易校验通过，主节点进入预准备状态，记录日志
		 */
		log := this.CreateConsensusLog(chainMessage, dataBlock, myselfPeer, msgtype.CONSENSUS_PREPARED)
		logkey := this.GetLogCacheKey(log)
		MemCache.SetDefault(logkey, log)
		datakey := this.GetDataBlockCacheKey(dataBlock.BlockId, dataBlock.SliceNumber)
		MemCache.SetDefault(datakey, dataBlock)
		log.PeerIds = strings.Join(peerIds, ",")
		/**
		 * 发送CONSENSUS_COMMITED给副节点，告知主节点的状态
		 */
		for _, peerId := range peerIds {
			if myselfPeer.PeerId == peerId {
				continue
			}
			// 封装消息，异步发送
			go action.ConsensusAction.ConsensusDataBlock(peerId, msgtype.CONSENSUS_COMMITED, dataBlock, "")
		}
		response := handler.Response(msgtype.CONSENSUS_PREPARED, msgtype.RESPONSE)

		return response, nil
	} else {
		service2.GetDataBlockService().Insert(dataBlock)
		response := handler.Ok(msgtype.CONSENSUS_REPLY)

		return response, nil
	}

	return nil, nil
}

/**
 * vice收到commited消息，完成后，向primary发送reply消息
 */
func (this *StdConsensus) ReceiveCommited(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	// 本节点是主副节点都会收到
	logger.Infof("receive ReceiveCommited")
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

	log := this.CreateConsensusLog(chainMessage, dataBlock, myselfPeer, msgtype.CONSENSUS_COMMITED)

	/**
	 * 数据块记录有效
	 */
	//dataBlock = service2.GetDataBlockService().GetDataBlock(blockId, sliceNumber)
	if dataBlock != nil {
		dataBlock.Status = entity2.EntityStatus_Effective
		service2.GetDataBlockService().Insert(dataBlock)
	}

	/**
	 * 异步返回leader reply
	 */
	go action.ConsensusAction.ConsensusLog(dataBlock.PrimaryPeerId, msgtype.CONSENSUS_REPLY, log, "")

	return nil, nil
}

/**
leader收到reply消息，判断完成后向客户就返回reply消息
与pbft的差异是pbft在上一步就完成了向客户端发送reply
*/
func (this *StdConsensus) ReceiveReply(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
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
		messageLog.Status = msgtype.CONSENSUS_REPLY
		MemCache.SetDefault(key, messageLog)
	}
	/**
	 */
	key = this.GetDataBlockCacheKey(messageLog.BlockId, messageLog.SliceNumber)
	var dataBlock *entity.DataBlock
	d, ok := MemCache.Get(key)
	if ok {
		dataBlock = d.(*entity.DataBlock)
	}
	peerIds := strings.Split(dataBlock.PeerIds, ",")
	if peerIds != nil && len(peerIds) > 0 {
		log := &entity.ConsensusLog{}
		log.PrimaryPeerId = primaryPeerId
		log.BlockId = messageLog.BlockId
		log.SliceNumber = messageLog.SliceNumber
		log.PrimarySequenceId = messageLog.PrimarySequenceId
		log.PeerId = peerId
		log.Status = msgtype.CONSENSUS_REPLY
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
		logger.Infof("findCountBy current status:%v;count:%v", msgtype.CONSENSUS_REPLY, count)
		// 收到足够的数目
		if count > config.ConsensusParams.StdMinPeerNum {
			//保存dataBlock
			service2.GetDataBlockService().Insert(dataBlock)
			log.PeerId = myPeerId
			log.Status = msgtype.CONSENSUS_REPLY
			go action.ConsensusAction.ConsensusLog(dataBlock.PeerId, msgtype.CONSENSUS_REPLY, log, "")
		}
	}

	return nil, nil
}

func init() {
	stdConsensus = &StdConsensus{}
	stdConsensus.MemCache = MemCache
	handler.RegistChainMessageHandler(msgtype.CONSENSUS, action.ConsensusAction.Send, stdConsensus.ReceiveConsensus, action.ConsensusAction.Response)
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_COMMITED, action.ConsensusAction.Send, stdConsensus.ReceiveCommited, action.ConsensusAction.Response)
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_REPLY, action.ConsensusAction.Send, stdConsensus.ReceiveReply, action.ConsensusAction.Response)
}
