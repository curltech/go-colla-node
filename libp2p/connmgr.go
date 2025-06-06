package libp2p

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/pipe/handler"
	"github.com/libp2p/go-libp2p/core/network"
	connmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
	ma "github.com/multiformats/go-multiaddr"
)

type MonitorConnMgr struct {
	*connmgr.BasicConnMgr
}

func NewConnManager(basicConnMgr *connmgr.BasicConnMgr) *MonitorConnMgr {
	monitorConnMgr := MonitorConnMgr{}
	monitorConnMgr.BasicConnMgr = basicConnMgr

	return &monitorConnMgr
}

// Notifee returns a sink through which Notifiers can inform the BasicConnMgr when
// events occur. Currently, the notifee only reacts upon connection events
// {Connected, Disconnected}.
func (cm *MonitorConnMgr) Notifee() network.Notifiee {
	return (*cmNotifee)(cm)
}

type cmNotifee MonitorConnMgr

// Connected is called by notifiers to inform that a new connection has been established.
// The notifee updates the BasicConnMgr to start tracking the connection. If the new connection
// count exceeds the high watermark, a trim may be triggered.
func (nn *cmNotifee) Connected(n network.Network, c network.Conn) {
	nn.BasicConnMgr.Notifee().Connected(n, c)
	peerId := c.RemotePeer().String()
	addr := c.RemoteMultiaddr().String()
	logger.Sugar.Debugf("New Connected! %v %v, addr:%v", peerId, c.ID(), addr)
}

// Disconnected is called by notifiers to inform that an existing connection has been closed or terminated.
// The notifee updates the BasicConnMgr accordingly to stop tracking the connection, and performs housekeeping.
func (nn *cmNotifee) Disconnected(n network.Network, c network.Conn) {
	nn.BasicConnMgr.Notifee().Disconnected(n, c)
	peerId := c.RemotePeer().String()
	addr := c.RemoteMultiaddr().String()
	logger.Sugar.Debugf("New Disconnected! %v %v, addr:%v", peerId, c.ID(), addr)
	handler.Disconnect(peerId, "", c.ID())
}

// Listen is no-op in this implementation.
func (nn *cmNotifee) Listen(n network.Network, addr ma.Multiaddr) {
	nn.BasicConnMgr.Notifee().Listen(n, addr)
	logger.Sugar.Debugf("New Listen! %v", addr.String())
}

// ListenClose is no-op in this implementation.
func (nn *cmNotifee) ListenClose(n network.Network, addr ma.Multiaddr) {
	nn.BasicConnMgr.Notifee().ListenClose(n, addr)
	logger.Sugar.Debugf("New ListenClose! %v", addr.String())
}

// OpenedStream is no-op in this implementation.
func (nn *cmNotifee) OpenedStream(n network.Network, s network.Stream) {
	//nn.BasicConnMgr.Notifee().OpenedStream(n, s)
	logger.Sugar.Debugf("New OpenedStream! %v %v", s.ID(), s.Protocol())
}

// ClosedStream is no-op in this implementation.
func (nn *cmNotifee) ClosedStream(n network.Network, s network.Stream) {
	peerId := s.Conn().RemotePeer().String()
	logger.Sugar.Debugf("New ClosedStream! %v %v %v", s.ID(), s.Protocol(), peerId)
	//nn.BasicConnMgr.Notifee().ClosedStream(n, s)
	handler.Close(peerId, string(s.Protocol()), s.Conn().ID(), s.ID())
}
