package libp2p

import (
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/datastore/handler"
	"github.com/curltech/go-colla-node/libp2p/datastore/xorm"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/chain/action/dht"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	badger "github.com/ipfs/go-ds-badger2"
	"github.com/ipfs/go-ds-flatfs"
	"github.com/ipfs/go-ds-leveldb"
	"github.com/ipfs/go-ds-redis"
	sqlds "github.com/ipfs/go-ds-sql"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"time"
)

func dhtOptions() []kaddht.Option {
	options := make([]kaddht.Option, 0)

	// Mode configures which mode the DHT operates in (Client, Server, Auto).
	//
	// Defaults to ModeAuto.
	//必须设置成2，否则会不支持kad协议，无法自动更新节点信息和dht
	mode, _ := config.GetInt("p2p.dht.datastore", 2)
	var m kaddht.ModeOpt
	if mode == 0 {
		m = kaddht.ModeAuto
	} else if mode == 1 {
		m = kaddht.ModeClient
	} else if mode == 2 {
		m = kaddht.ModeServer
	}
	modeOpt := kaddht.Mode(m)
	options = append(options, modeOpt)

	// BootstrapPeers configures the bootstrapping nodes that we will connect to to seed
	// and refresh our Routing Table if it becomes empty.
	peerinfos := getPeerInfosFromConf()
	for _, peerinfo := range peerinfos {
		bootstrapPeer := kaddht.BootstrapPeers(*peerinfo)
		options = append(options, bootstrapPeer)
	}

	// Datastore configures the DHT to use the specified datastore.
	//
	// Defaults to an in-memory (temporary) map.
	dsname, _ := config.GetString("p2p.dht.datastore", "dispatch")
	switch dsname {
	case "dispatch":
		datastore := kaddht.Datastore(handler.NewDispatchDatastore())
		options = append(options, datastore)
	case "xorm":
		datastore := kaddht.Datastore(xorm.NewXormDatastore())
		options = append(options, datastore)
	case "redis":
		ds, _ := redis.NewDatastore(nil)
		datastore := kaddht.Datastore(ds)
		options = append(options, datastore)
	//case "bolt":
	//	ds, _ := bolt.NewDatastore("", "", true)
	//	datastore := kaddht.Datastore(ds)
	//	options = append(options, datastore)
	case "file":
		ds, _ := flatfs.CreateOrOpen("", nil, true)
		datastore := kaddht.Datastore(ds)
		options = append(options, datastore)
	case "leveldb":
		ds, _ := leveldb.NewDatastore("", nil)
		datastore := kaddht.Datastore(ds)
		options = append(options, datastore)
	case "badger":
		ds, _ := badger.NewDatastore("", nil)
		datastore := kaddht.Datastore(ds)
		options = append(options, datastore)
	case "sqlds":
		ds := sqlds.NewDatastore(nil, nil)
		datastore := kaddht.Datastore(ds)
		options = append(options, datastore)
	}
	//else if dsname == "crdt" {
	//	ds,_:=crdt.New(nil)
	//	datastore := kaddht.Datastore(ds)
	//	options = append(options, datastore)
	//}

	// ProtocolPrefix sets an application specific prefix to be attached to all DHT protocols. For example,
	// /myapp/kad/1.0.0 instead of /ipfs/kad/1.0.0. Prefix should be of the form /myapp.
	//
	// Defaults to dht.DefaultPrefix
	prefix, _ := config.GetString("p2p.dht.prefix", "/curltech")
	protocolPrefix := kaddht.ProtocolPrefix(protocol.ID(prefix))
	options = append(options, protocolPrefix)

	// NamespacedValidator adds a validator namespaced under `ns`. This option fails
	// if the DHT is not using a `record.NamespacedValidator` as its validator (it
	// uses one by default but this can be overridden with the `Validator` option).
	// Adding a namespaced validator without changing the `Validator` will result in
	// adding a new validator in addition to the default public key and IPNS validators.
	// The "pk" and "ipns" namespaces cannot be overridden here unless a new `Validator`
	// has been set first.
	//
	// Example: Given a validator registered as `NamespacedValidator("ipns",
	// myValidator)`, all records with keys starting with `/ipns/` will be validated
	// with `myValidator`.
	// 增加自己的ns数据的校验器，比如PeerEndpoint,PeerClient等
	// 注册datastore相关的服务，与相应的ns校验器绑定
	validator := kaddht.NamespacedValidator(ns.PeerEndpoint_Prefix, ns.PeerEndpointValidator{})
	options = append(options, validator)
	validator = kaddht.NamespacedValidator(ns.PeerClient_Prefix, ns.PeerClientValidator{})
	options = append(options, validator)
	validator = kaddht.NamespacedValidator(ns.PeerClient_Mobile_Prefix, ns.PeerClientValidator{})
	options = append(options, validator)
	validator = kaddht.NamespacedValidator(ns.ChainApp_Prefix, ns.PeerClientValidator{})
	options = append(options, validator)
	validator = kaddht.NamespacedValidator(ns.DataBlock_Prefix, ns.DataBlockValidator{})
	options = append(options, validator)
	validator = kaddht.NamespacedValidator(ns.DataBlock_Owner_Prefix, ns.DataBlockValidator{})
	options = append(options, validator)
	validator = kaddht.NamespacedValidator(ns.PeerTransaction_Src_Prefix, ns.PeerTransactionValidator{})
	options = append(options, validator)
	validator = kaddht.NamespacedValidator(ns.PeerTransaction_Target_Prefix, ns.PeerTransactionValidator{})
	options = append(options, validator)
	validator = kaddht.NamespacedValidator(ns.TransactionKey_Prefix, ns.TransactionKeyValidator{})
	options = append(options, validator)
	validator = kaddht.NamespacedValidator(ns.PeerTransaction_P2PChat_Prefix, ns.PeerTransactionValidator{})
	options = append(options, validator)
	validator = kaddht.NamespacedValidator(ns.PeerTransaction_GroupFile_Prefix, ns.PeerTransactionValidator{})
	options = append(options, validator)
	validator = kaddht.NamespacedValidator(ns.PeerTransaction_Channel_Prefix, ns.PeerTransactionValidator{})
	options = append(options, validator)
	validator = kaddht.NamespacedValidator(ns.PeerTransaction_ChannelArticle_Prefix, ns.PeerTransactionValidator{})
	options = append(options, validator)

	// RoutingTableRefreshPeriod sets the period for refreshing buckets in the
	// routing table. The DHT will refresh buckets every period by:
	//
	// 1. First searching for nearby peers to figure out how many buckets we should try to fill.
	// 1. Then searching for a random key in each bucket that hasn't been queried in
	//    the last refresh period.
	period, _ := config.GetInt("p2p.dht.period", 300)
	routingTableRefreshPeriod := kaddht.RoutingTableRefreshPeriod(time.Duration(period) * time.Second)
	options = append(options, routingTableRefreshPeriod)

	// DisableAutoRefresh completely disables 'auto-refresh' on the DHT routing
	// table. This means that we will neither refresh the routing table periodically
	// nor when the routing table size goes below the minimum threshold.
	//disableAutoRefresh, _ := config.GetBool("p2p.dht.disableAutoRefresh", true)
	//if disableAutoRefresh {
	//	disableAutoRefreshOption := kaddht.DisableAutoRefresh()
	//	options = append(options, disableAutoRefreshOption)
	//}

	// RoutingTableFilter sets a function that approves which peers may be added to the routing table. The host should
	// already have at least one connection to the peer under consideration.
	//disableRoutingTableFilter, _ := config.GetBool("p2p.dht.disableRoutingTableFilter", false)
	//if !disableRoutingTableFilter {
	//	routingTableFilterOption := dht.RoutingTableFilter(routingTableFilter)
	//	options = append(options, routingTableFilterOption)
	//}

	// RoutingTableLatencyTolerance sets the maximum acceptable latency for peers
	// in the routing table's cluster.
	//latency, _ := config.GetInt("p2p.dht.latency", 0)
	//routingTableLatencyTolerance := dht.RoutingTableLatencyTolerance(time.Duration(latency))
	//options = append(options, routingTableLatencyTolerance)
	// RoutingTableRefreshQueryTimeout sets the timeout for routing table refresh
	// queries.
	//timeout, _ := config.GetInt("p2p.dht.timeout", 0)
	//routingTableRefreshQueryTimeout := dht.RoutingTableRefreshQueryTimeout(time.Duration(timeout))
	//options = append(options, routingTableRefreshQueryTimeout)

	// Validator configures the DHT to use the specified validator.
	//
	// Defaults to a namespaced validator that can validate both public key (under the "pk"
	// namespace) and IPNS records (under the "ipns" namespace). Setting the validator
	// implies that the user wants to control the validators and therefore the default
	// public key and IPNS validators will not be added.
	//validator, _ := config.GetInt("p2p.dht.datastore", 0)
	//validatorOption := dht.Validator(validator)
	//options = append(options, validatorOption)

	// ProtocolExtension adds an application specific protocol to the DHT protocol. For example,
	// /ipfs/lan/kad/1.0.0 instead of /ipfs/kad/1.0.0. extension should be of the form /lan.
	//ext, _ := config.GetString("p2p.dht.ext", "")
	//if ext != "" {
	//	protocolExtension := dht.ProtocolExtension(protocol.ID(ext))
	//	options = append(options, protocolExtension)
	//}

	// V1ProtocolOverride overrides the protocolID used for /kad/1.0.0 with another. This is an
	// advanced feature, and should only be used to handle legacy networks that have not been
	// using protocolIDs of the form /app/kad/1.0.0.
	//
	// This option will override and ignore the ProtocolPrefix and ProtocolExtension options
	//proto, _ := config.GetString("p2p.dht.proto", "")
	//if proto != "" {
	//	v1protocolOverride := dht.V1ProtocolOverride(protocol.ID(proto))
	//	options = append(options, v1protocolOverride)
	//}

	// BucketSize configures the bucket size (k in the Kademlia paper) of the routing table.
	//
	// The default value is 20.
	bucketSize, _ := config.GetInt("p2p.dht.bucketSize", 20)
	bucketSizeOption := kaddht.BucketSize(bucketSize)
	options = append(options, bucketSizeOption)

	// Concurrency configures the number of concurrent requests (alpha in the Kademlia paper) for a given query path.
	//
	// The default value is 10.
	alpha, _ := config.GetInt("p2p.dht.alpha", 10)
	concurrency := kaddht.Concurrency(alpha)
	options = append(options, concurrency)
	// Resiliency configures the number of peers closest to a target that must have responded in order for a given query
	// path to complete.
	//
	// The default value is 3.
	beta, _ := config.GetInt("p2p.dht.beta", 3)
	resiliency := kaddht.Resiliency(beta)
	options = append(options, resiliency)
	// MaxRecordAge specifies the maximum time that any node will hold onto a record ("PutValue record")
	// from the time its received. This does not apply to any other forms of validity that
	// the record may contain.
	// For example, a record may contain an ipns entry with an EOL saying its valid
	// until the year 2020 (a great time in the future). For that record to stick around
	// it must be rebroadcasted more frequently than once every 'MaxRecordAge'
	//maxAge, _ := config.GetInt("p2p.dht.maxAge", 0)
	//maxRecordAge := dht.MaxRecordAge(time.Duration(maxAge))
	//options = append(options, maxRecordAge)

	// DisableProviders disables storing and retrieving provider records.
	//
	// Defaults to enabled.
	//
	// WARNING: do not change this unless you're using a forked DHT (i.e., a private
	// network and/or distinct DHT protocols with the `Protocols` option).
	//disableProviders, _ := config.GetBool("p2p.dht.disableProviders", true)
	//if disableProviders {
	//	disableProvidersOption := dht.DisableProviders()
	//	options = append(options, disableProvidersOption)
	//}

	// DisableValues disables storing and retrieving value records (including
	// public keys).
	//
	// Defaults to enabled.
	//
	// WARNING: do not change this unless you're using a forked DHT (i.e., a private
	// network and/or distinct DHT protocols with the `Protocols` option).
	//disableValues, _ := config.GetBool("p2p.dht.disableValues", true)
	//if disableValues {
	//	disableValuesOption := dht.DisableValues()
	//	options = append(options, disableValuesOption)
	//}

	// ProvidersOptions are options passed directly to the provider manager.
	//
	// The provider manager adds and gets provider records from the datastore, cahing
	// them in between. These options are passed to the provider manager allowing
	// customisation of things like the GC interval and cache implementation.
	//providersOption, _ := config.GetBool("p2p.dht.providersOption", false)
	//if !providersOption {
	//	providersOptions := dht.ProvidersOptions(providersOption)
	//	options = append(options, providersOptions)
	//}

	// QueryFilter sets a function that approves which peers may be dialed in a query
	//queryFilter, _ := config.GetBool("p2p.dht.queryFilter", false)
	//if !queryFilter {
	//	queryFilterOption := dht.QueryFilter(queryFilter)
	//	options = append(options, queryFilterOption)
	//}

	// RoutingTablePeerDiversityFilter configures the implementation of the `PeerIPGroupFilter` that will be used
	// to construct the diversity filter for the Routing Table.
	// Please see the docs for `peerdiversity.PeerIPGroupFilter` AND `peerdiversity.Filter` for more details.
	//disablePeerIPGroupFilter, _ := config.GetBool("p2p.dht.disablePeerIPGroupFilter", true)
	//if !disablePeerIPGroupFilter {
	//	routingTablePeerDiversityFilter := dht.RoutingTablePeerDiversityFilter(disablePeerIPGroupFilter)
	//	options = append(options, routingTablePeerDiversityFilter)
	//}

	return options
}

func routingTableFilter(dht *kaddht.IpfsDHT, conns []network.Conn) bool {

	return true
}

func PeerAdded(id peer.ID) {
	peerId := id.Pretty()
	logger.Sugar.Debugf("PeerEndpointDHT.RoutingTable add peer: %v", peerId)
	// PeerEndPointAction
	response, err := dht.PeerEndPointAction.PeerEndPoint(peerId)
	if response == msgtype.OK {
		logger.Sugar.Infof("successfully PeerEndPoint: %v", peerId)
	} else {
		logger.Sugar.Errorf("failed to PeerEndPoint: %v, err: %v", peerId, err)
	}
	// 更改状态
	peerEndPoints, err := service.GetPeerEndpointService().GetLocal(peerId)
	if err != nil {
		logger.Sugar.Errorf("failed to GetLocal PeerEndPoint: %v, err: %v", peerId, err)
	} else {
		if peerEndPoints != nil && len(peerEndPoints) > 0 {
			currentTime := time.Now()
			for _, peerEndPoint := range peerEndPoints {
				peerEndPoint.ActiveStatus = entity.ActiveStatus_Up
				peerEndPoint.LastUpdateTime = &currentTime
				err := service.GetPeerEndpointService().PutLocal(peerEndPoint)
				if err != nil {
					logger.Sugar.Errorf("failed to PutLocal PeerEndPoint: %v, err: %v", peerId, err)
				}
			}
		}
	}
}

func PeerRemoved(id peer.ID) {
	peerId := id.Pretty()
	logger.Sugar.Debugf("PeerEndpointDHT.RoutingTable remove peer: %v", peerId)
	// 更改状态
	peerEndPoints, err := service.GetPeerEndpointService().GetLocal(peerId)
	if err != nil {
		logger.Sugar.Errorf("failed to GetLocal PeerEndPoint: %v, err: %v", peerId, err)
	} else {
		if peerEndPoints != nil && len(peerEndPoints) > 0 {
			currentTime := time.Now()
			for _, peerEndPoint := range peerEndPoints {
				peerEndPoint.ActiveStatus = entity.ActiveStatus_Down
				peerEndPoint.LastUpdateTime = &currentTime
				err := service.GetPeerEndpointService().PutLocal(peerEndPoint)
				if err != nil {
					logger.Sugar.Errorf("failed to PutLocal PeerEndPoint: %v, err: %v", peerId, err)
				}
			}
		}
	}
}
