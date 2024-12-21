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
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"strings"
	"time"
)

// PipePool libp2p的连接池
var NetworkConnectionPool = make(map[string]network.Conn)

var RequestPipePool = make(map[string]*pipe.Pipe)

var ResponsePipePool = make(map[string]*pipe.Pipe)

func GetResponsePipe(connectSessionId string) *pipe.Pipe {
	p, ok := ResponsePipePool[connectSessionId]
	if ok {
		return p
	} else {
		conn, ok := NetworkConnectionPool[connectSessionId]
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
				ResponsePipePool[connectSessionId] = p
				return p
			}
		}
	}
	return nil
}

// GetRequestPipe 主动发送消息获取管道，如果流不存在，创建一个
func GetRequestPipe(peerId string, protocolId string) *pipe.Pipe {
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
				peerId := conn.RemotePeer().String()
				logger.Sugar.Debugf("GetRequestPipe-remote peer: %v %v, steamId: %v", peerId, conn.ID(), stream.ID())
			}
			return p
		}
	}
	return nil
}

func CreatePipe(stream network.Stream, direct string) *pipe.Pipe {
	p, err := pipe.CreatePipe(stream, HandleRaw, direct)
	if err != nil {
		logger.Sugar.Errorf(err.Error())
		return nil
	}
	if p != nil {
		conn := p.GetStream().Conn()
		if conn != nil {
			peerId := conn.RemotePeer().String()
			logger.Sugar.Debugf("CreatePipe-remote peer: %v %v, steamId: %v", peerId, conn.ID(), stream.ID())
			connectSessionId := conn.ID()
			logger.Sugar.Debugf("CreatePipe-connectSessionId: %v", connectSessionId)
			oldConn, ok := NetworkConnectionPool[connectSessionId]
			if ok {
				if conn != oldConn {
					logger.Sugar.Debugf("----------CreatePipe-resetConn: v%", connectSessionId)
					oldConn.Close()
					NetworkConnectionPool[connectSessionId] = conn
				}
			} else {
				logger.Sugar.Debugf("----------CreatePipe-newConn: %v", connectSessionId)
				NetworkConnectionPool[connectSessionId] = conn
			}
			_, ok = ResponsePipePool[connectSessionId]
			if !ok {
				ResponsePipePool[connectSessionId] = p
			}
		}
		return p
	}
	return nil
}

func Close(peerId string, protocolId string, connectSessionId string, streamId string) {
	logger.Sugar.Debugf("Close-connectSessionId: %v", connectSessionId)
	p, ok := ResponsePipePool[connectSessionId]
	if ok {
		if p.GetStream().ID() == streamId {
			delete(ResponsePipePool, connectSessionId)
		}
	}
}

func Disconnect(peerId string, clientId string, connectSessionId string) {
	logger.Sugar.Debugf("Disconnect-connectSessionId: %v", connectSessionId)
	_, ok := NetworkConnectionPool[connectSessionId]
	if ok {
		logger.Sugar.Debugf("----------deleteConn: %v", connectSessionId)
		delete(NetworkConnectionPool, connectSessionId)
		// 更新信息
		k := ns.GetPeerClientKey(peerId)
		peerClients, err := service.GetPeerClientService().GetLocals(k, clientId)
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
