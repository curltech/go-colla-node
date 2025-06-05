package receiver

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	svc "github.com/curltech/go-colla-node/p2p/dht/service"
	"sync"
	"time"
)

// PeeClientId sessionId与PeeClientd的对应关系,删除连接时有用
type PeeClientId struct {
	PeerId   string
	ClientId string
}

// PeerClientConnectionPool connectSessionId与PeeClientId的映射
var peerClientConnectionPool sync.Map //make(map[string]*PeeClientId)

func UpdatePeerClient(peerClient *entity.PeerClient) {
	if peerClient.PeerId == "" {
		logger.Sugar.Errorf("remotePeerId is blank")
		return
	}
	peerClientConnectionPool.Store(peerClient.ConnectSessionId, &PeeClientId{PeerId: peerClient.PeerId, ClientId: peerClient.ClientId})
	k := ns.GetPeerClientKey(peerClient.PeerId)
	peerClients, err := svc.GetPeerClientService().GetLocals(k, peerClient.ClientId)
	if err != nil {
		logger.Sugar.Errorf("failed to GetLocalPCs by peerId: %v, err: %v", peerClient.PeerId, err)
	}
	if len(peerClients) > 0 {
		for _, pc := range peerClients {
			var activeStatus = pc.ActiveStatus
			if activeStatus != entity.ActiveStatus_Up {
				currentTime := time.Now()
				pc.LastAccessTime = &currentTime
				pc.ActiveStatus = entity.ActiveStatus_Up
				pc.ConnectSessionId = peerClient.ConnectSessionId
				pc.ConnectPeerId = peerClient.ConnectPeerId
				pc.ClientId = peerClient.ClientId
				err = svc.GetPeerClientService().PutValues(pc)
				if err != nil {
					logger.Sugar.Errorf("failed to PutPCs, peerId: %v, err: %v", pc.PeerId, err)
				}
				break
			}
		}
	}
}

func HandleDisconnected(connectSessionId string) {
	v, ok := peerClientConnectionPool.Load(connectSessionId)
	if ok {
		var peerClientId *PeeClientId = v.(*PeeClientId)
		peerClientConnectionPool.Delete(connectSessionId)
		// 更新信息
		k := ns.GetPeerClientKey(peerClientId.PeerId)
		peerClients, err := svc.GetPeerClientService().GetLocals(k, peerClientId.ClientId)
		if err != nil {
			logger.Sugar.Errorf("failed to GetLocalPCs by peerId: %v, err: %v", peerClientId.PeerId, err)
		} else {
			if len(peerClients) > 0 {
				for _, peerClient := range peerClients {
					if peerClient.ConnectSessionId == connectSessionId {
						currentTime := time.Now()
						peerClient.LastAccessTime = &currentTime
						if peerClient.ActiveStatus != entity.ActiveStatus_Down {
							peerClient.ActiveStatus = entity.ActiveStatus_Down
							err = svc.GetPeerClientService().PutValues(peerClient)
							if err != nil {
								logger.Sugar.Errorf("failed to PutPCs, peerId: %v, err: %v", peerClientId.PeerId, err)
							}
							break
						}
					}
				}
			}
		}
	}
}
