package libp2p

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/pipe/handler"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/network"
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
	peerId := c.RemotePeer().Pretty()
	logger.Infof("New Connected! %v", peerId)
}

// Disconnected is called by notifiers to inform that an existing connection has been closed or terminated.
// The notifee updates the BasicConnMgr accordingly to stop tracking the connection, and performs housekeeping.
func (nn *cmNotifee) Disconnected(n network.Network, c network.Conn) {
	nn.BasicConnMgr.Notifee().Disconnected(n, c)
	peerId := c.RemotePeer().Pretty()
	addr := c.RemoteMultiaddr().String()
	logger.Infof("New Disconnected! %v, addr:%v", peerId, addr)
	handler.GetPipePool().Disconnect(peerId, c.ID())
}

// Listen is no-op in this implementation.
func (nn *cmNotifee) Listen(n network.Network, addr ma.Multiaddr) {
	nn.BasicConnMgr.Notifee().Listen(n, addr)
	logger.Infof("New Listen! %v", addr.String())
}

// ListenClose is no-op in this implementation.
func (nn *cmNotifee) ListenClose(n network.Network, addr ma.Multiaddr) {
	nn.BasicConnMgr.Notifee().ListenClose(n, addr)
	logger.Infof("New ListenClose! %v", addr.String())
}

// OpenedStream is no-op in this implementation.
func (nn *cmNotifee) OpenedStream(n network.Network, s network.Stream) {
	nn.BasicConnMgr.Notifee().OpenedStream(n, s)
	logger.Infof("New OpenedStream! %v %v", s.ID(), s.Protocol())
}

// ClosedStream is no-op in this implementation.
func (nn *cmNotifee) ClosedStream(n network.Network, s network.Stream) {
	nn.BasicConnMgr.Notifee().ClosedStream(n, s)
	logger.Infof("New ClosedStream! %v %v", s.ID(), s.Protocol())
	handler.GetPipePool().Close(s.Conn().RemotePeer().Pretty(), string(s.Protocol()), s.Conn().ID(), s.ID())
}
