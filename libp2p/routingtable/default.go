package routingtable

import (
	kbucket "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p-kbucket/peerdiversity"
	"github.com/libp2p/go-libp2p/core/peer"
	"time"
)

/*
*
提供了routingTable的服务的封装供外部调用
*/
type PeerEntityRoutingTable struct {
	RoutingTable *kbucket.RoutingTable
}

func (this *PeerEntityRoutingTable) PeerAdded(peerAdded func(id peer.ID)) {
	this.RoutingTable.PeerAdded = peerAdded
}

func (this *PeerEntityRoutingTable) PeerRemoved(peerRemoved func(id peer.ID)) {
	this.RoutingTable.PeerRemoved = peerRemoved
}

func (this *PeerEntityRoutingTable) TryAddPeer(p peer.ID, queryPeer bool, isReplaceable bool) (bool, error) {
	return this.RoutingTable.TryAddPeer(p, queryPeer, isReplaceable)
}

func (this *PeerEntityRoutingTable) GetPeerInfos() []kbucket.PeerInfo {
	return this.RoutingTable.GetPeerInfos()
}

func (this *PeerEntityRoutingTable) RemovePeer(p peer.ID) {
	this.RoutingTable.RemovePeer(p)
}

func (this *PeerEntityRoutingTable) Find(id peer.ID) peer.ID {
	return this.RoutingTable.Find(id)
}

func (this *PeerEntityRoutingTable) NearestPeer(id kbucket.ID) peer.ID {
	return this.RoutingTable.NearestPeer(id)
}

func (this *PeerEntityRoutingTable) NearestPeers(id kbucket.ID, count int) []peer.ID {
	return this.RoutingTable.NearestPeers(id, count)
}

func (this *PeerEntityRoutingTable) MarkAllPeersIrreplaceable() {
	this.RoutingTable.MarkAllPeersIrreplaceable()
}

func (this *PeerEntityRoutingTable) ListPeers() []peer.ID {
	return this.RoutingTable.ListPeers()
}

func (this *PeerEntityRoutingTable) Size() int {
	return this.RoutingTable.Size()
}

func (this *PeerEntityRoutingTable) Close() {
	err := this.RoutingTable.Close()
	if err != nil {
		panic(err)
	}
}

func (this *PeerEntityRoutingTable) Print() {
	this.RoutingTable.Print()
}

func (this *PeerEntityRoutingTable) GetDiversityStats() []peerdiversity.CplDiversityStats {
	return this.RoutingTable.GetDiversityStats()
}

func (this *PeerEntityRoutingTable) NPeersForCpl(cpl uint) int {
	return this.RoutingTable.NPeersForCpl(cpl)
}

func (this *PeerEntityRoutingTable) UpdateLastSuccessfulOutboundQueryAt(p peer.ID, t time.Time) bool {
	return this.RoutingTable.UpdateLastSuccessfulOutboundQueryAt(p, t)
}

func (this *PeerEntityRoutingTable) UpdateLastUsefulAt(p peer.ID, t time.Time) bool {
	return this.RoutingTable.UpdateLastUsefulAt(p, t)
}

func (this *PeerEntityRoutingTable) GetTrackedCplsForRefresh() []time.Time {
	return this.RoutingTable.GetTrackedCplsForRefresh()
}

func (this *PeerEntityRoutingTable) GenRandPeerID(targetCpl uint) (peer.ID, error) {
	return this.RoutingTable.GenRandPeerID(targetCpl)
}

func (this *PeerEntityRoutingTable) ResetCplRefreshedAtForID(id kbucket.ID, newTime time.Time) {
	this.RoutingTable.ResetCplRefreshedAtForID(id, newTime)
}
