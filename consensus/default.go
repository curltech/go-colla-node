package consensus

import (
	"errors"
	"fmt"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/p2p/chain/entity"
	service2 "github.com/curltech/go-colla-node/p2p/chain/service"
	entity1 "github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/libp2p/go-libp2p-core/peer"
	kb "github.com/libp2p/go-libp2p-kbucket"
	"github.com/patrickmn/go-cache"
	"time"
)

type Consensus struct {
	MemCache *cache.Cache
}

func (this *Consensus) GetLogCacheKey(log *entity.ConsensusLog) string {
	key := fmt.Sprintf("%v:%v:%v:%v:%v:%v", log.PrimaryPeerId, log.BlockId, log.SliceNumber, log.PrimarySequenceId, log.PeerId, log.Status)
	return key
}

func (this *Consensus) GetDataBlockCacheKey(blockId string, sliceNumber uint64) string {
	key := fmt.Sprintf("%v:%v", blockId, sliceNumber)
	return key
}

func (this *Consensus) NearestConsensusPeer() []peer.ID {
	id := kb.ConvertKey(global.Global.PeerId.String())
	ids := dht.PeerEndpointDHT.RoutingTable.NearestPeers(id, 10)

	return ids
}

/**
主节点挑选副节点
*/
func (this *Consensus) ChooseConsensusPeer() []string {
	peerIds := make([]string, 0)
	peerEndpoints := service.GetPeerEndpointService().GetRand(10)
	for _, consensusPeer := range peerEndpoints {
		peerIds = append(peerIds, consensusPeer.PeerId)
	}

	return peerIds
}

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
	service2.GetConsensusLogService().Insert(log)

	return log
}

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
