package dht

import (
	"bytes"
	"context"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/datastore/handler"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/libp2p/routingtable"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/gogo/protobuf/proto"
	"github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	u "github.com/ipfs/go-ipfs-util"
	"github.com/jbenet/goprocess"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	kb "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p-kbucket/peerdiversity"
	record "github.com/libp2p/go-libp2p-record"
	recpb "github.com/libp2p/go-libp2p-record/pb"
	"github.com/multiformats/go-base32"
	"strings"
	"time"
)

/**
提供了dht的服务的封装供外部调用，但是p2p的节点之间的消息不通过这里
*/
type PeerEntityDHT struct {
	DHT          *dht.IpfsDHT
	RoutingTable *routingtable.PeerEntityRoutingTable
}

var PeerEndpointDHT *PeerEntityDHT = &PeerEntityDHT{}

var PeerClientDHT *PeerEntityDHT = &PeerEntityDHT{}

var ChainAppDHT *PeerEntityDHT = &PeerEntityDHT{}

func (this *PeerEntityDHT) PeerID() peer.ID {
	return this.DHT.PeerID()
}

func (this *PeerEntityDHT) PeerKey() []byte {
	return this.DHT.PeerKey()
}

func (this *PeerEntityDHT) Host() host.Host {
	return this.DHT.Host()
}

func (this *PeerEntityDHT) FindLocal(id peer.ID) peer.AddrInfo {
	return this.DHT.FindLocal(id)
}

func (this *PeerEntityDHT) Ping(p peer.ID) error {
	return this.DHT.Ping(global.Global.Context, p)
}

func (this *PeerEntityDHT) GetPublicKey(p peer.ID) (crypto.PubKey, error) {
	return this.DHT.GetPublicKey(global.Global.Context, p)
}

func (this *PeerEntityDHT) GetClosestPeers(key string) ([]peer.ID, error) {
	ctx, cancel := context.WithTimeout(global.Global.Context, time.Duration(time.Second*1))
	start := time.Now()
	pArray, err := this.DHT.GetClosestPeers(ctx, key)
	end := time.Now()
	logger.Sugar.Infof("GetClosestPeers time:%v, %v", key, end.Sub(start))
	cancel()
	if err != nil {
		return nil, err
	} else {
		return pArray, nil
	}
}

func (this *PeerEntityDHT) PutLocal(key string, value []byte, opts ...routing.Option) (err error) {
	logger.Sugar.Debugf("putting value in local datastore by key %v", key)

	// don't even allow local users to put bad values.
	if err := this.DHT.Validator.Validate(key, value); err != nil {
		return err
	}

	old, err := this.GetLocal(key)
	if err != nil {
		// Means something is wrong with the datastore.
		return err
	}

	// Check if we have an old value that's not the same as the new one.
	if old != nil && !bytes.Equal(old.GetValue(), value) {
		// Check to see if the new one is better.
		i, err := this.DHT.Validator.Select(key, [][]byte{value, old.GetValue()})
		if err != nil {
			return err
		}
		if i != 0 {
			//return fmt.Errorf("can't replace a newer value with an older value")
			logger.Sugar.Warnf("can't replace a newer value with an older value")
			return nil
		}
	}

	rec := record.MakePutRecord(key, value)
	rec.TimeReceived = u.FormatRFC3339(time.Now())
	data, err := proto.Marshal(rec)
	if err != nil {
		logger.Sugar.Errorf("failed to put marshal record for local put by key: %v, err: %v", key, err)
		return err
	}

	//return dht.datastore.Put(mkDsKey(key), data)
	dsKey := ds.NewKey(base32.RawStdEncoding.EncodeToString([]byte(key)))
	return handler.NewDispatchDatastore().Put(global.Global.Context, dsKey, data)
}

func (this *PeerEntityDHT) PutValue(key string, value []byte, opts ...routing.Option) (err error) {
	start := time.Now()
	error := this.DHT.PutValue(global.Global.Context, key, value, opts...)
	end := time.Now()
	logger.Sugar.Infof("PutValue time:%v, %v", key, end.Sub(start))
	if strings.HasPrefix(key, "/"+ns.PeerClient_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.PeerClient_Mobile_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.PeerClient_Name_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.DataBlock_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.DataBlock_Owner_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.PeerTransaction_Src_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.PeerTransaction_Target_Prefix) == true {
		if error != nil && error.Error() == "can't replace a newer value with an older value" {
			logger.Sugar.Warnf("can't replace a newer value with an older value")
			return nil
		} else if error == kb.ErrLookupFailure {
			logger.Sugar.Warnf("failed to find any peer in table")
			return nil
		}
	}
	return error
}

func (this *PeerEntityDHT) GetLocal(key string) (*recpb.Record, error) {
	logger.Sugar.Debugf("finding value in local datastore by key %v", key)

	dsKey := ds.NewKey(base32.RawStdEncoding.EncodeToString([]byte(key)))
	//buf, err := dht.datastore.Get(dskey)
	buf, err := handler.NewDispatchDatastore().Get(global.Global.Context, dsKey)
	if err == ds.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		logger.Sugar.Errorf("error retrieving record from local datastore by key %v, err: %v", key, err)
		return nil, err
	}
	rec := new(recpb.Record)
	err = proto.Unmarshal(buf, rec)
	if err != nil {
		// Bad data in datastore, log it but don't return an error, we'll just overwrite it
		logger.Sugar.Errorf("failed to unmarshal record from local datastore by key: %v, err: %v", key, err)
		return nil, nil
	}
	err = this.DHT.Validator.Validate(string(rec.GetKey()), rec.GetValue())
	if err != nil {
		// Invalid record in datastore, probably expired but don't return an error,
		// we'll just overwrite it
		logger.Sugar.Infof("local record verify failed by key: %v, err: %v", rec.GetKey(), err)
		return nil, nil
	}

	// Double check the key. Can't hurt.
	if rec != nil && string(rec.GetKey()) != key {
		logger.Sugar.Errorf("BUG: found a DHT record that didn't match it's key, expected: %v, got: %v", key, rec.GetKey())
		return nil, nil

	}
	return rec, nil
}

func (this *PeerEntityDHT) GetValue(key string, opts ...routing.Option) (_ []byte, err error) {
	start := time.Now()
	byteArr, error := this.DHT.GetValue(global.Global.Context, key, opts...)
	end := time.Now()
	logger.Sugar.Infof("GetValue time:%v, %v", key, end.Sub(start))
	if (strings.HasPrefix(key, "/"+ns.PeerClient_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.PeerClient_Mobile_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.PeerClient_Name_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.DataBlock_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.DataBlock_Owner_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.PeerTransaction_Src_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.PeerTransaction_Target_Prefix) == true) &&
		error == kb.ErrLookupFailure {
		logger.Sugar.Warnf("failed to find any peer in table")
		return byteArr, nil
	} else {
		return byteArr, error
	}
}

func (this *PeerEntityDHT) SearchValue(key string, opts ...routing.Option) (<-chan []byte, error) {
	valChs, error := this.DHT.SearchValue(global.Global.Context, key, opts...)
	if (strings.HasPrefix(key, "/"+ns.PeerClient_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.PeerClient_Mobile_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.PeerClient_Name_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.DataBlock_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.DataBlock_Owner_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.PeerTransaction_Src_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.PeerTransaction_Target_Prefix) == true) &&
		error == kb.ErrLookupFailure {
		logger.Sugar.Warnf("failed to find any peer in table")
		return valChs, nil
	} else {
		return valChs, error
	}
}

func (this *PeerEntityDHT) GetValues(key string, opts ...routing.Option) ([][]byte, error) {
	start := time.Now()
	//recvdVals, error := this.DHT.GetValues(global.Global.Context, key, opts...)
	valChs, error := this.DHT.SearchValue(global.Global.Context, key, opts...)
	var recvdVals [][]byte
	for r := range valChs {
		recvdVals = append(recvdVals, r)
	}
	end := time.Now()
	logger.Sugar.Infof("GetValues time:%v, %v", key, end.Sub(start))
	if (strings.HasPrefix(key, "/"+ns.PeerClient_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.PeerClient_Mobile_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.PeerClient_Name_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.DataBlock_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.DataBlock_Owner_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.PeerTransaction_Src_Prefix) == true ||
		strings.HasPrefix(key, "/"+ns.PeerTransaction_Target_Prefix) == true) &&
		error == kb.ErrLookupFailure {
		logger.Sugar.Warnf("failed to find any peer in table")
		return recvdVals, nil
	} else {
		return recvdVals, error
	}
}

func (this *PeerEntityDHT) FindPeer(id peer.ID) (_ peer.AddrInfo, err error) {
	start := time.Now()
	addrInfo, err := this.DHT.FindPeer(global.Global.Context, id)
	end := time.Now()
	logger.Sugar.Infof("FindPeer time:%v, %v", id.Pretty(), end.Sub(start))
	return addrInfo, err
}

func (this *PeerEntityDHT) RefreshRoutingTable() <-chan error {
	return this.DHT.RefreshRoutingTable()
}
func (this *PeerEntityDHT) ForceRefresh() <-chan error {
	return this.DHT.ForceRefresh()
}

func (this *PeerEntityDHT) Close() error {
	return this.DHT.Close()
}

func (this *PeerEntityDHT) GetRoutingTableDiversityStats() []peerdiversity.CplDiversityStats {
	return this.DHT.GetRoutingTableDiversityStats()
}

func (this *PeerEntityDHT) Mode() dht.ModeOpt {
	return this.DHT.Mode()
}

func (this *PeerEntityDHT) Context() context.Context {
	return this.DHT.Context()
}

func (this *PeerEntityDHT) Process() goprocess.Process {
	return this.DHT.Process()
}

func (this *PeerEntityDHT) Provide(key cid.Cid, brdcst bool) (err error) {
	return this.DHT.Provide(global.Global.Context, key, brdcst)
}

func (this *PeerEntityDHT) FindProviders(c cid.Cid) ([]peer.AddrInfo, error) {
	return this.DHT.FindProviders(global.Global.Context, c)
}

func (this *PeerEntityDHT) FindProvidersAsync(key cid.Cid, count int) <-chan peer.AddrInfo {
	return this.DHT.FindProvidersAsync(global.Global.Context, key, count)
}

func (this *PeerEntityDHT) Bootstrap() error {
	return this.DHT.Bootstrap(global.Global.Context)
}

func (this *PeerEntityDHT) PutMyself() error {
	// 写自己的数据到peerendpoint中
	peerEndpoint := entity.PeerEndpoint{}
	byteMyselfPeer, err := message.Marshal(global.Global.MyselfPeer)
	if err != nil {
		return err
	}
	err = message.Unmarshal(byteMyselfPeer, &peerEndpoint)
	if err != nil {
		return err
	}
	peerEndpoint.ActiveStatus = entity.ActiveStatus_Up
	bytePeerEndpoint, err := message.Marshal(peerEndpoint)
	if err != nil {
		return err
	}
	key := ns.GetPeerEndpointKey(peerEndpoint.PeerId)
	return this.PutValue(key, bytePeerEndpoint)
}
