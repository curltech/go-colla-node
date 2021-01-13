package action

import (
	"errors"
	"github.com/curltech/go-colla-core/crypto/std"
	entity3 "github.com/curltech/go-colla-core/entity"
	"github.com/curltech/go-colla-core/util/message"
	entity2 "github.com/curltech/go-colla-node/p2p/chain/entity"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/chain/service"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/kataras/golog"
	"unsafe"
)

type storeAction struct {
	BaseAction
}

var StoreAction storeAction

/**
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (this *storeAction) PCReceive(chainMessage *msg.PCChainMessage) (interface{}, error) {
	golog.Infof("Receive %v message", this.MsgType)
	dataBlock := chainMessage.MessagePayload.Payload.(*entity2.DataBlock)
	err := service.ValidateDB(chainMessage.MessagePayload)
	if err != nil {
		return msgtype.ERROR, err
	}

	// 只针对第一个分片处理一次
	if dataBlock.SliceNumber == 1 {
		if len(dataBlock.TransportKey) == 0 {
			return msgtype.ERROR, errors.New("NullTransportKey")
		}
		transportKey := std.DecodeBase64(dataBlock.TransportKey)
		transactionKeys := make([]*entity2.TransactionKey, 0)
		err = message.TextUnmarshal(*(*string)(unsafe.Pointer(&transportKey)), &transactionKeys)
		if err != nil {
			return msgtype.ERROR, errors.New("TransactionKeysTextUnmarshalFailure")
		}
		dataBlock.TransactionKeys = transactionKeys
	}

	transportPayload := std.DecodeBase64(dataBlock.TransportPayload)
	dataBlock.TransactionAmount = service.GetTransactionAmount(transportPayload)

	//dataBlock.CreateTimestamp = uint64(currentTime.Nanosecond())
	//dataBlock.CreateTimestampNanos = uint64(currentTime.Nanosecond())
	dataBlock.Status = entity3.EntityStatus_Effective

	err = service.PutDBs(dataBlock)
	if err != nil {
		return msgtype.ERROR, err
	}

	return msgtype.OK, nil
}

/**
处理返回消息
*/
func (this *storeAction) PCResponse(chainMessage *msg.PCChainMessage) error {
	golog.Infof("Response %v message", this.MsgType)

	return nil
}

func init() {
	StoreAction = storeAction{}
	StoreAction.MsgType = msgtype.STORE
	handler.RegistChainMessageHandler(msgtype.STORE, StoreAction.Send, StoreAction.Receive, StoreAction.Response)
	handler.RegistPCChainMessageHandler(msgtype.STORE, StoreAction.Send, StoreAction.PCReceive, StoreAction.PCResponse)
}
