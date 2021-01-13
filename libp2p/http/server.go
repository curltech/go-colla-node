package http

import (
	"github.com/curltech/go-colla-node/libp2p/global"
	gostream "github.com/libp2p/go-libp2p-gostream"
	nethttp "net/http"
)

func Start() {
	listener, _ := gostream.Listen(global.Global.Host, DefaultP2PProtocol)
	defer listener.Close()
	go func() {
		nethttp.HandleFunc("/hello", func(w nethttp.ResponseWriter, r *nethttp.Request) {
			w.Write([]byte("Hi!"))
		})
		server := &nethttp.Server{}
		server.Serve(listener)
	}()
}
