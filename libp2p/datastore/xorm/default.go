package xorm

import (
	"context"
	"errors"
	"fmt"
	"github.com/curltech/go-colla-core/content"
	"github.com/curltech/go-colla-core/crypto/std"
	baseentity "github.com/curltech/go-colla-core/entity"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-core/util/reflect"
	"github.com/curltech/go-colla-node/libp2p/datastore/handler"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/ns"
	chainentity "github.com/curltech/go-colla-node/p2p/chain/entity"
	handler2 "github.com/curltech/go-colla-node/p2p/chain/handler"
	service1 "github.com/curltech/go-colla-node/p2p/chain/service"
	dhtentity "github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	util "github.com/ipfs/go-ipfs-util"
	record "github.com/libp2p/go-libp2p-record"
	recpb "github.com/libp2p/go-libp2p-record/pb"
	"github.com/multiformats/go-base32"
	"google.golang.org/protobuf/proto"
	"strings"
	"time"
)

// XormDatastore uses a standard Go map for internal storage.
type XormDatastore struct {
}

// NewXormDatastore constructs a XormDatastore. It is _not_ thread-safe by
// default, wrap using sync.MutexWrap if you need thread safety (the answer here
// is usually yes).
func NewXormDatastore() (this *XormDatastore) {
	return &XormDatastore{}
}

// Put implements Datastore.Put
func (this *XormDatastore) Put(ctx context.Context, key datastore.Key, value []byte) (err error) {
	req, err := handler.NewKeyRequest(key)
	if err != nil {
		return err
	}

	keyId := strings.TrimPrefix(key.String(), "/")
	keyBuf, err := base32.RawStdEncoding.DecodeString(keyId)
	if err != nil {
		return err
	}
	keyString := string(keyBuf)
	namespace, _, err := record.SplitKey(keyString)
	if err != nil {
		return err
	}

	rec := new(recpb.Record)
	err = proto.Unmarshal(value, rec)
	if err != nil {
		logger.Sugar.Errorf("failed to unmarshal record from value", "key", key, "error", err)

		return err
	}
	value = rec.Value
	entities, err := req.Service.ParseJSON(value)
	if err != nil {
		return err
	}
	for _, entity := range entities {
		keyvalue, err := reflect.GetValue(entity, req.Keyname)
		if err != nil || keyvalue == nil {
			logger.Sugar.Errorf("NoKeyValue")
			return errors.New("NoKeyValue")
		}
		old, _ := req.Service.NewEntity(nil)
		//reflect.SetValue(old, req.Keyname, keyvalue)
		if namespace == ns.PeerClient_Prefix || namespace == ns.PeerClient_Mobile_Prefix || namespace == ns.PeerClient_Email_Prefix || namespace == ns.PeerClient_Name_Prefix {
			peerId, err := reflect.GetValue(entity, "PeerId")
			if err != nil || peerId == nil {
				logger.Sugar.Errorf("NoPeerId")
				return errors.New("NoPeerId")
			}
			reflect.SetValue(old, "PeerId", peerId)
			clientId, err := reflect.GetValue(entity, "ClientId")
			if err != nil || clientId == nil {
				logger.Sugar.Errorf("NoClientId")
				return errors.New("NoClientId")
			}
			reflect.SetValue(old, "ClientId", clientId)
		} else if namespace == ns.DataBlock_Prefix || namespace == ns.DataBlock_Owner_Prefix {
			blockId, err := reflect.GetValue(entity, "BlockId")
			if err != nil || blockId == nil {
				logger.Sugar.Errorf("NoBlockId")
				return errors.New("NoBlockId")
			}
			reflect.SetValue(old, "BlockId", blockId)
			sliceNumber, err := reflect.GetValue(entity, "SliceNumber")
			if err != nil || sliceNumber == 0 {
				logger.Sugar.Errorf("NoSliceNumber")
				return errors.New("NoSliceNumber")
			}
			reflect.SetValue(old, "SliceNumber", sliceNumber)
		} else if namespace == ns.PeerTransaction_Src_Prefix || namespace == ns.PeerTransaction_Target_Prefix ||
			namespace == ns.PeerTransaction_P2PChat_Prefix || namespace == ns.PeerTransaction_GroupFile_Prefix ||
			namespace == ns.PeerTransaction_Channel_Prefix || namespace == ns.PeerTransaction_ChannelArticle_Prefix {
			targetPeerId, err := reflect.GetValue(entity, "TargetPeerId")
			if err != nil || targetPeerId == nil {
				logger.Sugar.Errorf("NoTargetPeerId")
				return errors.New("NoTargetPeerId")
			}
			reflect.SetValue(old, "TargetPeerId", targetPeerId)
			blockId, err := reflect.GetValue(entity, "BlockId")
			if err != nil || blockId == nil {
				logger.Sugar.Errorf("NoBlockId")
				return errors.New("NoBlockId")
			}
			reflect.SetValue(old, "BlockId", blockId)
			sliceNumber, err := reflect.GetValue(entity, "SliceNumber")
			if err != nil || sliceNumber == 0 {
				logger.Sugar.Errorf("NoSliceNumber")
				return errors.New("NoSliceNumber")
			}
			reflect.SetValue(old, "SliceNumber", sliceNumber)
			businessNumber, err := reflect.GetValue(entity, "BusinessNumber")
			if err != nil || businessNumber == nil {
				logger.Sugar.Errorf("NoBusinessNumber")
				return errors.New("NoBusinessNumber")
			}
			reflect.SetValue(old, "BusinessNumber", businessNumber)
		} else { // e.g. namespace == ns.PeerEndpoint_Prefix
			reflect.SetValue(old, req.Keyname, keyvalue)
		}
		currentTime := time.Now()
		found, err := req.Service.Get(old, false, "", "")
		if found {
			id, err := reflect.GetValue(old, baseentity.FieldName_Id)
			if err != nil {
				id = uint64(0)
			}
			reflect.SetValue(entity, baseentity.FieldName_Id, id)
			if namespace == ns.PeerClient_Prefix || namespace == ns.PeerClient_Mobile_Prefix || namespace == ns.PeerClient_Email_Prefix || namespace == ns.PeerClient_Name_Prefix {

			} else if namespace == ns.DataBlock_Prefix || namespace == ns.DataBlock_Owner_Prefix {
				oldp := old.(*chainentity.DataBlock)
				p := entity.(*chainentity.DataBlock)
				// 校验Owner
				if p.BlockType == chainentity.BlockType_P2pChat && len(p.TransportPayload) == 0 {
					if oldp.BusinessNumber != p.PeerId {
						return errors.New(fmt.Sprintf("InconsistentDataBlockPeerId, blockId: %v, peerId: %v, oldBusinessNumber: %v", p.BlockId, p.PeerId, oldp.BusinessNumber))
					}
				} else {
					if oldp.PeerId != p.PeerId {
						return errors.New(fmt.Sprintf("InconsistentDataBlockPeerId, blockId: %v, peerId: %v, oldPeerId: %v", p.BlockId, p.PeerId, oldp.PeerId))
					}
				}
				// 负载为空表示删除
				if len(p.TransportPayload) == 0 {
					// 只针对第一个分片处理一次
					if p.SliceNumber == 1 {
						for i := uint64(1); i <= oldp.SliceSize; i++ {
							contentId := std.EncodeHex(std.Hash(fmt.Sprintf("%v-%v", oldp.BlockId, i), "sha3_256"))
							content.FileContent.Write(contentId, nil)
						}
						condition := &chainentity.DataBlock{}
						condition.BlockId = p.BlockId
						req.Service.Delete(condition, "")
						// 删除TransactionKeys
						condition2 := &chainentity.TransactionKey{}
						condition2.BlockId = p.BlockId
						service1.GetTransactionKeyService().Delete(condition2, "")
						// 删除PeerTransaction
						for i := uint64(1); i <= oldp.SliceSize; i++ {
							peerTransaction := chainentity.PeerTransaction{}
							peerTransaction.SrcPeerId = p.PeerId
							peerTransaction.TargetPeerId = global.Global.MyselfPeer.PeerId
							peerTransaction.BlockId = p.BlockId
							peerTransaction.SliceNumber = i
							peerTransaction.TransactionType = fmt.Sprintf("%v-%v", dhtentity.TransactionType_DataBlock, p.BlockType)
							peerTransaction.BusinessNumber = p.BusinessNumber
							peerTransaction.Status = baseentity.EntityState_Deleted
							err = service1.GetPeerTransactionService().PutPTs(&peerTransaction)
							if err != nil {
								return err
							}
						}
					}
					continue
				}
			} else if namespace == ns.PeerTransaction_Src_Prefix || namespace == ns.PeerTransaction_Target_Prefix ||
				namespace == ns.PeerTransaction_P2PChat_Prefix || namespace == ns.PeerTransaction_GroupFile_Prefix ||
				namespace == ns.PeerTransaction_Channel_Prefix || namespace == ns.PeerTransaction_ChannelArticle_Prefix {
				oldp := old.(*chainentity.PeerTransaction)
				p := entity.(*chainentity.PeerTransaction)
				// Status == baseentity.EntityState_Deleted表示删除
				if p.Status == baseentity.EntityState_Deleted {
					req.Service.Delete(oldp, "")
					continue
				}
			}
		} else {
			reflect.SetValue(entity, baseentity.FieldName_Id, uint64(0))
			if namespace == ns.PeerClient_Prefix || namespace == ns.PeerClient_Mobile_Prefix || namespace == ns.PeerClient_Email_Prefix || namespace == ns.PeerClient_Name_Prefix {

			} else if namespace == ns.DataBlock_Prefix || namespace == ns.DataBlock_Owner_Prefix {
				p := entity.(*chainentity.DataBlock)
				// 负载为空表示删除
				if len(p.TransportPayload) == 0 {
					continue
				}
			} else if namespace == ns.PeerTransaction_Src_Prefix || namespace == ns.PeerTransaction_Target_Prefix ||
				namespace == ns.PeerTransaction_P2PChat_Prefix || namespace == ns.PeerTransaction_GroupFile_Prefix ||
				namespace == ns.PeerTransaction_Channel_Prefix || namespace == ns.PeerTransaction_ChannelArticle_Prefix {
				p := entity.(*chainentity.PeerTransaction)
				// Status == baseentity.EntityState_Deleted表示删除
				if p.Status == baseentity.EntityState_Deleted {
					continue
				}
			}
		}
		if namespace == ns.DataBlock_Prefix || namespace == ns.DataBlock_Owner_Prefix {
			p := entity.(*chainentity.DataBlock)
			if len(p.TransportPayload) > handler2.PayloadLimit {
				transportPayload := std.DecodeBase64(p.TransportPayload)
				contentId := std.EncodeHex(std.Hash(fmt.Sprintf("%v-%v", p.BlockId, p.SliceNumber), "sha3_256"))
				content.FileContent.Write(contentId, transportPayload)
				p.TransportPayload = ""
			}
		}

		affected, err := req.Service.Upsert(entity)
		if affected > 0 {
			logger.Sugar.Debugf("%v:%v put successfully", req.Keyname, req.Keyvalue)
			if namespace == ns.DataBlock_Prefix || namespace == ns.DataBlock_Owner_Prefix {
				oldp := old.(*chainentity.DataBlock)
				p := entity.(*chainentity.DataBlock)
				// 只针对第一个分片处理一次
				if p.SliceNumber == 1 {
					// 删除多余废弃分片
					if p.SliceSize < oldp.SliceSize {
						condition := &chainentity.DataBlock{}
						condition.BlockId = p.BlockId
						req.Service.Delete(condition, "SliceNumber > ?", p.SliceSize)
						// 删除PeerTransaction
						for i := p.SliceSize + 1; i <= oldp.SliceSize; i++ {
							peerTransaction := chainentity.PeerTransaction{}
							peerTransaction.SrcPeerId = p.PeerId
							peerTransaction.TargetPeerId = global.Global.MyselfPeer.PeerId
							peerTransaction.BlockId = p.BlockId
							peerTransaction.SliceNumber = i
							peerTransaction.TransactionType = fmt.Sprintf("%v-%v", dhtentity.TransactionType_DataBlock, p.BlockType)
							peerTransaction.BusinessNumber = p.BusinessNumber
							peerTransaction.Status = baseentity.EntityState_Deleted
							err = service1.GetPeerTransactionService().PutPTs(&peerTransaction)
							if err != nil {
								return err
							}
						}
					}
					// 保存TransactionKeys
					for _, tk := range p.TransactionKeys {
						tkBlockId := tk.BlockId
						if tkBlockId != p.BlockId {
							logger.Sugar.Errorf("InvalidTKBlockId")
							return errors.New("InvalidTKBlockId")
						}
						tkPeerId := tk.PeerId
						if tkPeerId == "" {
							logger.Sugar.Errorf("NoTKPeerId")
							return errors.New("NoTKPeerId")
						}
						oldTk := &chainentity.TransactionKey{}
						oldTk.BlockId = tkBlockId
						oldTk.PeerId = tkPeerId
						tkFound, _ := service1.GetTransactionKeyService().Get(oldTk, false, "", "")
						if tkFound {
							tk.Id = oldTk.Id
						} else {
							tk.Id = uint64(0)
						}

						tkAffected, _ := service1.GetTransactionKeyService().Upsert(tk)
						if tkAffected > 0 {
							logger.Sugar.Infof("BlockId: %v, PeerId: %v, upsert TransactionKey successfully", tkBlockId, tkPeerId)
						} else {
							logger.Sugar.Errorf("BlockId: %v, PeerId: %v, upsert TransactionKey fail", tkBlockId, tkPeerId)
							return errors.New(fmt.Sprintf("BlockId: %v, PeerId: %v, upsert TransactionKey fail", tkBlockId, tkPeerId))
						}
					}
				}

				// PeerTransaction（BlockType_ChatAttach不需要保存PeerTransaction）
				if p.BlockType != chainentity.BlockType_ChatAttach {
					peerTransaction := chainentity.PeerTransaction{}
					peerTransaction.PrimaryPeerId = p.PrimaryPeerId
					peerTransaction.SrcPeerId = p.PeerId
					peerTransaction.SrcPeerType = dhtentity.PeerType_PeerClient
					peerTransaction.PrimaryPeerId = p.PrimaryPeerId
					peerTransaction.TargetPeerId = global.Global.MyselfPeer.PeerId
					peerTransaction.TargetPeerType = dhtentity.PeerType_PeerEndpoint
					peerTransaction.BlockId = p.BlockId
					peerTransaction.SliceNumber = p.SliceNumber
					peerTransaction.ParentBusinessNumber = p.ParentBusinessNumber
					peerTransaction.BusinessNumber = p.BusinessNumber
					peerTransaction.TransactionTime = &currentTime
					peerTransaction.CreateTimestamp = p.CreateTimestamp
					peerTransaction.Amount = p.TransactionAmount
					peerTransaction.TransactionType = fmt.Sprintf("%v-%v", dhtentity.TransactionType_DataBlock, p.BlockType)
					peerTransaction.Metadata = p.Metadata
					peerTransaction.Thumbnail = p.Thumbnail
					peerTransaction.Name = p.Name
					peerTransaction.Description = p.Description
					err = service1.GetPeerTransactionService().PutPTs(&peerTransaction)
					if err != nil {
						return err
					}
				}
			}
		} else {
			logger.Sugar.Errorf("%v:%v upsert fail", req.Keyname, req.Keyvalue)
			return errors.New(fmt.Sprintf("%v:%v upsert fail", req.Keyname, req.Keyvalue))
		}
	}

	return nil
}

// Sync implements Datastore.Sync
func (this *XormDatastore) Sync(ctx context.Context, prefix datastore.Key) error {
	return nil
}

/*
*
GetValue其实可以支持返回多条记录和全文检索结果
一般Key的格式是/peerEndpoint/12D3KooWG59NPEuY1dseFzXMSyYbHQb1pfpPiMq5fk7c48exxNJp
如果需要支持条件查询，第二个/后的格式就不是这样的，可以用=表示条件，类似url，甚至类似elastic的查询条件
*/
func (this *XormDatastore) Get(ctx context.Context, key datastore.Key) (value []byte, err error) {
	req, err := handler.NewKeyRequest(key)
	if err != nil {
		return nil, err
	}

	//rec.Key = key.Bytes()
	keyId := strings.TrimPrefix(key.String(), "/")
	keyBuf, err := base32.RawStdEncoding.DecodeString(keyId)
	if err != nil {
		return nil, err
	}
	keyString := string(keyBuf)
	namespace, _, err := record.SplitKey(keyString)
	if err != nil {
		return nil, err
	}

	entities := this.get(req)

	if namespace == ns.PeerClient_Prefix || namespace == ns.PeerClient_Mobile_Prefix || namespace == ns.PeerClient_Email_Prefix || namespace == ns.PeerClient_Name_Prefix {
		if len(*entities.(*[]*dhtentity.PeerClient)) == 0 {
			return nil, datastore.ErrNotFound
		}
	} else if namespace == ns.DataBlock_Prefix || namespace == ns.DataBlock_Owner_Prefix {
		if len(*entities.(*[]*chainentity.DataBlock)) == 0 {
			return nil, datastore.ErrNotFound
		}
		for _, entity := range *entities.(*[]*chainentity.DataBlock) {
			if entity.SliceNumber == 1 {
				condition := &chainentity.TransactionKey{}
				condition.BlockId = entity.BlockId
				transactionKeys := make([]*chainentity.TransactionKey, 0)
				service1.GetTransactionKeyService().Find(&transactionKeys, condition, "", 0, 0, "")
				if len(transactionKeys) > 0 {
					entity.TransactionKeys = transactionKeys
				}
			}
		}
	} else if namespace == ns.PeerTransaction_Src_Prefix || namespace == ns.PeerTransaction_Target_Prefix {
		if len(*entities.(*[]*chainentity.PeerTransaction)) == 0 {
			return nil, datastore.ErrNotFound
		}
	}

	val, err := message.Marshal(entities)
	if err != nil {
		return nil, err
	}
	rec := new(recpb.Record)
	rec.Key = keyBuf
	rec.Value = val
	rec.TimeReceived = util.FormatRFC3339(time.Now())
	buf, err := proto.Marshal(rec)
	if err != nil {
		logger.Sugar.Errorf("failed to marshal record from datastore", "key", keyString, "error", err)
		return nil, err
	}

	return buf, nil
}

func (this *XormDatastore) get(req *handler.DispatchRequest) interface{} {
	entity, _ := req.Service.NewEntity(nil)
	for k, v := range req.Keyvalue {
		if req.Name == ns.PeerClient_Mobile_Prefix {
			reflect.SetValue(entity, ns.PeerClient_Mobile_KeyKind, v)
			break
		} else if req.Name == ns.PeerClient_Email_Prefix {
			reflect.SetValue(entity, ns.PeerClient_Email_KeyKind, v)
			break
		} else if req.Name == ns.PeerClient_Name_Prefix {
			reflect.SetValue(entity, ns.PeerClient_Name_KeyKind, v)
			break
		} else if req.Name == ns.DataBlock_Owner_Prefix {
			reflect.SetValue(entity, ns.DataBlock_Owner_KeyKind, v)
			break
		} else if req.Name == ns.PeerTransaction_Src_Prefix {
			reflect.SetValue(entity, k, v)
			reflect.SetValue(entity, ns.PeerTransaction_Type_KeyKind, fmt.Sprintf("%v-%v", dhtentity.TransactionType_DataBlock, chainentity.BlockType_Collection))
			break
		} else if req.Name == ns.PeerTransaction_Target_Prefix {
			reflect.SetValue(entity, ns.PeerTransaction_Target_KeyKind, v)
			reflect.SetValue(entity, ns.PeerTransaction_Type_KeyKind, fmt.Sprintf("%v-%v", dhtentity.TransactionType_DataBlock, chainentity.BlockType_Collection))
			break
		} else if req.Name == ns.PeerTransaction_P2PChat_Prefix {
			reflect.SetValue(entity, ns.PeerTransaction_P2PChat_KeyKind, v)
			reflect.SetValue(entity, ns.PeerTransaction_Type_KeyKind, fmt.Sprintf("%v-%v", dhtentity.TransactionType_DataBlock, chainentity.BlockType_P2pChat))
			break
		} else if req.Name == ns.PeerTransaction_GroupFile_Prefix {
			reflect.SetValue(entity, ns.PeerTransaction_GroupFile_KeyKind, v)
			reflect.SetValue(entity, ns.PeerTransaction_Type_KeyKind, fmt.Sprintf("%v-%v", dhtentity.TransactionType_DataBlock, chainentity.BlockType_GroupFile))
			break
		} else if req.Name == ns.PeerTransaction_Channel_Prefix {
			reflect.SetValue(entity, ns.PeerTransaction_Channel_KeyKind, v)
			/*reflect.SetValue(entity, ns.PeerTransaction_Type_KeyKind, fmt.Sprintf("%v-%v", dhtentity.TransactionType_DataBlock, chainentity.BlockType_Channel))*/
			break
		} else if req.Name == ns.PeerTransaction_ChannelArticle_Prefix {
			reflect.SetValue(entity, ns.PeerTransaction_ChannelArticle_KeyKind, v)
			reflect.SetValue(entity, ns.PeerTransaction_Type_KeyKind, fmt.Sprintf("%v-%v", dhtentity.TransactionType_DataBlock, chainentity.BlockType_ChannelArticle))
			break
		} else {
			err := reflect.SetValue(entity, k, v)
			if err != nil {
				continue
			}
		}
	}
	entities, _ := req.Service.NewEntities(nil)
	req.Service.Find(entities, entity, "", 0, 0, "")
	if req.Name == ns.DataBlock_Prefix || req.Name == ns.DataBlock_Owner_Prefix {
		for _, entity := range *entities.(*[]*chainentity.DataBlock) {
			if entity.TransportPayload == "" {
				contentId := std.EncodeHex(std.Hash(fmt.Sprintf("%v-%v", entity.BlockId, entity.SliceNumber), "sha3_256"))
				transportPayload, _ := content.FileContent.Read(contentId)
				entity.TransportPayload = std.EncodeBase64(transportPayload)
			}
		}
	}

	return entities
}

// Has implements Datastore.Has
func (this *XormDatastore) Has(ctx context.Context, key datastore.Key) (exists bool, err error) {
	req, err := handler.NewKeyRequest(key)
	if err != nil {
		return false, err
	}
	entities := this.get(req)
	if entities != nil {
		es := reflect.ToArray(entities)
		if len(es) > 0 {
			return true, nil
		}
	}

	return false, nil
}

// GetSize implements Datastore.GetSize
func (this *XormDatastore) GetSize(ctx context.Context, key datastore.Key) (size int, err error) {
	req, err := handler.NewKeyRequest(key)
	if err != nil {
		return 0, err
	}
	entity, _ := req.Service.NewEntity(nil)
	for k, v := range req.Keyvalue {
		err := reflect.SetValue(entity, k, v)
		if err != nil {
			continue
		}
	}
	count, _ := req.Service.Count(entity, "")

	return int(count), nil
}

// Delete implements Datastore.Delete
func (this *XormDatastore) Delete(ctx context.Context, key datastore.Key) (err error) {
	req, err := handler.NewKeyRequest(key)
	if err != nil {
		return err
	}
	entity, err := req.Service.NewEntity(nil)
	if err != nil {
		return err
	}
	v, ok := req.Keyvalue[req.Keyname]
	if !ok {
		return errors.New("Delete need keyvalue")
	}
	reflect.SetValue(entity, req.Keyname, v)
	affected, err := req.Service.Delete(entity, "")
	if affected > 0 {
		logger.Sugar.Infof("%v:%v delete successfully", req.Keyname, req.Keyvalue)
	} else {
		logger.Sugar.Errorf("%v:%v delete fail", req.Keyname, req.Keyvalue)
		return errors.New("delete fail")
	}

	return nil
}

// Query implements Datastore.Query
func (this *XormDatastore) Query(ctx context.Context, q dsq.Query) (dsq.Results, error) {
	logger.Sugar.Warnf("query trigger:%v:%v", q.Prefix, q.String())
	return nil, nil
}

func (this *XormDatastore) Batch(ctx context.Context) (datastore.Batch, error) {
	return datastore.NewBasicBatch(this), nil
}

func (this *XormDatastore) Close() error {
	return nil
}

func init() {
	handler.RegistDatastore(ns.PeerEndpoint_Prefix, NewXormDatastore())
	handler.RegistKeyname(ns.PeerEndpoint_Prefix, dhtentity.PeerEndpoint{}.KeyName())

	handler.RegistDatastore(ns.PeerClient_Prefix, NewXormDatastore())
	handler.RegistKeyname(ns.PeerClient_Prefix, dhtentity.PeerClient{}.KeyName())

	handler.RegistDatastore(ns.PeerClient_Mobile_Prefix, NewXormDatastore())
	handler.RegistKeyname(ns.PeerClient_Mobile_Prefix, ns.PeerClient_Mobile_KeyKind)

	handler.RegistDatastore(ns.PeerClient_Email_Prefix, NewXormDatastore())
	handler.RegistKeyname(ns.PeerClient_Email_Prefix, ns.PeerClient_Email_KeyKind)

	handler.RegistDatastore(ns.PeerClient_Name_Prefix, NewXormDatastore())
	handler.RegistKeyname(ns.PeerClient_Name_Prefix, ns.PeerClient_Name_KeyKind)

	handler.RegistDatastore(ns.ChainApp_Prefix, NewXormDatastore())
	handler.RegistKeyname(ns.ChainApp_Prefix, dhtentity.ChainApp{}.KeyName())

	handler.RegistDatastore(ns.DataBlock_Prefix, NewXormDatastore())
	handler.RegistKeyname(ns.DataBlock_Prefix, chainentity.DataBlock{}.KeyName())

	handler.RegistDatastore(ns.DataBlock_Owner_Prefix, NewXormDatastore())
	handler.RegistKeyname(ns.DataBlock_Owner_Prefix, ns.DataBlock_Owner_KeyKind)

	handler.RegistDatastore(ns.PeerTransaction_Src_Prefix, NewXormDatastore())
	handler.RegistKeyname(ns.PeerTransaction_Src_Prefix, chainentity.PeerTransaction{}.KeyName())

	handler.RegistDatastore(ns.PeerTransaction_Target_Prefix, NewXormDatastore())
	handler.RegistKeyname(ns.PeerTransaction_Target_Prefix, ns.PeerTransaction_Target_KeyKind)

	handler.RegistDatastore(ns.TransactionKey_Prefix, NewXormDatastore())
	handler.RegistKeyname(ns.TransactionKey_Prefix, chainentity.TransactionKey{}.KeyName())

	handler.RegistDatastore(ns.PeerTransaction_P2PChat_Prefix, NewXormDatastore())
	handler.RegistKeyname(ns.PeerTransaction_P2PChat_Prefix, ns.PeerTransaction_P2PChat_KeyKind)

	handler.RegistDatastore(ns.PeerTransaction_GroupFile_Prefix, NewXormDatastore())
	handler.RegistKeyname(ns.PeerTransaction_GroupFile_Prefix, ns.PeerTransaction_GroupFile_KeyKind)

	handler.RegistDatastore(ns.PeerTransaction_Channel_Prefix, NewXormDatastore())
	handler.RegistKeyname(ns.PeerTransaction_Channel_Prefix, ns.PeerTransaction_Channel_KeyKind)

	handler.RegistDatastore(ns.PeerTransaction_ChannelArticle_Prefix, NewXormDatastore())
	handler.RegistKeyname(ns.PeerTransaction_ChannelArticle_Prefix, ns.PeerTransaction_ChannelArticle_KeyKind)
}
