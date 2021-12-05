package global

import (
	"context"
	"gitee.com/cristiane/go-push-sdk/push"
	openpgp "github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/libp2p/go-libp2p-core/connmgr"
	libp2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
	"strings"
)

type global struct {
	Context           context.Context
	Host              host.Host
	PeerId            peer.ID
	PeerPrivateKey    libp2pcrypto.PrivKey
	PeerPublicKey     libp2pcrypto.PubKey
	PrivateKey        *openpgp.Key
	PublicKey         *openpgp.Key
	Multiaddrs        []multiaddr.Multiaddr
	Rendezvous        string
	ConnectionManager connmgr.ConnManager
	PeerEndpointDHT   *dht.IpfsDHT
	ChainProtocolID   protocol.ID
	MyselfPeer        *entity.MyselfPeer

	WebrtcstarHost host.Host

	PushRegisterClient *push.RegisterClient
	HuaweiAccessToken string
}

var Global = global{}

const GeneralP2pAddrFormat = "%v/p2p/%v"
const DefaultP2pAddrFormat = "/dns4/%v/tcp/%v/p2p/%v" // 未来可能切换到通用地址"%v/p2p/%v", fullAddr, peerId
const DefaultAddrFormat = "/ip4/%v/tcp/%v"
const DefaultDnsAddrFormat = "/dns4/%v/tcp/%v"
const DefaultWsAddrFormat = "/ip4/%v/tcp/%v/ws"
const DefaultWssAddrFormat = "/ip4/%v/tcp/%v/wss"
const DefaultDnsWsAddrFormat = "/dns4/%v/tcp/%v/ws"
const DefaultDnsWssAddrFormat = "/dns4/%v/tcp/%v/wss"
const DefaultWebrtcstarAddrFormat = "/ip4/%v/tcp/%v/ws/p2p-webrtc-star"
const DefaultAddr = "0.0.0.0"
const DefaultPort = "3719"
const DefaultWsPort = "4719"
const DefaultWssPort = "5719"
const DefaultExternalPort = "3720"
const DefaultExternalWsPort = "4720"
const DefaultExternalWssPort = "5720"

func Print() {
	Global.Multiaddrs = Global.Host.Addrs()
	logger.Sugar.Infof("p2p local peer address:%v", Global.Multiaddrs)
	logger.Sugar.Infof("p2p local peer peerId:%v", string(Global.PeerId))
	logger.Sugar.Infof("p2p local peer rendezvous:%v", Global.Rendezvous)
	logger.Sugar.Infof("protocolID are:%v", Global.ChainProtocolID)
	logger.Sugar.Infof("successfully start p2p server, enjoy it!")
}

func IsMyself(peerId string) bool {
	id := string(Global.PeerId)
	if strings.Contains(peerId, id) {
		return true
	}
	return false
}
