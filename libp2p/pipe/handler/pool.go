package handler

import (
	"context"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/libp2p/pipe"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"strings"
	"sync"
	"time"
)

type PipePool struct {
	connectionPool map[string]network.Conn
	requestPool    map[string]*pipe.Pipe
	responsePool   map[string]*pipe.Pipe
	lock           sync.Mutex
}

var pipePool = &PipePool{connectionPool: make(map[string]network.Conn), requestPool: make(map[string]*pipe.Pipe), responsePool: make(map[string]*pipe.Pipe)}

func GetPipePool() *PipePool {
	return pipePool
}

func (this *PipePool) GetResponsePipe(peerId string, connectSessionId string) *pipe.Pipe {
	this.lock.Lock()
	defer this.lock.Unlock()
	key := peerId + ":" + connectSessionId
	logger.Sugar.Debugf("GetResponsePipe-key: %v", key)
	p, ok := this.responsePool[key]
	if ok {
		return p
	} else {
		conn, ok := this.connectionPool[key]
		if ok {
			stream, err := conn.NewStream(context.Background())
			if err != nil {
				logger.Sugar.Errorf(err.Error())
				return nil
			}
			stream.SetProtocol(global.Global.ChainProtocolID)
			p, err := pipe.CreatePipe(stream, HandleRaw, msgtype.MsgDirect_Request)
			if err != nil {
				logger.Sugar.Errorf(err.Error())
				return nil
			}
			if p != nil {
				this.responsePool[key] = p
				return p
			}
		}
	}
	return nil
}

/**
主动发送消息获取管道，如果流不存在，创建一个
*/
func (this *PipePool) GetRequestPipe(peerId string, protocolId string) *pipe.Pipe {
	/*this.lock.Lock()
	defer this.lock.Unlock()
	reqKey := GetPeerId(peerId) + ":" + protocolId
	logger.Sugar.Infof("GetRequestPipe-reqKey: %v", reqKey)
	p, ok := this.requestPool[reqKey]
	if ok {
		return p
	} else {*/
	var id peer.ID
	if strings.HasPrefix(peerId, "/") {
		addr, err := ma.NewMultiaddr(peerId)
		if err != nil {
			logger.Sugar.Errorf(err.Error())
			return nil
		}
		// Extract the peer ID from the multiaddr.
		info, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			logger.Sugar.Errorf(err.Error())
			return nil
		}
		// Add the destination's peer multiaddress in the peerstore.
		// This will be used during connection and stream creation by libp2p.
		//global.Global.Host.Connect(global.Global.Context, *info)
		global.Global.Host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
		id = info.ID
	} else {
		p, err := peer.Decode(peerId)
		if err != nil {
			logger.Sugar.Errorf(err.Error())
			return nil
		} else {
			id = p
		}
	}
	// 主动创建流和管道与其他peer沟通，发消息，handler用于最终发送前消息的预先处理，或者接收消息后的处理
	stream, err := global.Global.Host.NewStream(global.Global.Context, id, protocol.ID(protocolId))
	if err != nil {
		logger.Sugar.Errorf("NewStream failed:%v", err)
		return nil
	} else {
		// 设置通用的收到消息流的处理器，被动接收其他peer发送过来的消息，无论哪种协议类型，都放在HandleRaw中分发
		p, err := pipe.CreatePipe(stream, HandleRaw, msgtype.MsgDirect_Request)
		if err != nil {
			logger.Sugar.Errorf(err.Error())
			return nil
		}
		if p != nil {
			conn := p.GetStream().Conn()
			if conn != nil {
				peerId := conn.RemotePeer().Pretty()
				logger.Sugar.Infof("GetRequestPipe-remote peer: %v %v, steamId: %v", peerId, conn.ID(), stream.ID())
				/*key := peerId + ":" + conn.ID()
				logger.Sugar.Infof("GetRequestPipe-key: %v", key)
				oldConn, ok := this.connectionPool[key]
				if ok {
					if conn != oldConn {
						logger.Sugar.Infof("----------GetRequestPipe-resetConn: v%", key)
						oldConn.Close()
						this.connectionPool[key] = conn
					}
				} else {
					logger.Sugar.Infof("----------GetRequestPipe-newConn: %v", key)
					this.connectionPool[key] = conn
				}*/
			}
			/*this.requestPool[reqKey] = p*/
			return p
		}
	}
	/*}*/
	return nil
}

func (this *PipePool) CreatePipe(stream network.Stream, direct string) *pipe.Pipe {
	this.lock.Lock()
	defer this.lock.Unlock()
	p, err := pipe.CreatePipe(stream, HandleRaw, direct)
	if err != nil {
		logger.Sugar.Errorf(err.Error())
		return nil
	}
	if p != nil {
		conn := p.GetStream().Conn()
		if conn != nil {
			peerId := conn.RemotePeer().Pretty()
			logger.Sugar.Debugf("CreatePipe-remote peer: %v %v, steamId: %v", peerId, conn.ID(), stream.ID())
			key := peerId + ":" + conn.ID()
			logger.Sugar.Debugf("CreatePipe-key: %v", key)
			oldConn, ok := this.connectionPool[key]
			if ok {
				if conn != oldConn {
					logger.Sugar.Debugf("----------CreatePipe-resetConn: v%", key)
					oldConn.Close()
					this.connectionPool[key] = conn
				}
			} else {
				logger.Sugar.Debugf("----------CreatePipe-newConn: %v", key)
				this.connectionPool[key] = conn
			}
			_, ok = this.responsePool[key]
			if !ok {
				this.responsePool[key] = p
			}

			/*reqKey := peerId + ":" + string(p.GetStream().Protocol())
			logger.Sugar.Infof("CreatePipe-reqKey: %v", reqKey)
			_, ok = this.requestPool[reqKey]
			if !ok {
				this.requestPool[reqKey] = p
			}*/
		}
		return p
	}
	return nil
}

func (this *PipePool) Close(peerId string, protocolId string, connectSessionId string, streamId string) {
	this.lock.Lock()
	defer this.lock.Unlock()
	key := peerId + ":" + connectSessionId
	logger.Sugar.Debugf("Close-key: %v", key)
	p, ok := this.responsePool[key]
	if ok {
		if p.GetStream().ID() == streamId {
			delete(this.responsePool, key)
		}
	}

	/*reqKey := peerId + ":" + protocolId
	logger.Sugar.Infof("Close-reqKey: %v", reqKey)
	p, ok = this.requestPool[reqKey]
	if ok {
		if p.GetStream().ID() == streamId {
			delete(this.requestPool, reqKey)
		}
	}*/
}

func (this *PipePool) Disconnect(peerId string, connectSessionId string) {
	this.lock.Lock()
	defer this.lock.Unlock()
	key := peerId + ":" + connectSessionId
	logger.Sugar.Debugf("Disconnect-key: %v", key)
	_, ok := this.connectionPool[key]
	if ok {
		logger.Sugar.Debugf("----------deleteConn: %v", key)
		delete(this.connectionPool, key)
		// 更新信息
		peerClients, err := service.GetPeerClientService().GetLocals(ns.PeerClient_KeyKind, peerId, "", "")
		if err != nil {
			logger.Sugar.Errorf("failed to GetLocalPCs by peerId: %v, err: %v", peerId, err)
		} else {
			if len(peerClients) > 0 {
				for _, peerClient := range peerClients {
					if peerClient.ConnectSessionId == connectSessionId {
						currentTime := time.Now()
						peerClient.LastAccessTime = &currentTime
						peerClient.ActiveStatus = entity.ActiveStatus_Down
						err = service.GetPeerClientService().PutValues(peerClient)
						if err != nil {
							logger.Sugar.Errorf("failed to PutPCs, peerId: %v, err: %v", peerId, err)
						}
						break
					}
				}
			}
		}
	}
}
