package wss

import (
	"fmt"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"golang.org/x/crypto/acme/autocert"
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
			err = http.ServeTLS(l.Listener, l, cert, key)
			//err = http.Serve(l.Listener, l)
		} else {
			// 假如域名存在，使用LetsEncrypt certificates
			if config.TlsParams.Domain != "" {
				logger.Sugar.Infof("Domain specified, using LetsEncrypt to autogenerate and serve certs for %s\n", config.TlsParams.Domain)
				m := &autocert.Manager{
					Cache:      autocert.DirCache("certs"),
					Prompt:     autocert.AcceptTOS,
					HostPolicy: autocert.HostWhitelist(config.TlsParams.Domain),
				}
				server := &http.Server{
					Addr:      config.TlsParams.Domain,
					TLSConfig: m.TLSConfig(),
				}
				server.Handler = l
				logger.Sugar.Infof("libp2p wss calls from wss://%s to %s with LetsEncrypt started!")
				err = server.ListenAndServeTLS("", "")
				if err != nil {
					logger.Sugar.Errorf("failed to server.ListenAndServeTLS: %v", err.Error())
				}
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
