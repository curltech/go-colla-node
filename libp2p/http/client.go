package http

import (
	"bufio"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/protocol"
	gostream "github.com/libp2p/go-libp2p-gostream"
	"github.com/libp2p/go-libp2p/core/peer"
	"io"
	"net"
	nethttp "net/http"
)

func Connect() {
	tr := &nethttp.Transport{}
	tr.RegisterProtocol("libp2p", NewTransport(global.Global.Host))
	client := &nethttp.Client{Transport: tr}
	res, err := client.Get("libp2p://Qmaoi4isbcTbFfohQyn28EiYM5CDWQx9QRCjDh3CTeiY7P/hello")
	if err != nil {
		logger.Sugar.Infof("response:", res)
	}
}

// DefaultP2PProtocol is used to tag and identify streams
// handled by go-libp2p-stdhttp
var DefaultP2PProtocol protocol.ID = "/libp2p-stdhttp"

// options holds configuration options for the transport.
type options struct {
	Protocol protocol.ID
}

// Option allows to set the libp2p transport options.
type Option func(o *options)

// ProtocolOption sets the Protocol Tag associated to the libp2p roundtripper.
func ProtocolOption(p protocol.ID) Option {
	return func(o *options) {
		o.Protocol = p
	}
}

// RoundTripper implemenets stdhttp.RoundTrip and can be used as
// custom transport with Go stdhttp.Client.
type RoundTripper struct {
	h    host.Host
	opts options
}

// NewTransport returns a new RoundTripper which uses the provided
// libP2P host to perform an stdhttp request and obtain the response.
//
// The typical use case for NewTransport is to register the "libp2p"
// protocol with a Transport, as in:
//     t := &stdhttp.Transport{}
//     t.RegisterProtocol("libp2p", p2phttp.NewTransport(host, ProtocolOption(DefaultP2PProtocol)))
//     c := &stdhttp.Client{Transport: t}
//     res, err := c.Get("libp2p://Qmaoi4isbcTbFfohQyn28EiYM5CDWQx9QRCjDh3CTeiY7P/index.html")
//     ...
func NewTransport(h host.Host, opts ...Option) *RoundTripper {
	defOpts := options{
		Protocol: DefaultP2PProtocol,
	}
	for _, o := range opts {
		o(&defOpts)
	}

	return &RoundTripper{h, defOpts}
}

// we wrap the response body and close the stream
// only when it's closed.
type respBody struct {
	io.ReadCloser
	conn net.Conn
}

// Closes the response's body and the connection.
func (rb *respBody) Close() error {
	rb.conn.Close()
	return rb.ReadCloser.Close()
}

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (rt *RoundTripper) RoundTrip(r *nethttp.Request) (*nethttp.Response, error) {
	addr := r.Host
	if addr == "" {
		addr = r.URL.Host
	}

	pid, err := peer.Decode(addr)
	if err != nil {
		return nil, err
	}

	conn, err := gostream.Dial(r.Context(), rt.h, peer.ID(pid), rt.opts.Protocol)
	if err != nil {
		if r.Body != nil {
			r.Body.Close()
		}
		return nil, err
	}

	// Write the request while reading the response
	go func() {
		err := r.Write(conn)
		if err != nil {
			conn.Close()
		}
		if r.Body != nil {
			r.Body.Close()
		}
	}()

	resp, err := nethttp.ReadResponse(bufio.NewReader(conn), r)
	if err != nil {
		return resp, err
	}

	resp.Body = &respBody{
		ReadCloser: resp.Body,
		conn:       conn,
	}

	return resp, nil
}
