package handler

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/libp2p/pipe"
	"github.com/curltech/go-colla-node/p2p/chain/service"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"sync"
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

func (this *PipePool) Connect(p *pipe.Pipe) {
	this.lock.Lock()
	defer this.lock.Unlock()
	conn := p.GetStream().Conn()
	if conn != nil {
		peerId := conn.RemotePeer().Pretty()
		logger.Infof("remote peer:%v", peerId)
		key := peerId + ":" + conn.ID()
		oldConn, ok := this.connectionPool[key]
		if ok {
			if conn != oldConn {
				oldConn.Close()
				this.connectionPool[key] = conn
			}
		} else {
			this.connectionPool[key] = conn
		}

		_, ok = this.responsePool[key]
		if !ok {
			this.responsePool[key] = p
		}

		key = peerId + ":" + string(p.GetStream().Protocol())
		_, ok = this.requestPool[key]
		if !ok {
			this.requestPool[key] = p
		}
	}
}

func (this *PipePool) GetResponsePipe(peerId string, connectSessionId string) *pipe.Pipe {
	this.lock.Lock()
	defer this.lock.Unlock()
	peerId = GetAddrInfo(peerId)
	key := peerId + ":" + connectSessionId
	p, ok := this.responsePool[key]
	if ok {
		return p
	} else {
		conn, ok := this.connectionPool[key]
		if ok {
			stream, err := conn.NewStream()
			if err == nil {
				stream.SetProtocol(global.Global.ChainProtocolID)
				p, err := CreatePipe(stream, msgtype.MsgDirect_Request)
				if err == nil {
					this.responsePool[key] = p

					return p
				}
			}
		}
	}

	return nil
}

/**
主动发送消息获取管道，如果流不存在，创建一个
*/
func (this *PipePool) GetRequestPipe(peerId string, protocolId string) (*pipe.Pipe, error) {
	this.lock.Lock()
	defer this.lock.Unlock()
	var err error
	var key = peerId + ":" + protocolId
	p, ok := this.requestPool[key]
	if ok {
		return p, nil
	} else {
		p, err := createPipe(peerId, protocolId)
		if err != nil {
			logger.Errorf("createPipe failure")
		}

		return p, err
	}

	return nil, err
}

/**
主动创建流和管道与其他peer沟通，发消息，handler用于最终发送前消息的预先处理，或者接收消息后的处理
peerId是带地址信息的/ip4/192.168.0.104/tcp/3721/p2p/12D3KooWPpZrX5bNEpJcHYFACTKkmMMxF39oU6Rm2WeK4rr8mRVp
*/
func createPipe(peerId string, protocolId string) (*pipe.Pipe, error) {
	addr, err := ma.NewMultiaddr(peerId)
	if err != nil {
		logger.Errorf(err.Error())
		return nil, err
	}
	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		logger.Errorf(err.Error())
		return nil, err
	}
	// Add the destination's peer multiaddress in the peerstore.
	// This will be used during connection and stream creation by libp2p.
	//global.Global.Host.Connect(global.Global.Context, *info)
	global.Global.Host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	stream, err := global.Global.Host.NewStream(global.Global.Context, info.ID, protocol.ID(protocolId))
	if err != nil {
		logger.Errorf("CreatePipe failed:%v", err)
		return nil, err
	} else {
		/**
		设置通用的收到消息流的处理器，被动接收其他peer发送过来的消息，无论哪种协议类型，都放在HandleRaw中分发
		*/
		return CreatePipe(stream, msgtype.MsgDirect_Request)
	}

	return nil, err
}

func CreatePipe(stream network.Stream, msgtype string) (*pipe.Pipe, error) {
	pipe, err := pipe.CreatePipe(stream, HandleRaw, msgtype)
	if err != nil {
		logger.Errorf(err.Error())
		return nil, err
	}
	if pipe != nil {
		GetPipePool().Connect(pipe)

		return pipe, nil
	}
	return nil, nil
}

func (this *PipePool) Close(peerId string, protocolId string, connectSessionId string, streamId string) {
	this.lock.Lock()
	defer this.lock.Unlock()
	key := peerId + ":" + connectSessionId
	p, ok := this.responsePool[key]
	if ok {
		if p.GetStream().ID() == streamId {
			delete(this.responsePool, key)
			peerClients, err := service.GetLocalPCs(ns.PeerClient_KeyKind, peerId, "", "")
			if err != nil {
				logger.Errorf("failed to GetLocalPCs by peerId: %v", peerId)
				return
			}
			if len(peerClients) > 0 {
				for _, peerClient := range peerClients {
					if peerClient.ConnectSessionId == connectSessionId {
						peerClient.ActiveStatus = entity.ActiveStatus_Down
						service.PutPCs(peerClient)
						break
					}
				}
			}
		}
	}
	key = peerId + ":" + protocolId
	p, ok = this.requestPool[key]
	if ok {
		if p.GetStream().ID() == streamId {
			delete(this.requestPool, key)
		}
	}
}

func (this *PipePool) Disconnect(peerId string, connectSessionId string) {
	this.lock.Lock()
	defer this.lock.Unlock()
	key := peerId + ":" + connectSessionId
	_, ok := this.connectionPool[key]
	if ok {
		delete(this.connectionPool, key)
	}
}
