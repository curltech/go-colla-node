package libp2p

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	discovery2 "github.com/libp2p/go-libp2p/p2p/discovery"
	"time"
)

type discoveryNotifee struct {
	PeerChan chan peer.AddrInfo
}

/**
新peer发现的时候被调用
*/
func (notifee *discoveryNotifee) HandlePeerFound(peer peer.AddrInfo) {
	notifee.PeerChan <- peer
	if global.Global.Host.Network().Connectedness(peer.ID) != network.Connected {
		logger.Sugar.Infof("Found %s!\n", peer.ID.ShortString())
		global.Global.Host.Connect(global.Global.Context, peer)
	}
}

/**
通过mdns协议发现新节点
peerChan := mdns(ctx, host, group)
peer := <-peerChan // will block untill we discover a peer
fmt.Println("Found peer:", peer, ", connecting")
*/
func mdns() chan peer.AddrInfo {
	// An hour might be a long long period in practical applications. But this is fine for us
	ser, err := discovery2.NewMdnsService(global.Global.Context, global.Global.Host, time.Hour, global.Global.Rendezvous)
	if err != nil {
		logger.Sugar.Errorf("%v", err)
	}

	//register with service so that we get notified about peer discovery
	n := &discoveryNotifee{}
	n.PeerChan = make(chan peer.AddrInfo)

	ser.RegisterNotifee(n)

	return n.PeerChan
}

func GetFoundPeer(peerChan chan peer.AddrInfo) peer.AddrInfo {
	peer := <-peerChan // will block untill we discover a peer
	logger.Sugar.Infof("Found peer:", peer, ", connecting")

	if err := global.Global.Host.Connect(global.Global.Context, peer); err != nil {
		logger.Sugar.Infof("Connection failed:", err)
	}

	return peer
}
