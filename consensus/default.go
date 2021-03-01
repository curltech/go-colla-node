package consensus

import (
	"errors"
	"fmt"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/p2p/chain/entity"
	entity1 "github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	"github.com/curltech/go-colla-node/p2p/msg"
	kb "github.com/libp2p/go-libp2p-kbucket"
	"github.com/patrickmn/go-cache"
	"golang.org/x/exp/rand"
	"time"
)

type Consensus struct {
	MemCache *cache.Cache
}

/**
获取日志缓存的key
*/
func (this *Consensus) GetLogCacheKey(log *entity.ConsensusLog) string {
	key := fmt.Sprintf("%v:%v:%v:%v:%v:%v", log.PrimaryPeerId, log.BlockId, log.SliceNumber, log.PrimarySequenceId, log.PeerId, log.Status)
	return key
}

/**
获取datablock缓存的key
*/
func (this *Consensus) GetDataBlockCacheKey(blockId string, sliceNumber uint64) string {
	key := fmt.Sprintf("%v:%v", blockId, sliceNumber)
	return key
}

/**
获取最近的peer集合
*/
func (this *Consensus) NearestConsensusPeer(key string) []string {
	if config.ConsensusParams.PeerNum == 0 {
		return nil
	}
	peerIds := make([]string, 0)
	id := kb.ConvertKey(key)
	ids := dht.PeerEndpointDHT.RoutingTable.NearestPeers(id, config.ConsensusParams.PeerRange)
	for i := 0; i < config.ConsensusParams.PeerNum; i++ {
		r := rand.Intn(len(ids))
		peerIds = append(peerIds, ids[r].Pretty())
	}

	return peerIds
}

/**
主节点挑选副节点
*/
func (this *Consensus) ChooseConsensusPeer() []string {
	if config.ConsensusParams.PeerNum == 0 {
		return nil
	}
	peerIds := make([]string, 0)
	peerEndpoints := service.GetPeerEndpointService().GetRand(config.ConsensusParams.PeerNum)
	for _, consensusPeer := range peerEndpoints {
		peerIds = append(peerIds, consensusPeer.PeerId)
	}

	return peerIds
}

/**
创建新的ConsensusLog，存入数据库中
*/
func (this *Consensus) CreateConsensusLog(chainMessage *msg.ChainMessage, dataBlock *entity.DataBlock, myselfPeer *entity1.MyselfPeer, status string) *entity.ConsensusLog {
	log := &entity.ConsensusLog{}
	log.BlockId = dataBlock.BlockId
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
	log.Status = status
	t := time.Now()
	log.StatusDate = &t
	log.CreateDate = &t
	log.TransactionAmount = dataBlock.TransactionAmount

	return log
}

/**
从消息中提取datablock
*/
func (this *Consensus) GetDataBlock(chainMessage *msg.ChainMessage) (*entity.DataBlock, error) {
	var dataBlock *entity.DataBlock
	if chainMessage.Payload != nil {
		var ok bool
		dataBlock, ok = chainMessage.Payload.(*entity.DataBlock)
		if !ok {
			return nil, errors.New("NotDataBlock")
		}
	}
	if dataBlock == nil {
		return nil, errors.New("NoPayload")
	}
	return dataBlock, nil
}

/**
从消息中提取ConsensusLog
*/
func (this *Consensus) GetConsensusLog(chainMessage *msg.ChainMessage) (*entity.ConsensusLog, error) {
	var messageLog *entity.ConsensusLog
	if chainMessage.Payload != nil {
		var ok bool
		messageLog, ok = chainMessage.Payload.(*entity.ConsensusLog)
		if !ok {
			return nil, errors.New("NotPbftConsensusLog")
		}
	}
	if messageLog == nil {
		return nil, errors.New("NoPayload")
	}
	return messageLog, nil
}
