package main

import (
	_ "github.com/curltech/go-colla-core/cache"
	libp2p "github.com/curltech/go-colla-node/libp2p"
	"github.com/curltech/go-colla-node/transport/websocket/http"

	/**
	  引入包定义，执行对应包的init函数，从而引入某功能，在init函数根据初始化参数配置决定是否启动该功能
	*/
	_ "github.com/curltech/go-colla-core/content"
	_ "github.com/curltech/go-colla-core/repository/search"
	_ "github.com/curltech/go-colla-node/consensus/std"
	_ "github.com/curltech/go-colla-node/transport/httpproxy"
	//_ "github.com/curltech/go-colla-node/transport/websocket"
	_ "github.com/curltech/go-colla-node/turn/server"
	_ "github.com/curltech/go-colla-node/webrtc"
	_ "github.com/curltech/go-colla-node/webrtc/ion/sfu"
	_ "github.com/curltech/go-colla-node/webrtc/peer"
	_ "github.com/curltech/go-colla-node/webrtc/peer/simplepeer"
)

func main() {
	go http.Start()
	libp2p.Start()
}
