package ipfs

import (
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-node/ipfs/server"
)

/**
作为ipfs的节点启动
*/
func Start() {
	if config.IpfsParams.Enable {
		go server.Start()
	}
}
