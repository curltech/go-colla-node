package fastwss

import (
	"fmt"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/transport/util"
	fastwebsocket "github.com/fasthttp/websocket"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/valyala/fasthttp"
	"net"
)

type listener struct {
	net.Listener

	laddr ma.Multiaddr

	closed   chan struct{}
	incoming chan *Conn
}

/**
libp2p的wss fasthttp websocket的实现，与原始的ws相比，最重要的改动就是在此处根据配置启动了fasthttp和tls
*/
func (l *listener) serve() {
	defer close(l.closed)
	tlsmode := config.TlsParams.Mode
	var err error
	logger.Sugar.Infof("Start libp2p wss server")
	if config.Libp2pParams.EnableWss {
		if tlsmode == "cert" {
			cert := config.TlsParams.Cert
			key := config.TlsParams.Key
			err = util.FastHttpServeTLS(l.Listener, l.ServeHTTP, cert, key)
		} else {
			// 假如域名存在，使用LetsEncrypt certificates
			if config.TlsParams.Domain != "" {
				err = util.FastHttpLetsEncryptServe(l.Listener, config.TlsParams.Domain, l.ServeHTTP)
			} else {
				logger.Sugar.Errorf("No Tls domain: %v")
			}
		}
	} else {
		err = fasthttp.Serve(l.Listener, l.ServeHTTP)
	}
	if err != nil {
		logger.Sugar.Errorf("Start libp2p wss server fail:%v", err.Error())
	}
}

/**
fasthttp实际的处理方法
*/
func (l *listener) ServeHTTP(ctx *fasthttp.RequestCtx) {
	upgrader := fastwebsocket.FastHTTPUpgrader{
		CheckOrigin: func(ctx *fasthttp.RequestCtx) bool { return true },
	}
	err := upgrader.Upgrade(ctx, func(c *fastwebsocket.Conn) {
		select {
		//当连接建立的时候，填充输入管道
		case l.incoming <- NewConn(c):
		case <-l.closed:
			c.Close()
		}
		// The connection has been hijacked, it's safe to return.
	})

	if err != nil {
		if _, ok := err.(fastwebsocket.HandshakeError); ok {
			logger.Sugar.Errorf(err.Error())
		}
		return
	}
}

func (l *listener) Accept() (manet.Conn, error) {
	select {
	case c, ok := <-l.incoming:
		if !ok {
			return nil, fmt.Errorf("listener is closed")
		}

		//新的连接到来的时候
		mnc, err := manet.WrapNetConn(c)
		if err != nil {
			c.Close()
			return nil, err
		}

		return mnc, nil
	case <-l.closed:
		return nil, fmt.Errorf("listener is closed")
	}
}

func (l *listener) Multiaddr() ma.Multiaddr {
	return l.laddr
}
