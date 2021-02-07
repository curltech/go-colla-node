package libp2p

import (
	"fmt"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/entity"
	"github.com/curltech/go-colla-core/logger"
	dht2 "github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/util"
	dhtentity "github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	//"github.com/curltech/go-colla-node/p2p/chain/action/dht"
	//"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/libp2p/go-libp2p-core/peer"
	"strings"
)

/**
获取启动节点清单，从配置文件读取
*/
func getPeerInfosFromConf() []*peer.AddrInfo {
	bootstraps, err := config.GetString("libp2p.dht.bootstraps")
	if err != nil {
		logger.Sugar.Errorf("config bootstraps error")
	}
	addrInfos := make([]*peer.AddrInfo, 0)
	bootstrapPeers := strings.Split(bootstraps, ",")
	if len(bootstrapPeers) > 0 {
		addrs := util.ToMultiaddr(bootstrapPeers)
		addrInfos = util.MultiaddrToAddInfo(addrs)
		logger.Sugar.Infof("config bootstraps:%v", addrInfos)

		return addrInfos
	}

	return addrInfos
}

func getPeerInfosFromStore() []*peer.AddrInfo {
	peerinfos := make([]*peer.AddrInfo, 0)
	peerEndpoints := make([]*dhtentity.PeerEndpoint, 0)
	peerEndpoint := dhtentity.PeerEndpoint{}
	peerEndpoint.Status = entity.EntityStatus_Effective
	//peerEndpoint.ActiveStatus = dhtentity.ActiveStatus_Up
	service.GetPeerEndpointService().Find(&peerEndpoints, &peerEndpoint, "", 0, 0, "")
	if len(peerEndpoints) > 0 {
		for _, peerEndpoint := range peerEndpoints {
			addrInfos, err := util.ToAddInfos(peerEndpoint.PeerId, peerEndpoint.Address)
			if err != nil {
				continue
			}
			for _, addrInfo := range addrInfos {
				peerinfos = append(peerinfos, addrInfo)
			}
		}
		logger.Sugar.Infof("store bootstraps:%v", peerinfos)
	}

	return peerinfos
}

func GetPeerInfos() []*peer.AddrInfo {
	peerinfos := make([]*peer.AddrInfo, 0)
	peerInfoPtrs := getPeerInfosFromConf()
	if len(peerInfoPtrs) > 0 {
		for _, peerInfoPtr := range peerInfoPtrs {
			isLocalhost := false
			for _, addr := range peerInfoPtr.Addrs {
				if strings.Contains(addr.String(), "localhost") || strings.Contains(addr.String(), "127.0.0.1") {
					isLocalhost = true
					continue
				}
			}
			if !isLocalhost {
				peerinfos = append(peerinfos, peerInfoPtr)
			}
		}
	}
	peerInfoPtrs = getPeerInfosFromStore()
	if len(peerInfoPtrs) > 0 {
		for _, peerInfoPtr := range peerInfoPtrs {
			peerinfos = append(peerinfos, peerInfoPtr)
		}
	}

	return peerinfos
}

/**
后台线程，检查在数据库中是否存在，如果不存在，连接，寻找节点，加入dht
如果连接不上，调度连接根据配置的启动节点进行刷新
*/
func Bootstrap() error {
	bootstrapPeerInfos := GetPeerInfos()
	for _, peerinfo := range bootstrapPeerInfos {
		if peerinfo.ID == global.Global.Host.ID() {
			continue
		}
		peerInfo := *peerinfo
		go func() {
			peerId := fmt.Sprintf(global.GeneralP2pAddrFormat, peerInfo.Addrs[0], peerInfo.ID)
			if strings.Contains(peerId, "127.0.0.1") || strings.Contains(peerId, "localhost") {
				return
			}
			logger.Sugar.Infof("will connect %v", peerId)
			if err := global.Global.Host.Connect(global.Global.Context, peerInfo); err != nil {
				logger.Sugar.Errorf("Failed to connect to bootstrap node: %v, err: %v", peerInfo, err)
			} else {
				logger.Sugar.Infof("Successfully connect to bootstrap node: %v", peerInfo)
			}
			dht2.PingPing(peerInfo.ID)
			/*response, err := dht.PingAction.Ping(peerId, peerId)
			if response == msgtype.OK {
				logger.Sugar.Infof("Successfully ping(action) bootstrap node: %v", peerInfo)
			} else {
				logger.Sugar.Errorf("Failed to ping(action) bootstrap node: %v, err: %v", peerInfo, err)
			}*/
		}()
	}

	// 初始化dht，5分钟刷新路由表
	logger.Sugar.Infof("Bootstrapping the DHT")
	if err := global.Global.PeerEndpointDHT.Bootstrap(global.Global.Context); err != nil {
		return err
	}

	return nil
}
