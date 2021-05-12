package libp2p

import (
	"context"
	"fmt"
	openpgpcrypto "github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/crypto"
	"github.com/curltech/go-colla-core/crypto/openpgp"
	"github.com/curltech/go-colla-core/crypto/std"
	entity2 "github.com/curltech/go-colla-core/entity"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/pipe/handler"
	"github.com/curltech/go-colla-node/libp2p/pubsub"
	"github.com/curltech/go-colla-node/libp2p/routingtable"
	"github.com/curltech/go-colla-node/libp2p/util"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	"github.com/curltech/go-colla-node/transport/websocket/wss"
	"github.com/libp2p/go-libp2p"
	autonat "github.com/libp2p/go-libp2p-autonat"
	relay "github.com/libp2p/go-libp2p-circuit"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	libp2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-core/routing"
	dht2 "github.com/libp2p/go-libp2p-kad-dht"
	mplex "github.com/libp2p/go-libp2p-mplex"
	noise "github.com/libp2p/go-libp2p-noise"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	secio "github.com/libp2p/go-libp2p-secio"
	libp2ptls "github.com/libp2p/go-libp2p-tls"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	ws "github.com/libp2p/go-ws-transport"
	ma "github.com/multiformats/go-multiaddr"
	"time"
)

/**
golang.org/x/crypto的库需要替换成libp2p的定制版本
*/

/**
根据配置的协议编号自定义流协议，其他peer连接自己的时候，用于在节点间接收和发送数据
*/
func chainProtocolStream() protocol.ID {
	if config.P2pParams.ChainProtocolID == "" {
		panic("NoChainProtocolID")
	}
	global.Global.ChainProtocolID = protocol.ID(config.P2pParams.ChainProtocolID)
	handler.ProtocolStream(global.Global.ChainProtocolID)

	return global.Global.ChainProtocolID
}

func listenAddr() libp2p.Option {
	if len(config.Libp2pParams.Addrs) == 0 {
		//没有单独激活websocket，所以激活tcp
		if !config.Libp2pParams.EnableWebsocket {
			tcpAddr := fmt.Sprintf(global.DefaultAddrFormat, config.Libp2pParams.Addr, config.Libp2pParams.Port)
			config.Libp2pParams.Addrs = append(config.Libp2pParams.Addrs, tcpAddr)
		}
		//激活websocket或者wss
		if config.Libp2pParams.EnableWss {
			wssAddr := fmt.Sprintf(global.DefaultWssAddrFormat, config.Libp2pParams.Addr, config.Libp2pParams.WssPort)
			config.Libp2pParams.Addrs = append(config.Libp2pParams.Addrs, wssAddr)
		} else {
			wsAddr := fmt.Sprintf(global.DefaultWsAddrFormat, config.Libp2pParams.Addr, config.Libp2pParams.WsPort)
			config.Libp2pParams.Addrs = append(config.Libp2pParams.Addrs, wsAddr)
		}

		//if config.Libp2pParams.EnableWebrtcStar {
		//	webrtcAddr := fmt.Sprintf(global.DefaultWebrtcstarAddrFormat, config.Libp2pParams.Addr, config.Libp2pParams.WebrtcStarPort)
		//	config.Libp2pParams.Addrs = append(config.Libp2pParams.Addrs, webrtcAddr)
		//}
	}

	listenAddrOption := libp2p.ListenAddrStrings(config.Libp2pParams.Addrs...)

	return listenAddrOption
}

func p2pOptions() []libp2p.Option {
	options := []libp2p.Option{}
	// support TLS connections
	if config.Libp2pParams.EnableTls {
		tlsOption := libp2p.Security(libp2ptls.ID, libp2ptls.New)
		options = append(options, tlsOption)
		logger.Sugar.Debugf("start tls option")
	}

	// support noise connections
	if config.Libp2pParams.EnableNoise {
		noiseOption := libp2p.Security(noise.ID, noise.New)
		options = append(options, noiseOption)
		logger.Sugar.Debugf("start noise option")
	}

	// support secio connections
	if config.Libp2pParams.EnableSecio {
		secioOption := libp2p.Security(secio.ID, secio.New)
		options = append(options, secioOption)
		logger.Sugar.Debugf("start secio option")
	}

	// support QUIC - experimental
	if config.Libp2pParams.EnableQuic {
		quicOption := libp2p.Transport(libp2pquic.NewTransport)
		options = append(options, quicOption)
		logger.Sugar.Debugf("start quic option")
	}

	// Let's prevent our peer from having too many
	// connections by attaching a connection manager.
	basicConnMgr := connmgr.NewConnManager(
		config.Libp2pParams.LowWater,                               // Lowwater
		config.Libp2pParams.HighWater,                              // HighWater,
		time.Duration(config.Libp2pParams.GracePeriod)*time.Minute, // GracePeriod
	)
	monitorConnMgr := NewConnManager(basicConnMgr)
	global.Global.ConnectionManager = monitorConnMgr
	connectionManagerOption := libp2p.ConnectionManager(global.Global.ConnectionManager)
	options = append(options, connectionManagerOption)
	logger.Sugar.Debugf("start ConnectionManager option")

	//是否用缺省的ping服务
	defaultPing, _ := config.GetBool("p2p.dht.defaultPing", true)
	if !defaultPing {
		pingOption := libp2p.Ping(defaultPing)
		options = append(options, pingOption)
		ps := &ping.PingService{Host: global.Global.Host}
		global.Global.Host.SetStreamHandler(ping.ID, ps.PingHandler)
	}

	// Attempt to open ports using uPNP for NATed hosts.
	if config.Libp2pParams.EnableNatPortMap {
		natPortMap := libp2p.NATPortMap()
		options = append(options, natPortMap)
		logger.Sugar.Debugf("start NATPortMap option")
	}

	// Attempt to open EnableNATService，实施回拨确认机制
	if config.Libp2pParams.EnableNATService {
		enableNATService := libp2p.EnableNATService()
		//libp2p.AutoNATServiceRateLimit()
		options = append(options, enableNATService)
		logger.Sugar.Debugf("start EnableNATService option")
	}

	//if config.Libp2pParams.NATManager {
	//	natManager := libp2p.NATManager(nil)
	//	options = append(options, natManager)
	//	logger.Sugar.Infof("start NATManager option")
	//}

	// Attempt to open ConnectionGater，控制连接的安全性
	if config.Libp2pParams.ConnectionGater {
		connectionGater := libp2p.ConnectionGater(nil)
		options = append(options, connectionGater)
		logger.Sugar.Debugf("start ConnectionGater option")
	}

	if config.Libp2pParams.ForceReachabilityPublic {
		forceReachabilityPublic := libp2p.ForceReachabilityPublic()
		options = append(options, forceReachabilityPublic)
		logger.Sugar.Debugf("start ForceReachabilityPublic option")
	}

	if config.Libp2pParams.ForceReachabilityPrivate {
		forceReachabilityPrivate := libp2p.ForceReachabilityPrivate()
		options = append(options, forceReachabilityPrivate)
		logger.Sugar.Debugf("start ForceReachabilityPrivate option")
	}

	// Let this host use the DHT to find other hosts
	if config.Libp2pParams.EnableRouting {
		routing := libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			return global.Global.PeerEndpointDHT, nil
		})
		options = append(options, routing)
		logger.Sugar.Debugf("start Routing option")
	}

	if config.Libp2pParams.EnableRelay {
		relay := libp2p.EnableRelay(relay.OptHop)
		options = append(options, relay)
		logger.Sugar.Debugf("start EnableRelay option")
	}

	// 1. When this libp2p node is configured to act as a relay "hop"
	//    (circuit.OptHop is passed to EnableRelay), this node will advertise itself
	//    as a public relay using the provided routing system.
	// 2. When this libp2p node is _not_ configured as a relay "hop", it will
	//    automatically detect if it is unreachable (e.g., behind a NAT). If so, it will
	//    find, configure, and announce a set of public relays.
	if config.Libp2pParams.EnableAutoRelay {
		relay := libp2p.EnableAutoRelay()
		options = append(options, relay)
		logger.Sugar.Debugf("start EnableAutoRelay option")
	}

	/**
	手工指定外部的地址和端口
	*/
	if config.Libp2pParams.EnableAddressFactory {
		addr := config.Libp2pParams.ExternalAddr
		var httpMultiAddr, wsMultiAddr ma.Multiaddr
		var httpErr, wsErr error
		if addr == "" {
			logger.Sugar.Errorf("Server Addr(External IP) not defined, Peers might not be able to resolve this node if behind NAT\n")
		} else {
			/**
			这里应该支持多个，方便支持多种协议
			*/
			httpMultiAddr, httpErr = ma.NewMultiaddr(fmt.Sprintf(global.DefaultDnsAddrFormat, addr, config.Libp2pParams.ExternalPort))
			if httpErr != nil {
				logger.Sugar.Errorf("Error creating httpMultiAddr: %v\n", httpErr)
			}
			wsMultiAddr, wsErr = ma.NewMultiaddr(fmt.Sprintf(global.DefaultDnsWsAddrFormat, addr, config.Libp2pParams.ExternalWsPort))
			if wsErr != nil {
				logger.Sugar.Errorf("Error creating wsMultiAddr: %v\n", wsErr)
			}
			if httpErr == nil || wsErr == nil {
				addressFactory := func(addrs []ma.Multiaddr) []ma.Multiaddr {
					if httpErr == nil && httpMultiAddr != nil {
						addrs = append(addrs, httpMultiAddr)
					}
					if wsErr == nil && wsMultiAddr != nil {
						addrs = append(addrs, wsMultiAddr)
					}
					return addrs
				}
				addressFactoryOption := libp2p.AddrsFactory(addressFactory)
				options = append(options, addressFactoryOption)
				logger.Sugar.Debugf("start addressFactory option")
			}
		}
	}

	//是否单独激活websocket
	if config.Libp2pParams.EnableWebsocket {
		// libp2p wss, config.TlsParams.Mode="cert" or config.TlsParams.Domain
		if config.Libp2pParams.EnableWss {
			wssOption := libp2p.Transport(wss.New)
			options = append(options, wssOption)
			logger.Sugar.Debugf("start EnableWss option")
		} else {
			wsOption := libp2p.Transport(ws.New)
			options = append(options, wsOption)
			logger.Sugar.Debugf("start EnableWebsocket option")
		}
	}

	// support any other default transports (TCP)
	//defaultTransport := libp2p.DefaultTransports
	//options = append(options, defaultTransport)
	//libp2p.Transport(tcp.NewTCPTransport)
	//options = append(options, libp2p.Muxer("/yamux/1.0.0", yamux.DefaultConfig()))
	options = append(options, libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport))

	// 嵌入webrtc2.0的支持，和js端可以兼容，但只能单独使用
	// mplex "github.com/whyrusleeping/go-smux-multiplex"需要make deps
	///ip4/127.0.0.1/tcp/9090/http/p2p-webrtc-direct
	//if config.Libp2pParams.EnableWebrtc {
	//	transport := direct.NewTransport(
	//		webrtc.Configuration{},
	//		new(mplex.Transport),
	//	)
	//	wrOption := libp2p.Transport(transport)
	//	options = append(options, wrOption)
	//	logger.Sugar.Infof("start EnableWebrtc option")
	//}

	return options
}

func autoNat() {
	// If you want to help other peers to figure out if they are behind
	// NATs, you can launch the server-side of AutoNAT too (AutoRelay
	// already runs the client)
	if config.Libp2pParams.EnableAutoNat {
		//_ = autonat.EnableService(nil)
		_, err := autonat.New(global.Global.Context, global.Global.Host)
		if err != nil {
			panic(err)
		}
	}
}

/**
获取自己节点的记录，如果没有就创建一个，返回私钥用于启动节点
*/
func getMyselfPeer() (libp2pcrypto.PrivKey, *entity.MyselfPeer) {
	myself := entity.MyselfPeer{}
	myself.Status = entity2.EntityStatus_Effective
	password := config.ServerParams.Password
	if password == "" {
		panic("NoPassword")
	}
	found := service.GetMyselfPeerService().Get(&myself, false, "", "")
	if found {
		privateKey := std.DecodeBase64(myself.PeerPrivateKey)
		priv, err := libp2pcrypto.UnmarshalPrivateKey(privateKey)
		if err != nil {
			panic(err)
		}
		global.Global.PeerPrivateKey = priv
		global.Global.PeerPublicKey = priv.GetPublic()

		openpgpPrivateKey := std.DecodeBase64(myself.PrivateKey)
		openpgpPriv, err := openpgp.LoadPrivateKey(openpgpPrivateKey, password)
		if err != nil {
			panic(err)
		}
		global.Global.PrivateKey = openpgpPriv

		openpgpPublicKey := std.DecodeBase64(myself.PublicKey)
		openpgpPub, err := openpgp.LoadPublicKey(openpgpPublicKey)
		if err != nil {
			panic(err)
		}
		global.Global.PublicKey = openpgpPub

		return priv, &myself
	} else {
		name := config.ServerParams.Name
		if name == "" {
			panic("NoServerName")
		}
		email := config.ServerParams.Email
		if email == "" {
			panic("NoEmail")
		}
		myself = entity.MyselfPeer{}
		myself.Name = name
		myself.Email = email
		/**
		peerId对应的密钥对
		*/
		priv, _, _ := libp2pcrypto.GenerateKeyPair(libp2pcrypto.Ed25519, 0)
		global.Global.PeerPrivateKey = priv
		global.Global.PeerPublicKey = priv.GetPublic()
		bs, err := priv.Bytes()
		if err != nil {
			panic(err)
		}
		myself.PeerPrivateKey = std.EncodeBase64(bs)
		logger.Sugar.Debugf("PeerPrivateKey length: %v", len(myself.PeerPrivateKey))
		bs, err = priv.GetPublic().Bytes()
		if err != nil {
			panic(err)
		}
		myself.PeerPublicKey = std.EncodeBase64(bs)
		logger.Sugar.Debugf("PeerPublicKey length: %v", len(myself.PeerPublicKey))
		/**
		加密对应的密钥对openpgp
		*/
		privKey := openpgp.GenerateKeyPair(crypto.KeyPairType_Ed25519, []byte(password), false, name, email)
		privateKey := privKey.(*openpgpcrypto.Key)
		myself.PrivateKey = std.EncodeBase64(openpgp.BytePrivateKey(privateKey, []byte(password)))
		global.Global.PrivateKey = privateKey
		logger.Sugar.Debugf("PrivateKey length: %v", len(myself.PrivateKey))

		publicKey := openpgp.GetPublicKey(privateKey)
		global.Global.PublicKey = publicKey
		bs = openpgp.BytePublicKey(publicKey)
		myself.PublicKey = std.EncodeBase64(bs)
		logger.Sugar.Debugf("PublicKey length: %v", len(myself.PublicKey))

		myself.SecurityContext, _ = message.TextMarshal(crypto.SecurityContext{
			Protocol:    crypto.Protocol_openpgp,
			KeyPairType: crypto.KeyPairType_Ed25519,
		})
		currentTime := time.Now()
		myself.LastUpdateTime = &currentTime

		return priv, &myself
	}
}

/**
启动本机作为一个p2p节点主机
*/
func Start() {
	// 背景上下文
	global.Global.Context = context.Background()

	//1.设置监听地址
	listenAddrOption := listenAddr()
	//2.获取或者创建本节点的私钥
	priv, myself := getMyselfPeer()
	//3.根据私钥创建本机唯一的ID
	idOption := libp2p.Identity(priv)
	//4.启动自己的p2p节点
	var err error
	options := p2pOptions()
	options = append(options, listenAddrOption)
	options = append(options, idOption)

	//if config.Libp2pParams.EnableWebrtcStar {
	//	//starOption := webrtcstar.StarTransportOption(global.Global.PeerPrivateKey.GetPublic())
	//	//options = append(options, starOption...)
	//	logger.Sugar.Infof("start EnableWebrtcStar option")
	//}

	global.Global.Host, err = libp2p.New(global.Global.Context, options...)
	if err != nil {
		panic(err)
	}
	logger.Sugar.Infof("Host created, Addrs: %v", global.Global.Host.Addrs())
	autoNat()
	//5.启动成功后更新自己节点的信息到数据库
	upsertMyselfPeer(priv, myself)
	//6.自定义数据传输的流协议
	chainProtocolStream()
	//7.启动dht，并配置dhtOption
	NewPeerEndpointDHT(dhtOptions())
	//8.手工配置发现节点，只设置dhtOption的Bootstrap也能工作，就是慢点，需要等待刷新周期
	//这一步代码里会主动连接，所以比较快，所以装载PeerEndpoint比较合适
	Bootstrap()
	//把自己的信息写到分布式网络，但是不写其他节点通过GetValue也能找到
	dht.PeerEndpointDHT.PutMyself()
	//9.设置其他的路由发现方式，发现不能打开，会因为连接不上删除节点
	//routingDiscovery()
	//10.局域网可设置mdns路由发现方式
	//mdns()

	//handler.SetNetNotifiee()

	//ipfs.Start()

	global.Global.PeerEndpointDHT.RoutingTable().Print()
	global.Print()

	select {}
}

func connect(peer *peer.AddrInfo) {
	err := global.Global.Host.Connect(global.Global.Context, *peer)
	if err != nil {
		logger.Sugar.Errorf("cannot connect peer: %v", peer)
	}
}

func subcrible() *pubsub.PubsubTopic {
	topic, err := pubsub.Subscribe(config.Libp2pParams.Topic)
	if err != nil {
		logger.Sugar.Errorf("cannot subscribe topic: %v, err: %v", config.Libp2pParams.Topic, err)
	}

	return topic
}

func upsertMyselfPeer(priv libp2pcrypto.PrivKey, myself *entity.MyselfPeer) (needUpdate bool) {
	needUpdate = false
	id := myself.Id

	if config.ServerParams.ExternalAddr == "" {
		logger.Sugar.Errorf("NoHttpServerExternalAddr")
	}
	if config.ServerParams.ExternalPort == "" {
		logger.Sugar.Errorf("NoHttpServerExternalPort")
	}
	var discoveryAddress string = ""
	//discoveryAddress = config.ServerParams.ExternalAddr + ":" + config.ServerParams.ExternalPort
	addr := config.Libp2pParams.ExternalAddr
	if addr == "" {
		logger.Sugar.Errorf("Server Addr(External IP) not defined, Peers might not be able to resolve this node if behind NAT\n")
	} else {
		wssMultiAddr, wssErr := ma.NewMultiaddr(fmt.Sprintf(global.DefaultDnsWssAddrFormat, addr, config.Libp2pParams.ExternalWssPort))
		if wssErr != nil {
			logger.Sugar.Errorf("Error creating wssMultiAddr: %v\n", wssErr)
		} else if wssMultiAddr != nil {
			if myself.PeerId == "" {
				myself.PeerId = global.Global.Host.ID().String()
			}
			discoveryAddress = fmt.Sprintf(global.GeneralP2pAddrFormat, wssMultiAddr.String(), myself.PeerId)
		}
	}

	if id == 0 { //新的
		needUpdate = true
		myself.Status = entity2.EntityStatus_Effective
		myself.PeerId = global.Global.Host.ID().String()
		var err error
		var saddrs = util.ToString(global.Global.Host.Addrs())
		myself.Address, err = message.TextMarshal(saddrs)
		if err != nil {
			panic(err)
		}
		myself.DiscoveryAddress = discoveryAddress
		affected := service.GetMyselfPeerService().Insert(myself)
		if affected == 0 {
			panic("NoInsert")
		}
	} else {
		if myself.PeerId != global.Global.Host.ID().String() {
			needUpdate = true
			myself.PeerId = global.Global.Host.ID().String()
			logger.Sugar.Infof("PeerId changed")
		}
		var saddrs = util.ToString(global.Global.Host.Addrs())
		addrs, err := message.TextMarshal(saddrs)
		if err != nil {
			panic(err)
		}
		if myself.Address != addrs {
			needUpdate = true
			myself.Address = addrs
			logger.Sugar.Infof("Address changed")
		}
		if myself.DiscoveryAddress != discoveryAddress {
			needUpdate = true
			myself.DiscoveryAddress = discoveryAddress
			logger.Sugar.Infof("DiscoveryAddress changed")
		}
		bs, err := priv.GetPublic().Bytes()
		if err != nil {
			panic(err)
		}
		if myself.PeerPublicKey != std.EncodeBase64(bs) {
			needUpdate = true
			myself.PeerPublicKey = std.EncodeBase64(bs)
			logger.Sugar.Infof("PublicKey changed")
		}
		name := config.ServerParams.Name
		if name == "" {
			panic("NoServerName")
		}
		if myself.Name != name {
			needUpdate = true
			myself.Name = name
			logger.Sugar.Infof("Name changed")
		}
		email := config.ServerParams.Email
		if email == "" {
			panic("NoEmail")
		}
		if myself.Email != email {
			needUpdate = true
			myself.Email = email
			logger.Sugar.Infof("Email changed")
		}

		if needUpdate {
			affected := service.GetMyselfPeerService().Update([]interface{}{myself}, nil, "")
			if affected == 0 {
				panic("NoUpdate")
			}
		}
	}
	global.Global.PeerId = peer.ID(myself.PeerId)
	global.Global.MyselfPeer = myself

	return needUpdate
}

/**
为了节点发现启动DHT
*/
func NewPeerEndpointDHT(options []dht2.Option) *dht.PeerEntityDHT {
	var err error
	global.Global.PeerEndpointDHT, err = dht2.New(global.Global.Context, global.Global.Host, options...)
	if err != nil {
		panic(err)
	}
	dht.PeerEndpointDHT.DHT = global.Global.PeerEndpointDHT

	dht.PeerEndpointDHT.RoutingTable = &routingtable.PeerEntityRoutingTable{
		global.Global.PeerEndpointDHT.RoutingTable(),
	}
	dht.PeerEndpointDHT.RoutingTable.PeerAdded(PeerAdded)
	/**
	当一个节点被从路由表中移出的时候，将peerendpoint的对应记录设置为down
	*/
	dht.PeerEndpointDHT.RoutingTable.PeerRemoved(PeerRemoved)

	return dht.PeerEndpointDHT
}
