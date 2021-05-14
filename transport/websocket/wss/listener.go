package wss

import (
	"fmt"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/transport/util"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"net"
	"net/http"
)

type listener struct {
	net.Listener

	laddr ma.Multiaddr

	closed   chan struct{}
	incoming chan *Conn
}

/**
libp2p的wss websocket的实现，与原始的ws相比，最重要的改动就是在此处根据配置启动了tls
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
			err = util.HttpServeTLS(l.Listener, l, cert, key)
		} else {
			// 假如域名存在，使用LetsEncrypt certificates
			if config.TlsParams.Domain != "" {
				err = util.HttpLetsEncryptServe(l.Listener, config.TlsParams.Domain, l)
			} else {
				logger.Sugar.Errorf("No Tls domain: %v")
			}
		}
	} else {
		err = http.Serve(l.Listener, l)
	}
	if err != nil {
		logger.Sugar.Errorf("Start libp2p wss server fail:%v", err.Error())
	}
}

func (l *listener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// The upgrader writes a response for us.
		return
	}

	select {
	case l.incoming <- NewConn(c):
	case <-l.closed:
		c.Close()
	}
	// The connection has been hijacked, it's safe to return.
}

func (l *listener) Accept() (manet.Conn, error) {
	select {
	case c, ok := <-l.incoming:
		if !ok {
			return nil, fmt.Errorf("listener is closed")
		}

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
