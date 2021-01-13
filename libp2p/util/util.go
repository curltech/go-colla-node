package util

import (
	"fmt"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/kataras/golog"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

func GetStringAddr(addr string, peerId string) string {
	return fmt.Sprintf("%v/p2p/%v", addr, peerId)
}

func GetIdAddr(saddr string) (string, string) {
	addr, err := multiaddr.NewMultiaddr(saddr)
	if err != nil {
		return "", ""
	}
	addrinfo, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return "", ""
	}
	return addrinfo.ID.String(), addrinfo.Addrs[0].String()
}

func ToString(addrs []multiaddr.Multiaddr) []string {
	var saddrs = make([]string, 0)
	for _, addr := range addrs {
		saddr := addr.String()
		saddrs = append(saddrs, saddr)
	}

	return saddrs
}

func ToMultiaddr(saddrs []string) []multiaddr.Multiaddr {
	var addrs = make([]multiaddr.Multiaddr, 0)
	for _, saddr := range saddrs {
		addr, err := multiaddr.NewMultiaddr(saddr)
		if err != nil {
			continue
		}
		addrs = append(addrs, addr)
	}

	return addrs
}

func MultiaddrToAddInfo(addrs []multiaddr.Multiaddr) []*peer.AddrInfo {
	var addrInfos = make([]*peer.AddrInfo, 0)
	for _, addr := range addrs {
		addrinfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			continue
		}
		addrInfos = append(addrInfos, addrinfo)
	}

	return addrInfos
}

func ToAddInfo(peerId string, saddr string) (*peer.AddrInfo, error) {
	saddr = GetStringAddr(saddr, peerId)
	addr, err := multiaddr.NewMultiaddr(saddr)
	if err != nil {
		golog.Errorf("addr:%v can't build Multiaddr", saddr)
		return nil, err
	}
	addrInfo, err := peer.AddrInfoFromP2pAddr(addr)

	return addrInfo, err
}

func ToAddInfos(peerId string, address string) ([]*peer.AddrInfo, error) {
	var saddrs = make([]string, 0)
	err := message.TextUnmarshal(address, &saddrs)
	if err != nil {
		return nil, err
	}
	addrInfos := make([]*peer.AddrInfo, 0)
	for _, saddr := range saddrs {
		addrInfo, err := ToAddInfo(peerId, saddr)
		if err != nil {
			continue
		}
		addrInfos = append(addrInfos, addrInfo)
	}

	return addrInfos, nil
}

func Merge(addrInfos []*peer.AddrInfo) *peer.AddrInfo {
	addrInfo := peer.AddrInfo{}
	addrs := make([]multiaddr.Multiaddr, 0)
	var id peer.ID = ""
	for _, addrInfo := range addrInfos {
		if id == "" {
			id = addrInfo.ID
		} else if id != addrInfo.ID {
			continue
		}
		addrs = append(addrs, addrInfo.Addrs...)
	}
	addrInfo.Addrs = addrs
	addrInfo.ID = peer.ID(id)

	return &addrInfo
}
