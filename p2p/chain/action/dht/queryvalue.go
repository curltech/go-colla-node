package dht

import (
	"errors"
	"fmt"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	entity2 "github.com/curltech/go-colla-node/p2p/chain/entity"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	service1 "github.com/curltech/go-colla-node/p2p/chain/service"
	dhtentity "github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/msg/entity"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

type queryValueAction struct {
	action.BaseAction
}

var QueryValueAction queryValueAction

/**
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (this *queryValueAction) Receive(chainMessage *entity.ChainMessage) (*entity.ChainMessage, error) {
	logger.Sugar.Infof("Receive %v message", this.MsgType)
	var response *entity.ChainMessage = nil
	conditionBean, ok := chainMessage.Payload.(map[string]interface{})
	if !ok {
		response = handler.Error(chainMessage.MessageType, errors.New("ErrorCondition"))
		return response, nil
	}
	var getAllBlockIndex bool = false
	if conditionBean["getAllBlockIndex"] != nil {
		getAllBlockIndex = conditionBean["getAllBlockIndex"].(bool)
	}

	dataBlocks := make([]*entity2.DataBlock, 0)
	if getAllBlockIndex == true {
		ptMap := make(map[string]*entity2.PeerTransaction, 0)
		var blockType, createPeerId, receiverPeerId, businessNumber, parentBusinessNumber string
		if conditionBean["blockType"] != nil {
			blockType = conditionBean["blockType"].(string)
		}
		if len(blockType) == 0 {
			response = handler.Error(chainMessage.MessageType, errors.New("NullBlockType"))
			return response, nil
		}
		var key, keyKind string
		if blockType == entity2.BlockType_Collection {
			if conditionBean["createPeerId"] != nil {
				createPeerId = conditionBean["createPeerId"].(string)
			}
			if len(createPeerId) == 0 {
				response = handler.Error(chainMessage.MessageType, errors.New("NullCreatePeerId"))
				return response, nil
			}
			key = ns.GetPeerTransactionSrcKey(createPeerId)
			keyKind = ns.PeerTransaction_Src_KeyKind
		} else if blockType == entity2.BlockType_P2pChat {
			if conditionBean["businessNumber"] != nil {
				businessNumber = conditionBean["businessNumber"].(string)
			}
			if len(businessNumber) == 0 {
				response = handler.Error(chainMessage.MessageType, errors.New("NullBusinessNumber"))
				return response, nil
			}
			key = ns.GetPeerTransactionP2pChatKey(businessNumber)
			keyKind = ns.PeerTransaction_P2PChat_KeyKind
		} else if blockType == entity2.BlockType_GroupFile {
			if conditionBean["businessNumber"] != nil {
				businessNumber = conditionBean["businessNumber"].(string)
			}
			if len(businessNumber) == 0 {
				response = handler.Error(chainMessage.MessageType, errors.New("NullBusinessNumber"))
				return response, nil
			}
			key = ns.GetPeerTransactionGroupFileKey(businessNumber)
			keyKind = ns.PeerTransaction_GroupFile_KeyKind
		} else if blockType == entity2.BlockType_Channel {
			/*if conditionBean["businessNumber"] != nil {
				businessNumber = conditionBean["businessNumber"].(string)
			}
			if len(businessNumber) == 0 {
				response = handler.Error(chainMessage.MessageType, errors.New("NullBusinessNumber"))
				return response, nil
			}
			key = ns.GetPeerTransactionChannelKey(businessNumber)*/
			key = ns.GetPeerTransactionChannelKey(fmt.Sprintf("%v-%v", dhtentity.TransactionType_DataBlock, entity2.BlockType_Channel))
			keyKind = ns.PeerTransaction_Channel_KeyKind
		} else if blockType == entity2.BlockType_ChannelArticle {
			if conditionBean["parentBusinessNumber"] != nil {
				parentBusinessNumber = conditionBean["parentBusinessNumber"].(string)
			}
			if len(parentBusinessNumber) == 0 {
				response = handler.Error(chainMessage.MessageType, errors.New("NullParentBusinessNumber"))
				return response, nil
			}
			key = ns.GetPeerTransactionChannelArticleKey(parentBusinessNumber)
			keyKind = ns.PeerTransaction_ChannelArticle_KeyKind
		}
		if config.Libp2pParams.FaultTolerantLevel == 0 {
			recvdVals, err := dht.PeerEndpointDHT.GetValues(key)
			if err != nil {
				response = handler.Error(chainMessage.MessageType, err)
				return response, nil
			}
			for _, recvdVal := range recvdVals {
				pts := make([]*entity2.PeerTransaction, 0)
				err = message.TextUnmarshal(string(recvdVal), &pts)
				if err != nil {
					response = handler.Error(chainMessage.MessageType, err)
					return response, nil
				}
				for _, pt := range pts {
					if ptMap[pt.BlockId] == nil {
						ptMap[pt.BlockId] = pt
					}
				}
			}
		} else if config.Libp2pParams.FaultTolerantLevel == 1 {

		} else if config.Libp2pParams.FaultTolerantLevel == 2 {
			// 查询删除local记录
			locals, err := service1.GetPeerTransactionService().GetLocalPTs(keyKind, createPeerId, receiverPeerId, businessNumber, parentBusinessNumber)
			if err != nil {
				response = handler.Error(chainMessage.MessageType, err)
				return response, nil
			}
			if len(locals) > 0 {
				for _, local := range locals {
					if ptMap[local.BlockId] == nil {
						ptMap[local.BlockId] = local
					}
				}
				service1.GetPeerTransactionService().Delete(locals, "")
			}
			// 查询non-local记录
			recvdVals, err := dht.PeerEndpointDHT.GetValues(key)
			if err != nil {
				response = handler.Error(chainMessage.MessageType, err)
				return response, nil
			}
			// 恢复local记录
			err = service1.GetPeerTransactionService().PutLocalPTs(keyKind, locals)
			if err != nil {
				response = handler.Error(chainMessage.MessageType, err)
				return response, nil
			}
			// 整合记录
			for _, recvdVal := range recvdVals {
				pts := make([]*entity2.PeerTransaction, 0)
				err = message.TextUnmarshal(string(recvdVal), &pts)
				if err != nil {
					response = handler.Error(chainMessage.MessageType, err)
					return response, nil
				}
				for _, pt := range pts {
					if ptMap[pt.BlockId] == nil {
						ptMap[pt.BlockId] = pt
					}
				}
			}
		}
		for _, v := range ptMap {
			db := entity2.DataBlock{}
			db.BlockId = v.BlockId
			db.ParentBusinessNumber = v.ParentBusinessNumber
			db.BusinessNumber = v.BusinessNumber
			db.CreateTimestamp = v.CreateTimestamp
			db.PrimaryPeerId = v.PrimaryPeerId
			db.Metadata = v.Metadata
			db.Thumbnail = v.Thumbnail
			db.Name = v.Name
			db.Description = v.Description
			db.PeerId = v.SrcPeerId
			dataBlocks = append(dataBlocks, &db)
		}
	} else {
		var blockId string = ""
		if conditionBean["blockId"] != nil {
			blockId = conditionBean["blockId"].(string)
		}
		if len(blockId) == 0 {
			response = handler.Error(chainMessage.MessageType, errors.New("NullBlockId"))
			return response, nil
		}
		var sliceNumber uint64 = 0
		if conditionBean["sliceNumber"] != nil {
			sliceNumber = uint64(conditionBean["sliceNumber"].(float64))
		}
		var receiverPeer bool = false
		if conditionBean["receiverPeer"] != nil {
			receiverPeer = conditionBean["receiverPeer"].(bool)
		}
		var receiverPeerId string = ""
		if receiverPeer == true {
			if conditionBean["receiverPeerId"] != nil {
				receiverPeerId = conditionBean["receiverPeerId"].(string)
			}
			if len(receiverPeerId) == 0 {
				response = handler.Error(chainMessage.MessageType, errors.New("NullReceiverPeerId"))
				return response, nil
			}
		}
		//limit := conditionBean["limit"].(uint64)
		//service1.GetDataBlockService().QueryValue(&dataBlocks, blockId, sliceNumber)
		key := ns.GetDataBlockKey(blockId)
		if config.Libp2pParams.FaultTolerantLevel == 0 {
			recvdVals, err := dht.PeerEndpointDHT.GetValues(key)
			if err != nil {
				response = handler.Error(chainMessage.MessageType, err)
				return response, nil
			}
			for _, recvdVal := range recvdVals {
				dbs := make([]*entity2.DataBlock, 0)
				err = message.TextUnmarshal(string(recvdVal), &dbs)
				if err != nil {
					response = handler.Error(chainMessage.MessageType, err)
					return response, nil
				}
				for _, db := range dbs {
					var receivable bool
					if len(receiverPeerId) > 0 {
						if sliceNumber != 1 || len(db.TransactionKeys) == 0 {
							receivable = true
						} else {
							receivable = false
							for _, transactionKey := range db.TransactionKeys {
								if transactionKey.PeerId == receiverPeerId {
									receivable = true
									break
								}
							}
						}
					}
					if ((len(receiverPeerId) == 0 && len(db.PayloadKey) == 0) || (len(receiverPeerId) > 0 && receivable == true)) &&
						(sliceNumber > 0 && db.SliceNumber == uint64(sliceNumber)) {
						dataBlocks = append(dataBlocks, db)
					}
				}
			}
		} else if config.Libp2pParams.FaultTolerantLevel == 1 {

		} else if config.Libp2pParams.FaultTolerantLevel == 2 {
			// 查询删除local记录
			locals, err := service1.GetDataBlockService().GetLocalDBs(ns.DataBlock_KeyKind, "", blockId, receiverPeerId, 0)
			if err != nil {
				response = handler.Error(chainMessage.MessageType, err)
				return response, nil
			}
			if len(locals) > 0 {
				service1.GetDataBlockService().Delete(locals, "")
			}
			// 查询non-local记录
			recvdVals, err := dht.PeerEndpointDHT.GetValues(key)
			if err != nil {
				response = handler.Error(chainMessage.MessageType, err)
				return response, nil
			}
			// 恢复local记录
			err = service1.GetDataBlockService().PutLocalDBs(ns.DataBlock_KeyKind, locals)
			if err != nil {
				response = handler.Error(chainMessage.MessageType, err)
				return response, nil
			}
			// 更新local记录
			for _, recvdVal := range recvdVals {
				dbs := make([]*entity2.DataBlock, 0)
				err = message.TextUnmarshal(string(recvdVal), &dbs)
				if err != nil {
					response = handler.Error(chainMessage.MessageType, err)
					return response, nil
				}
				err = service1.GetDataBlockService().PutLocalDBs(ns.DataBlock_KeyKind, dbs)
				if err != nil {
					response = handler.Error(chainMessage.MessageType, err)
					return response, nil
				}
			}
			// 再次查询local记录
			dbs, err := service1.GetDataBlockService().GetLocalDBs(ns.DataBlock_KeyKind, "", blockId, receiverPeerId, sliceNumber)
			if err != nil {
				response = handler.Error(chainMessage.MessageType, err)
				return response, nil
			}
			if len(dbs) > 0 {
				for _, db := range dbs {
					var receivable bool
					if len(receiverPeerId) > 0 {
						if sliceNumber != 1 {
							receivable = true
						} else {
							receivable = false
							for _, transactionKey := range db.TransactionKeys {
								if transactionKey.PeerId == receiverPeerId {
									receivable = true
									break
								}
							}
						}
					}
					if ((len(receiverPeerId) == 0 && len(db.PayloadKey) == 0) || (len(receiverPeerId) > 0 && receivable == true)) &&
						(sliceNumber > 0 && db.SliceNumber == uint64(sliceNumber)) {
						dataBlocks = append(dataBlocks, db)
					}
				}
			}
		}
	}
	response = handler.Response(chainMessage.MessageType, dataBlocks)
	response.PayloadType = handler.PayloadType_DataBlock

	return response, nil
}

func init() {
	QueryValueAction = queryValueAction{}
	QueryValueAction.MsgType = msgtype.QUERYVALUE
	handler.RegistChainMessageHandler(msgtype.QUERYVALUE, QueryValueAction.Send, QueryValueAction.Receive, QueryValueAction.Response)
}
