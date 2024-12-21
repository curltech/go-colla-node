package consensus

import (
	"errors"
	"fmt"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/crypto/std"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/p2p/chain/entity"
	service2 "github.com/curltech/go-colla-node/p2p/chain/service"
	entity1 "github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	entity2 "github.com/curltech/go-colla-node/p2p/msg/entity"
	kb "github.com/libp2p/go-libp2p-kbucket"
	"github.com/patrickmn/go-cache"
	"math/rand"
	"time"
	"unsafe"
)

type Consensus struct {
	MemCache *cache.Cache
}

/*
*
获取日志缓存的key
*/
func (this *Consensus) GetLogCacheKey(log *entity.ConsensusLog) string {
	key := fmt.Sprintf("%v:%v:%v:%v:%v:%v", log.PrimaryPeerId, log.BlockId, log.SliceNumber, log.PrimarySequenceId, log.PeerId, log.Status)
	return key
}

/*
*
获取datablock缓存的key
*/
func (this *Consensus) GetDataBlockCacheKey(blockId string, sliceNumber uint64) string {
	key := fmt.Sprintf("%v:%v", blockId, sliceNumber)
	return key
}

// origin为原数组，count为随机取出的个数，最终返回一个count容量的目标数组
func randomSlice(origin []string, count int, seed int64) []string {
	tmpOrigin := make([]string, len(origin))
	copy(tmpOrigin, origin)
	rand.Seed(seed)
	rand.Shuffle(len(tmpOrigin), func(i int, j int) {
		tmpOrigin[i], tmpOrigin[j] = tmpOrigin[j], tmpOrigin[i]
	})

	result := make([]string, 0, count)
	for index, value := range tmpOrigin {
		if index == count {
			break
		}
		result = append(result, value)
	}
	return result
}

/*
*
获取最近的peer集合
*/
func (this *Consensus) NearestConsensusPeer(key string, createTimestamp uint64) []string {
	if config.ConsensusParams.PeerNum == 0 {
		return nil
	}
	peerIds := make([]string, 0)
	id := kb.ConvertKey(key)
	//ids := dht.PeerEndpointDHT.RoutingTable.NearestPeers(id, config.ConsensusParams.PeerRange)
	bucketSize, _ := config.GetInt("p2p.dht.bucketSize", 20)
	ids := dht.PeerEndpointDHT.RoutingTable.NearestPeers(id, bucketSize)
	if len(ids) > 0 {
		for _, id := range ids {
			peerIds = append(peerIds, id.String())
		}
	}
	if len(ids) > config.ConsensusParams.PeerNum {
		return randomSlice(peerIds, config.ConsensusParams.PeerNum, int64(createTimestamp))
	} else {
		return peerIds
	}
}

/*
*
主节点挑选副节点
*/
func (this *Consensus) ChooseConsensusPeer(dataBlock *entity.DataBlock) []string {
	if config.ConsensusParams.PeerNum == 0 {
		return nil
	}
	peerIds := make([]string, 0)
	seed := int64(dataBlock.CreateTimestamp)
	peerEndpoints := service.GetPeerEndpointService().GetRand(seed)
	for _, consensusPeer := range peerEndpoints {
		peerIds = append(peerIds, consensusPeer.PeerId)
	}

	return peerIds
}

/*
*
创建新的ConsensusLog，存入数据库中
*/
func (this *Consensus) CreateConsensusLog(chainMessage *entity2.ChainMessage, dataBlock *entity.DataBlock, myselfPeer *entity1.MyselfPeer, status string) *entity.ConsensusLog {
	log := &entity.ConsensusLog{}
	log.BlockId = dataBlock.BlockId
	log.BlockType = dataBlock.BlockType
	log.SliceNumber = dataBlock.SliceNumber
	log.PrimarySequenceId = dataBlock.PrimarySequenceId
	log.PayloadHash = dataBlock.PayloadHash
	log.ClientPeerId = chainMessage.SrcPeerId
	log.ClientAddress = chainMessage.SrcConnectAddress
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

/*
*
从消息中提取datablock
*/
func (this *Consensus) GetDataBlock(chainMessage *entity2.ChainMessage) (*entity.DataBlock, error) {
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
	// 只针对第一个分片处理一次
	if dataBlock.SliceNumber == 1 && dataBlock.TransactionKeys == nil {
		if len(dataBlock.TransportKey) > 0 {
			transportKey := std.DecodeBase64(dataBlock.TransportKey)
			transactionKeys := make([]*entity.TransactionKey, 0)
			err := message.TextUnmarshal(*(*string)(unsafe.Pointer(&transportKey)), &transactionKeys)
			if err != nil {
				return nil, errors.New("TransactionKeysTextUnmarshalFailure")
			}
			dataBlock.TransactionKeys = transactionKeys
		}
	}
	if dataBlock.TransportPayload != "" && dataBlock.TransactionAmount == 0 {
		transportPayload := std.DecodeBase64(dataBlock.TransportPayload)
		dataBlock.TransactionAmount = service2.GetDataBlockService().GetTransactionAmount(transportPayload)
	}
	return dataBlock, nil
}

/*
*
从消息中提取ConsensusLog
*/
func (this *Consensus) GetConsensusLog(chainMessage *entity2.ChainMessage) (*entity.ConsensusLog, error) {
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
