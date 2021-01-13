package action

import (
	"errors"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/ns"
	entity2 "github.com/curltech/go-colla-node/p2p/chain/entity"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	service1 "github.com/curltech/go-colla-node/p2p/chain/service"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/kataras/golog"
	"strconv"
	"time"
)

type queryPeerTransAction struct {
	BaseAction
}

var QueryPeerTransAction queryPeerTransAction

/**
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (this *queryPeerTransAction) PCReceive(chainMessage *msg.PCChainMessage) (interface{}, error) {
	golog.Infof("Receive %v message", this.MsgType)
	conditionBean := chainMessage.MessagePayload.Payload.(map[string]interface{})
	var srcPeerId string = ""
	var targetPeerId string = ""
	var transactionTimeStart *time.Time = nil
	if conditionBean["receiverPeerId"] != nil {
		srcPeerId = conditionBean["receiverPeerId"].(string)
	}
	if conditionBean["targetPeerId"] != nil {
		targetPeerId = conditionBean["targetPeerId"].(string)
	}
	if conditionBean["transactionTimeStart"] != nil {
		transactionTime, err := time.Parse(time.RFC3339Nano, conditionBean["transactionTimeStart"].(string))
		if err != nil {
			return msgtype.ERROR, err
		}
		transactionTimeStart = &transactionTime
	}

	ptMap := make(map[string]*entity2.PeerTransaction, 0)
	var key string
	if len(srcPeerId) > 0 {
		key = ns.GetPeerTransactionSrcKey(srcPeerId)
	} else if len(targetPeerId) > 0 {
		key = ns.GetPeerTransactionTargetKey(targetPeerId)
	} else {
		golog.Errorf("InvalidPeerTransactioKey")
		return msgtype.ERROR, errors.New("InvalidPeerTransactioKey")
	}
	if config.Libp2pParams.FaultTolerantLevel == 0 {
		recvdVals, err := dht.PeerEndpointDHT.GetValues(key, config.Libp2pParams.Nvals)
		if err != nil {
			return msgtype.ERROR, err
		}
		for _, recvdVal := range recvdVals {
			pts := make([]*entity2.PeerTransaction, 0)
			err = message.TextUnmarshal(string(recvdVal.Val), &pts)
			if err != nil {
				golog.Errorf("failed to TextUnmarshal PeerTransaction value: %v, err: %v", recvdVal.Val, err)
				return msgtype.ERROR, err
			}
			for _, pt := range pts {
				index := pt.BlockId + "/" + strconv.FormatUint(pt.TxSequenceId, 10) + "/" + strconv.FormatUint(pt.SliceNumber, 10) + "/" + pt.SrcPeerId + "/" + pt.TargetPeerId
				if ptMap[index] == nil {
					ptMap[index] = pt
				}
			}
		}
	} else if config.Libp2pParams.FaultTolerantLevel == 1 {
		// 查询删除local记录
		var locals []*entity2.PeerTransaction
		var err error
		if len(srcPeerId) > 0 {
			locals, err = service1.GetLocalPTs(ns.PeerTransaction_Src_KeyKind, srcPeerId, "")
		} else if len(targetPeerId) > 0 {
			locals, err = service1.GetLocalPTs(ns.PeerTransaction_Target_KeyKind, "", targetPeerId)
		}
		if err != nil {
			return msgtype.ERROR, err
		}
		if len(locals) > 0 {
			for _, local := range locals {
				index := local.BlockId + "/" + strconv.FormatUint(local.TxSequenceId, 10) + "/" + strconv.FormatUint(local.SliceNumber, 10) + "/" + local.SrcPeerId + "/" + local.TargetPeerId
				if ptMap[index] == nil {
					ptMap[index] = local
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
				golog.Errorf("failed to TextUnmarshal PeerTransaction value: %v, err: %v", recvdVal.Val, err)
				return msgtype.ERROR, err
			}
			for _, pt := range pts {
				index := pt.BlockId + "/" + strconv.FormatUint(pt.TxSequenceId, 10) + "/" + strconv.FormatUint(pt.SliceNumber, 10) + "/" + pt.SrcPeerId + "/" + pt.TargetPeerId
				if ptMap[index] == nil {
					ptMap[index] = pt
				}
			}
		}
	}

	peerTransactions := make([]*entity2.PeerTransaction, 0)
	for _, v := range ptMap {
		if transactionTimeStart != nil && v.TransactionTime.UTC().After(transactionTimeStart.UTC()) {
			peerTransactions = append(peerTransactions, v)
		}
	}

	return peerTransactions, nil
}

/**
处理返回消息
*/
func (this *queryPeerTransAction) PCResponse(chainMessage *msg.PCChainMessage) error {
	golog.Infof("Response %v message:%v", this.MsgType, chainMessage)

	return nil
}

func init() {
	QueryPeerTransAction = queryPeerTransAction{}
	QueryPeerTransAction.MsgType = msgtype.QUERYPEERTRANS
	handler.RegistChainMessageHandler(msgtype.QUERYPEERTRANS, QueryPeerTransAction.Send, QueryPeerTransAction.Receive, QueryPeerTransAction.Response)
	handler.RegistPCChainMessageHandler(msgtype.QUERYPEERTRANS, QueryPeerTransAction.Send, QueryPeerTransAction.PCReceive, QueryPeerTransAction.PCResponse)
}
