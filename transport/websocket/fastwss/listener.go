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
	"net/http"
)

type listener struct {
	net.Listener

	laddr ma.Multiaddr

	closed   chan struct{}
	incoming chan *Conn
}

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
			//err = util.FastHttpServeTLS(l.Listener, l, cert, key)
		} else {
			// 假如域名存在，使用LetsEncrypt certificates
			if config.TlsParams.Domain != "" {
				err = util.HttpLetsEncryptServe(l.Listener, config.TlsParams.Domain, l)
				//err = util.FastHttpLetsEncryptServe(l.Listener, config.TlsParams.Domain, l)
			} else {
				logger.Sugar.Errorf("No Tls domain: %v")
			}
		}
	} else {
		err = http.Serve(l.Listener, l)
		//err = fasthttp.Serve(l.Listener, l)
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

/**
没完成
*/
func (l *listener) fastHandler(ctx *fasthttp.RequestCtx) {
	upgrader := fastwebsocket.FastHTTPUpgrader{
		CheckOrigin: func(ctx *fasthttp.RequestCtx) bool { return true },
	}
	err := upgrader.Upgrade(ctx, func(ws *fastwebsocket.Conn) {
		defer ws.Close()
		for {
			mt, message, err := ws.ReadMessage()
			if err != nil {
				logger.Sugar.Errorf("read error:", err.Error())
				break
			}
			logger.Sugar.Infof("recv: %s", message)
			err = ws.WriteMessage(mt, message)
			if err != nil {
				logger.Sugar.Errorf("write error:", err.Error())
				break
			}
		}
	})

	if err != nil {
		if _, ok := err.(fastwebsocket.HandshakeError); ok {
			logger.Sugar.Errorf(err.Error())
		}
		return
	}

	logger.Sugar.Infof("conn done")
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