package libp2p

import (
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/pipe/handler"
	discovery "github.com/libp2p/go-libp2p-discovery"
)

/**
路由发现，设置约会地点
*/
func routingDiscovery() *discovery.RoutingDiscovery {
	logger.Sugar.Infof("Announcing ourselves...")
	routingDiscovery := discovery.NewRoutingDiscovery(global.Global.PeerEndpointDHT)
	var err error
	global.Global.Rendezvous, err = config.GetString("p2p.rendezvous", "curltech.io")
	if err != nil {
		logger.Sugar.Errorf("%v", err)
	}
	//广播约会地
	discovery.Advertise(global.Global.Context, routingDiscovery, global.Global.Rendezvous)
	logger.Sugar.Infof("Successfully announced:%v", global.Global.Rendezvous)

	return routingDiscovery
}

func FindPeers(routingDiscovery *discovery.RoutingDiscovery) {
	// 搜索约会地点的其他人
	logger.Sugar.Infof("Searching for other peers...")
	peerChan, err := routingDiscovery.FindPeers(global.Global.Context, global.Global.Rendezvous)
	if err != nil {
		panic(err)
	}
	// 遍历每一个发现的peer节点
	for peer := range peerChan {
		if peer.ID == global.Global.Host.ID() {
			continue
		}
		logger.Sugar.Infof("Found peer:%v", peer)
		logger.Sugar.Infof("Connecting to:%v", peer)
		p := handler.GetPipePool().GetRequestPipe(string(peer.ID), string(global.Global.ChainProtocolID))
		if p == nil {
			logger.Sugar.Infof("NoPipe, failed to connect to:%v", peer)
			continue
		}

		logger.Sugar.Infof("Connected to:%v", peer)
	}
}
