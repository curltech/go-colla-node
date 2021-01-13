package libp2p

import (
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/pipe/handler"
	"github.com/kataras/golog"
	discovery "github.com/libp2p/go-libp2p-discovery"
)

/**
路由发现，设置约会地点
*/
func routingDiscovery() *discovery.RoutingDiscovery {
	golog.Infof("Announcing ourselves...")
	routingDiscovery := discovery.NewRoutingDiscovery(global.Global.PeerEndpointDHT)
	var err error
	global.Global.Rendezvous, err = config.GetString("p2p.rendezvous", "curltech.io")
	if err != nil {
		golog.Errorf("%v", err)
	}
	//广播约会地
	discovery.Advertise(global.Global.Context, routingDiscovery, global.Global.Rendezvous)
	golog.Infof("Successfully announced:%v", global.Global.Rendezvous)

	return routingDiscovery
}

func FindPeers(routingDiscovery *discovery.RoutingDiscovery) {
	// 搜索约会地点的其他人
	golog.Infof("Searching for other peers...")
	peerChan, err := routingDiscovery.FindPeers(global.Global.Context, global.Global.Rendezvous)
	if err != nil {
		panic(err)
	}
	// 遍历每一个发现的peer节点
	for peer := range peerChan {
		if peer.ID == global.Global.Host.ID() {
			continue
		}
		golog.Infof("Found peer:%v", peer)
		golog.Infof("Connecting to:%v", peer)
		_, err := handler.GetPipePool().GetRequestPipe(string(peer.ID), string(global.Global.ChainProtocolID))
		if err != nil {
			golog.Infof("GetRequestPipe failed:%v", err)
			continue
		}

		golog.Infof("Connected to:%v", peer)
	}
}
