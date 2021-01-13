package dht

import (
	"errors"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/kataras/golog"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"time"
)

func PingPing(id peer.ID) error {
	err := PeerEndpointDHT.Ping(id)
	if err != nil {
		golog.Errorf("failed to Ping:%v:%v", id, err)
	} else {
		golog.Infof("successfully Ping:%v", id)
	}

	return err
}

func pingPong(id peer.ID) (time.Duration, error) {
	var err error
	result := ping.Ping(global.Global.Context, global.Global.Host, id)
	for i := 0; i < 5; i++ {
		select {
		case res := <-result:
			if res.Error != nil {
				golog.Errorf(res.Error.Error())
				err = res.Error
			} else {
				golog.Infof("%v service Ping took: %v", id, res.RTT)

				return res.RTT, nil
			}
		case <-time.After(time.Second * 4):
			golog.Errorf("failed to service Ping, timeout")
			err = errors.New("timeout")
		}
	}

	return 0, err
}

func connect(id peer.ID) (peer.AddrInfo, error) {
	addr, err := PeerEndpointDHT.FindPeer(id)
	if err != nil {
		golog.Errorf("failed to FindPeer: %v, err: %v", id.Pretty(), err)
	} else {
		golog.Infof("successfully FindPeer: %v", addr)
	}
	err = global.Global.Host.Connect(global.Global.Context, addr)
	if err != nil {
		golog.Errorf("failed to Connect: %v, err: %v", id.Pretty(), err)
	} else {
		golog.Infof("successfully Connect: %v", id.Pretty())
	}

	return addr, err
}
