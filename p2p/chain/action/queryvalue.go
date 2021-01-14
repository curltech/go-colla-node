package action

import (
	"errors"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/ns"
	entity2 "github.com/curltech/go-colla-node/p2p/chain/entity"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	service1 "github.com/curltech/go-colla-node/p2p/chain/service"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

type queryValueAction struct {
	BaseAction
}

var QueryValueAction queryValueAction

/**
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (this *queryValueAction) PCReceive(chainMessage *msg.PCChainMessage) (interface{}, error) {
	logger.Infof("Receive %v message", this.MsgType)
	conditionBean := chainMessage.MessagePayload.Payload.(map[string]interface{})
	var getAllBlockIndex bool = false
	if conditionBean["getAllBlockIndex"] != nil {
		getAllBlockIndex = conditionBean["getAllBlockIndex"].(bool)
	}
	dataBlocks := make([]*entity2.DataBlock, 0)

	if getAllBlockIndex == true {
		ptMap := make(map[string]*entity2.PeerTransaction, 0)
		var createPeerId string = ""
		if conditionBean["createPeerId"] != nil {
			createPeerId = conditionBean["createPeerId"].(string)
		}
		if len(createPeerId) == 0 {
			return msgtype.ERROR, errors.New("NullCreatePeerId")
		}
		key := ns.GetPeerTransactionSrcKey(createPeerId)
		if config.Libp2pParams.FaultTolerantLevel == 0 {
			recvdVals, err := dht.PeerEndpointDHT.GetValues(key, config.Libp2pParams.Nvals)
			if err != nil {
				return msgtype.ERROR, err
			}
			for _, recvdVal := range recvdVals {
				pts := make([]*entity2.PeerTransaction, 0)
				err = message.TextUnmarshal(string(recvdVal.Val), &pts)
				if err != nil {
					logger.Errorf("failed to TextUnmarshal PeerTransaction value: %v, err: %v", recvdVal.Val, err)
					return msgtype.ERROR, err
				}
				for _, pt := range pts {
					if ptMap[pt.BlockId] == nil {
						ptMap[pt.BlockId] = pt
					}
				}
			}
		} else if config.Libp2pParams.FaultTolerantLevel == 1 {
			// 查询删除local记录
			locals, err := service1.GetLocalPTs(ns.PeerTransaction_Src_KeyKind, createPeerId, "")
			if err != nil {
				return msgtype.ERROR, err
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
			recvdVals, err := dht.PeerEndpointDHT.GetValues(key, config.Libp2pParams.Nvals)
			if err != nil {
				return msgtype.ERROR, err
			}
			// 恢复local记录
			err = service1.PutLocalPTs(locals)
			if err != nil {
				return msgtype.ERROR, err
			}
			// 整合记录
			for _, recvdVal := range recvdVals {
				pts := make([]*entity2.PeerTransaction, 0)
				err = message.TextUnmarshal(string(recvdVal.Val), &pts)
				if err != nil {
					logger.Errorf("failed to TextUnmarshal PeerTransaction value: %v, err: %v", recvdVal.Val, err)
					return msgtype.ERROR, err
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
			db.BusinessNumber = v.BusinessNumber
			db.CreateTimestamp = v.CreateTimestamp
			dataBlocks = append(dataBlocks, &db)
		}
	} else {
		var blockId string = ""
		if conditionBean["blockId"] != nil {
			blockId = conditionBean["blockId"].(string)
		}
		if len(blockId) == 0 {
			return msgtype.ERROR, errors.New("NullBlockId")
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
				return msgtype.ERROR, errors.New("NullReceiverPeerId")
			}
		}
		var txSequenceId uint64 = 0
		if conditionBean["txSequenceId"] != nil {
			txSequenceId = uint64(conditionBean["txSequenceId"].(float64))
		}
		var sliceNumber uint64 = 0
		if conditionBean["sliceNumber"] != nil {
			sliceNumber = uint64(conditionBean["sliceNumber"].(float64))
		}
		//limit := conditionBean["limit"].(uint64)
		key := ns.GetDataBlockKey(blockId)
		if config.Libp2pParams.FaultTolerantLevel == 0 {
			recvdVals, err := dht.PeerEndpointDHT.GetValues(key, config.Libp2pParams.Nvals)
			if err != nil {
				return msgtype.ERROR, err
			}
			for _, recvdVal := range recvdVals {
				dbs := make([]*entity2.DataBlock, 0)
				err = message.TextUnmarshal(string(recvdVal.Val), &dbs)
				if err != nil {
					logger.Errorf("failed to TextUnmarshal DataBlock value: %v, err: %v", recvdVal.Val, err)
					return msgtype.ERROR, err
				}
				for _, db := range dbs {
					var receivable bool
					if len(receiverPeerId) > 0 {
						receivable = false
						for _, transactionKey := range db.TransactionKeys {
							if transactionKey.PeerId == receiverPeerId {
								receivable = true
								break
							}
						}
					}
					if ((len(receiverPeerId) == 0 && len(db.PayloadKey) == 0) || (len(receiverPeerId) > 0 && receivable == true)) &&
						(txSequenceId > 0 && db.TxSequenceId == uint64(txSequenceId)) &&
						(sliceNumber > 0 && db.SliceNumber == uint64(sliceNumber)) {
						dataBlocks = append(dataBlocks, db)
					}
				}
			}
		} else if config.Libp2pParams.FaultTolerantLevel == 1 {
			// 查询删除local记录
			locals, err := service1.GetLocalDBs(ns.DataBlock_KeyKind, "", blockId, receiverPeerId, 0, 0)
			if err != nil {
				return msgtype.ERROR, err
			}
			if len(locals) > 0 {
				service1.GetDataBlockService().Delete(locals, "")
			}
			// 查询non-local记录
			recvdVals, err := dht.PeerEndpointDHT.GetValues(key, config.Libp2pParams.Nvals)
			if err != nil {
				return msgtype.ERROR, err
			}
			// 恢复local记录
			err = service1.PutLocalDBs(locals)
			if err != nil {
				return msgtype.ERROR, err
			}
			// 更新local记录
			for _, recvdVal := range recvdVals {
				dbs := make([]*entity2.DataBlock, 0)
				err = message.TextUnmarshal(string(recvdVal.Val), &dbs)
				if err != nil {
					logger.Errorf("failed to TextUnmarshal DataBlock value: %v, err: %v", recvdVal.Val, err)
					return msgtype.ERROR, err
				}
				err = service1.PutLocalDBs(dbs)
				if err != nil {
					logger.Errorf("failed to PutLocalDBs DataBlock value: %v, err: %v", recvdVal.Val, err)
					return msgtype.ERROR, err
				}
			}
			// 再次查询local记录
			dbs, err := service1.GetLocalDBs(ns.DataBlock_KeyKind, "", blockId, receiverPeerId, txSequenceId, sliceNumber)
			if err != nil {
				return msgtype.ERROR, err
			}
			if len(dbs) > 0 {
				for _, db := range dbs {
					dataBlocks = append(dataBlocks, db)
				}
			}
		}
	}

	return dataBlocks, nil
}

/**
处理返回消息
*/
func (this *queryValueAction) PCResponse(chainMessage *msg.PCChainMessage) error {
	logger.Infof("Response %v message:%v", this.MsgType, chainMessage)

	return nil
}

func init() {
	QueryValueAction = queryValueAction{}
	QueryValueAction.MsgType = msgtype.QUERYVALUE
	handler.RegistChainMessageHandler(msgtype.QUERYVALUE, QueryValueAction.Send, QueryValueAction.Receive, QueryValueAction.Response)
	handler.RegistPCChainMessageHandler(msgtype.QUERYVALUE, QueryValueAction.Send, QueryValueAction.PCReceive, QueryValueAction.PCResponse)
}
