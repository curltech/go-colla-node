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
	"time"
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
	logger.Sugar.Infof("ReceiveConsensus")
	var response *msg.ChainMessage
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

	if dataBlock.BlockType != entity.BlockType_ChatAttach {
		var peerIds []string
		if config.ConsensusParams.StdMinPeerNum > 0 {
			//peerIds = this.ChooseConsensusPeer(dataBlock)
			peerIds = this.NearestConsensusPeer(dataBlock.BlockId, dataBlock.CreateTimestamp)
		}
		if peerIds != nil && len(peerIds) > 0 {
			/**
			 * 交易校验通过，主节点进入准备状态，记录日志
			 */
			//log := this.CreateConsensusLog(chainMessage, dataBlock, myselfPeer, msgtype.CONSENSUS_PREPARED)
			//log.PeerIds = strings.Join(peerIds, ",")
			//service2.GetConsensusLogService().Insert(log)
			//logkey := this.GetLogCacheKey(log)
			//MemCache.SetDefault(logkey, log)
			dataBlock.PeerIds = strings.Join(peerIds, ",")
			datakey := this.GetDataBlockCacheKey(dataBlock.BlockId, dataBlock.SliceNumber)
			MemCache.SetDefault(datakey, dataBlock)
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
			response = handler.Ok(msgtype.CONSENSUS)

			return response, nil
		}
	}
	// 保存dataBlock
	dataBlock.Status = entity2.EntityStatus_Effective
	start := time.Now()
	err = service2.GetDataBlockService().StoreValue(dataBlock, true)
	end := time.Now()
	logger.Sugar.Infof("ReceiveConsensus StoreValue time:%v", end.Sub(start))
	if err != nil {
		logger.Sugar.Errorf("ReceiveConsensus StoreValue failed:%v", err)
		response = handler.Error(msgtype.CONSENSUS_REPLY, err)
	} else {
		response = handler.Ok(msgtype.CONSENSUS_REPLY)
	}

	return response, nil
}

/**
 * vice收到commited消息，完成后，向primary发送reply消息
 */
func (this *StdConsensus) ReceiveCommited(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	// 本节点是副节点会收到
	logger.Sugar.Infof("receive ReceiveCommited")
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

	log := this.CreateConsensusLog(chainMessage, dataBlock, myselfPeer, msgtype.CONSENSUS_REPLY)
	//service2.GetConsensusLogService().Insert(log)

	// 保存dataBlock
	dataBlock.Status = entity2.EntityStatus_Effective
	start := time.Now()
	err = service2.GetDataBlockService().StoreValue(dataBlock, false)
	end := time.Now()
	logger.Sugar.Infof("ReceiveCommited StoreValue time:%v", end.Sub(start))
	if err != nil {
		logger.Sugar.Errorf("ReceiveCommited StoreValue failed:%v", err)
		return nil, err
	} else {
		/**
		 * 异步返回leader reply
		 */
		go action.ConsensusAction.ConsensusLog(dataBlock.PrimaryPeerId, msgtype.CONSENSUS_REPLY, log, "")
	}

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
	* 通过检查ConsensusLogEO日志判断是否该接受还是拒绝
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
	if primaryPeerId != myPeerId {
		logger.Sugar.Errorf("%v", messageLog)
		return nil, errors.New("SendMyselfMessage")
	}
	key := this.GetLogCacheKey(messageLog)
	var cacheLog *entity.ConsensusLog
	l, found := MemCache.Get(key)
	if found {
		cacheLog = l.(*entity.ConsensusLog)
	}
	if cacheLog != nil {
		existPayloadHash := cacheLog.PayloadHash
		if payloadHash != existPayloadHash {
			//go service.GetPeerEndpointService().modifyBadCount()
			return nil, errors.New("ErrorPayloadHash")
		}
	} else {
		MemCache.SetDefault(key, messageLog)
	}
	key = this.GetDataBlockCacheKey(messageLog.BlockId, messageLog.SliceNumber)
	var dataBlock *entity.DataBlock
	d, ok := MemCache.Get(key)
	if ok {
		dataBlock = d.(*entity.DataBlock)
	}
	var peerIds []string
	if dataBlock != nil {
		peerIds = strings.Split(dataBlock.PeerIds, ",")
	}
	if peerIds != nil && len(peerIds) > 0 {
		log := &entity.ConsensusLog{}
		log.PrimaryPeerId = primaryPeerId
		log.BlockId = messageLog.BlockId
		log.SliceNumber = messageLog.SliceNumber
		log.PrimarySequenceId = messageLog.PrimarySequenceId
		log.Status = msgtype.CONSENSUS_REPLY
		keys := make([]string, 0)
		count := 1
		for _, id := range peerIds {
			log.PeerId = id
			key = this.GetLogCacheKey(log)
			l, found = MemCache.Get(key)
			cacheLog = nil
			if found {
				cacheLog = l.(*entity.ConsensusLog)
				keys = append(keys, key)
			}
			if cacheLog != nil {
				count++
			}
		}
		logger.Sugar.Infof("findCountBy current status:%v;count:%v", msgtype.CONSENSUS_REPLY, count)
		// 收到足够的数目
		if count == config.ConsensusParams.StdMinPeerNum + 1 {
			// 保存dataBlock
			dataBlock.Status = entity2.EntityStatus_Effective
			log.PeerId = myPeerId
			go finalCommit(dataBlock, log)
			/*for _, key := range keys {
				MemCache.Delete(key)
			}*/
		}
	}

	return nil, nil
}

func finalCommit(dataBlock *entity.DataBlock, log *entity.ConsensusLog) {
	start := time.Now()
	err := service2.GetDataBlockService().StoreValue(dataBlock, true)
	end := time.Now()
	logger.Sugar.Infof("finalCommit StoreValue time:%v", end.Sub(start))
	if err != nil {
		logger.Sugar.Errorf("finalCommit StoreValue failed:%v", err)
	} else {
		go action.ConsensusAction.ConsensusLog(dataBlock.PeerId, msgtype.CONSENSUS_REPLY, log, "")
	}
}

func init() {
	stdConsensus = &StdConsensus{}
	stdConsensus.MemCache = MemCache
	handler.RegistChainMessageHandler(msgtype.CONSENSUS, action.ConsensusAction.Send, stdConsensus.ReceiveConsensus, action.ConsensusAction.Response)
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_COMMITED, action.ConsensusAction.Send, stdConsensus.ReceiveCommited, action.ConsensusAction.Response)
	handler.RegistChainMessageHandler(msgtype.CONSENSUS_REPLY, action.ConsensusAction.Send, stdConsensus.ReceiveReply, action.ConsensusAction.Response)
}
