package controller

import (
	"github.com/curltech/go-colla-core/container"
	"github.com/curltech/go-colla-core/crypto/std"
	"github.com/curltech/go-colla-node/libp2p/util"

	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/kataras/iris/v12"
	"github.com/libp2p/go-libp2p-core/peer"
	kb "github.com/libp2p/go-libp2p-kbucket"
	"time"
)

type DhtController struct {
}

var dhtController = &DhtController{}

func (this *DhtController) PeerID(ctx iris.Context) {
	Id := dht.PeerEndpointDHT.PeerID()
	ctx.JSON(Id)
}

func (this *DhtController) Host(ctx iris.Context) {
	host := dht.PeerEndpointDHT.Host()
	hostinfo := this.getAddrInfo(host.ID())
	ids := host.Peerstore().Peers()
	hostinfo["peers"] = this.getAddrInfos(ids)
	ctx.JSON(&hostinfo)
}

func (this *DhtController) getAddrInfos(ids []peer.ID) []*map[string]interface{} {
	host := dht.PeerEndpointDHT.Host()
	if ids != nil && len(ids) > 0 {
		addrInfos := make([]*map[string]interface{}, 0)
		for _, id := range ids {
			if id == host.ID() {
				continue
			}
			addrInfo := this.getAddrInfo(id)
			addrInfos = append(addrInfos, &addrInfo)
		}
		return addrInfos
	}

	return nil
}

func (this *DhtController) getAddrInfo(id peer.ID) map[string]interface{} {
	addrInfo := make(map[string]interface{}, 0)
	host := dht.PeerEndpointDHT.Host()
	ai := host.Peerstore().PeerInfo(id)
	addrInfo["peerId"] = ai.ID
	addrInfo["addrs"] = ai.Addrs
	buf, _ := host.Peerstore().PubKey(ai.ID).Bytes()
	addrInfo["pubKey"] = std.EncodeBase64(buf)
	addrInfo["protocols"], _ = host.Peerstore().GetProtocols(ai.ID)

	return addrInfo
}

func (this *DhtController) GetPublicKey(ctx iris.Context) {
	param := P2pParam{}
	ctx.ReadJSON(&param)
	if param.PeerId == "" {
		ctx.StopWithJSON(iris.StatusInternalServerError, "NoPeerId")

		return
	}
	id, err := peer.Decode(param.PeerId)
	if err != nil {
		ctx.StopWithJSON(iris.StatusInternalServerError, err.Error())

		return
	}
	pk, err := dht.PeerEndpointDHT.GetPublicKey(id)
	if err != nil {
		ctx.StopWithJSON(iris.StatusInternalServerError, err.Error())

		return
	}
	ctx.JSON(pk)
}

func (this *DhtController) Connect(ctx iris.Context) {
	param := P2pParam{}
	ctx.ReadJSON(&param)
	if param.PeerId == "" {
		ctx.StopWithJSON(iris.StatusInternalServerError, "NoPeerId")

		return
	}
	if param.Addr == "" {
		ctx.StopWithJSON(iris.StatusInternalServerError, "NoAddr")

		return
	}
	addrInfo, err := util.ToAddInfo(param.PeerId, param.Addr)
	if err != nil {
		ctx.StopWithJSON(iris.StatusInternalServerError, err.Error())

		return
	}
	if addrInfo == nil {
		ctx.StopWithJSON(iris.StatusInternalServerError, "NoAddrInfo")

		return
	}
	start := time.Now()
	err = global.Global.Host.Connect(global.Global.Context, *addrInfo)
	if err != nil {
		ctx.StopWithJSON(iris.StatusInternalServerError, err.Error())

		return
	}
	interval := time.Since(start)
	if err != nil {
		ctx.StopWithJSON(iris.StatusInternalServerError, err.Error())

		return
	} else {
		result := make(map[string]interface{})
		result["start"] = start
		result["interval"] = interval
		ctx.JSON(&result)
	}
}

func (this *DhtController) Ping(ctx iris.Context) {
	param := P2pParam{}
	ctx.ReadJSON(&param)
	if param.PeerId == "" {
		ctx.StopWithJSON(iris.StatusInternalServerError, "NoPeerId")

		return
	}
	id, err := peer.Decode(param.PeerId)
	if err != nil {
		ctx.StopWithJSON(iris.StatusInternalServerError, err.Error())

		return
	}
	start := time.Now()
	err = dht.PeerEndpointDHT.Ping(id)
	interval := time.Since(start)
	if err != nil {
		ctx.StopWithJSON(iris.StatusInternalServerError, err.Error())

		return
	}
	result := make(map[string]interface{})
	result["start"] = start
	result["interval"] = interval
	ctx.JSON(&result)
}

func (this *DhtController) FindLocal(ctx iris.Context) {
	param := P2pParam{}
	ctx.ReadJSON(&param)
	if param.PeerId == "" {
		ctx.StopWithJSON(iris.StatusInternalServerError, "NoPeerId")

		return
	}
	id, err := peer.Decode(param.PeerId)
	if err != nil {
		ctx.StopWithJSON(iris.StatusInternalServerError, err.Error())

		return
	}
	ctx.JSON(dht.PeerEndpointDHT.FindLocal(id))
}

func (this *DhtController) FindPeer(ctx iris.Context) {
	param := P2pParam{}
	ctx.ReadJSON(&param)
	if param.PeerId == "" {
		ctx.StopWithJSON(iris.StatusInternalServerError, "NoPeerId")

		return
	}
	id, err := peer.Decode(param.PeerId)
	if err != nil {
		ctx.StopWithJSON(iris.StatusInternalServerError, err.Error())

		return
	}
	addrInfo, err := dht.PeerEndpointDHT.FindPeer(id)
	if err != nil {
		ctx.StopWithJSON(iris.StatusInternalServerError, err.Error())

		return
	}
	ctx.JSON(addrInfo)
}

func (this *DhtController) GetValue(ctx iris.Context) {
	param := P2pParam{}
	ctx.ReadJSON(&param)
	key := param.Key
	if param.Key == "" {
		ctx.StopWithJSON(iris.StatusInternalServerError, "NoKey")

		return
	}
	buf, err := dht.PeerEndpointDHT.GetValue(key)
	if err != nil {
		ctx.StopWithJSON(iris.StatusInternalServerError, err.Error())

		return
	}
	ctx.JSON(string(buf))
}

func (this *DhtController) PutValue(ctx iris.Context) {
	param := P2pParam{}
	ctx.ReadJSON(&param)
	if param.Key == "" {
		ctx.StopWithJSON(iris.StatusInternalServerError, "NoKey")

		return
	}
	key := param.Key
	value := param.Value

	err := dht.PeerEndpointDHT.PutValue(key, []byte(value))
	if err != nil {
		ctx.StopWithJSON(iris.StatusInternalServerError, err.Error())

		return
	}
	ctx.JSON("ok")
}

func (this *DhtController) PutMyself(ctx iris.Context) {
	err := dht.PeerEndpointDHT.PutMyself()
	if err != nil {
		ctx.StopWithJSON(iris.StatusInternalServerError, err.Error())

		return
	}
	ctx.JSON("Ok")
}

type RoutingTableController struct {
}

var routingTableController = &RoutingTableController{}

func (this *RoutingTableController) ListPeers(ctx iris.Context) {
	ids := dht.PeerEndpointDHT.RoutingTable.ListPeers()
	peers := make([]*map[string]interface{}, 0)
	if ids != nil && len(ids) > 0 {
		peers = dhtController.getAddrInfos(ids)
	}
	ctx.JSON(peers)
}

func (this *RoutingTableController) GetPeerInfos(ctx iris.Context) {
	peerInfos := dht.PeerEndpointDHT.RoutingTable.GetPeerInfos()
	ctx.JSON(peerInfos)
}

func (this *RoutingTableController) Find(ctx iris.Context) {
	param := P2pParam{}
	ctx.ReadJSON(&param)
	if param.PeerId == "" {
		ctx.StopWithJSON(iris.StatusInternalServerError, "NoPeerId")

		return
	}
	id, err := peer.Decode(param.PeerId)
	if err != nil {
		ctx.StopWithJSON(iris.StatusInternalServerError, err.Error())

		return
	}
	ctx.JSON(dht.PeerEndpointDHT.RoutingTable.Find(id))
}

func (this *RoutingTableController) NearestPeer(ctx iris.Context) {
	param := P2pParam{}
	ctx.ReadJSON(&param)
	if param.PeerId == "" {
		ctx.StopWithJSON(iris.StatusInternalServerError, "NoPeerId")

		return
	}
	_, err := peer.Decode(param.PeerId)
	if err != nil {
		ctx.StopWithJSON(iris.StatusInternalServerError, err.Error())

		return
	}
	id := kb.ConvertKey(param.PeerId)
	ctx.JSON(dht.PeerEndpointDHT.RoutingTable.NearestPeer(id))
}

func (this *RoutingTableController) NearestPeers(ctx iris.Context) {
	param := P2pParam{}
	ctx.ReadJSON(&param)
	if param.PeerId == "" {
		ctx.StopWithJSON(iris.StatusInternalServerError, "NoPeerId")

		return
	}
	id := kb.ConvertKey(param.PeerId)
	ids := dht.PeerEndpointDHT.RoutingTable.NearestPeers(id, 10)
	if ids != nil && len(ids) > 0 {
		peers := dhtController.getAddrInfos(ids)
		ctx.JSON(peers)
	}
}

func init() {
	container.RegistController("dht", dhtController)
	container.RegistController("routingTable", routingTableController)
}
