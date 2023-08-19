package notifiee

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/libp2p/go-libp2p/core/network"
	ma "github.com/multiformats/go-multiaddr"
)

type NetNotifiee struct {
}

var netNotifiee = NetNotifiee{}

func (noti *NetNotifiee) Listen(net network.Network, addr ma.Multiaddr) {
	logger.Sugar.Infof("new net Listen addr:%v", addr)
}

func (noti *NetNotifiee) ListenClose(net network.Network, addr ma.Multiaddr) {
	logger.Sugar.Infof("close net ListenClose addr:%v", addr)
}

func (noti *NetNotifiee) Connected(net network.Network, conn network.Conn) {
	logger.Sugar.Infof("new conn id:%v, addr:%v, peer:%v", conn.ID(), conn.RemoteMultiaddr(), conn.RemotePeer())
}

func (noti *NetNotifiee) Disconnected(net network.Network, conn network.Conn) {
	logger.Sugar.Infof("dis conn id:%v, addr:%v, peer:%v", conn.ID(), conn.RemoteMultiaddr(), conn.RemotePeer())
}

func (noti *NetNotifiee) OpenedStream(net network.Network, stream network.Stream) {
	logger.Sugar.Infof("new stream id:%v, protocol:%v", stream.ID(), stream.Protocol())
}

func (noti *NetNotifiee) ClosedStream(net network.Network, stream network.Stream) {
	logger.Sugar.Infof("close stream id:%v, protocol:%v", stream.ID(), stream.Protocol())
}

func SetNetNotifiee() {
	net := global.Global.Host.Network()
	net.Notify(&netNotifiee)
}
