package client

import (
	"fmt"
	"github.com/curltech/go-colla-core/logger"
	"net"
	"strings"
	"time"

	"github.com/pion/logging"
	"github.com/pion/turn/v2"
)

type tcpClient struct {
	client    *turn.Client
	relayConn net.PacketConn
}

var TcpClient = &tcpClient{}

func (this *tcpClient) Dial(host string, port string, user string, realm string) {
	if len(host) == 0 {
		logger.Sugar.Errorf("'host' is required")
		return
	}

	if len(user) == 0 {
		logger.Sugar.Errorf("'user' is required")
		return
	}

	cred := strings.SplitN(user, "=", 2)
	if len(cred) != 2 {
		logger.Sugar.Errorf("'user' must be a=b format")
		return
	}

	// Dial TURN Server
	addr := fmt.Sprintf("%s:%s", host, port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		logger.Sugar.Errorf(err.Error())
		return
	}
	// Start a new TURN Client and wrap our net.Conn in a STUNConn
	// This allows us to simulate datagram based communication over a net.Conn
	cfg := &turn.ClientConfig{
		STUNServerAddr: addr,
		TURNServerAddr: addr,
		Conn:           turn.NewSTUNConn(conn),
		Username:       cred[0],
		Password:       cred[1],
		Realm:          realm,
		LoggerFactory:  logging.NewDefaultLoggerFactory(),
	}

	this.client, err = turn.NewClient(cfg)
	if err != nil {
		logger.Sugar.Errorf(err.Error())
		return
	}

	// Start listening on the conn provided.
	err = this.client.Listen()
	if err != nil {
		logger.Sugar.Errorf(err.Error())
		return
	}

	// Allocate a relay socket on the TURN server. On success, it
	// will return a net.PacketConn which represents the remote
	// socket.
	this.relayConn, err = this.client.Allocate()
	if err != nil {
		logger.Sugar.Errorf(err.Error())
		return
	}

	// The relayConn's local address is actually the transport
	// address assigned on the TURN server.
	logger.Sugar.Infof("relayed-address=%s", this.relayConn.LocalAddr().String())
}

func (this *tcpClient) Close() {
	if this.relayConn != nil {
		if closeErr := this.relayConn.Close(); closeErr != nil {
			logger.Sugar.Errorf(closeErr.Error())
			return
		}
	}
	if this.client != nil {
		this.client.Close()
	}
}

func (this *tcpClient) Ping() error {
	// Send BindingRequest to learn our external IP
	mappedAddr, err := this.client.SendBindingRequest()
	if err != nil {
		logger.Sugar.Errorf(err.Error())
		return err
	}

	// Set up pinger socket (pingerConn)
	pingerConn, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		logger.Sugar.Errorf(err.Error())
		return err
	}
	defer func() {
		if closeErr := pingerConn.Close(); closeErr != nil {
			logger.Sugar.Errorf(closeErr.Error())
			return
		}
	}()

	// Punch a UDP hole for the relayConn by sending a data to the mappedAddr.
	// This will trigger a TURN client to generate a permission request to the
	// TURN server. After this, packets from the IP address will be accepted by
	// the TURN server.
	_, err = this.relayConn.WriteTo([]byte("Hello"), mappedAddr)
	if err != nil {
		logger.Sugar.Errorf(err.Error())
		return err
	}

	// Start read-loop on pingerConn
	go func() {
		buf := make([]byte, 1500)
		for {
			n, from, pingerErr := pingerConn.ReadFrom(buf)
			if pingerErr != nil {
				logger.Sugar.Errorf(pingerErr.Error())
				break
			}

			msg := string(buf[:n])
			if sentAt, pingerErr := time.Parse(time.RFC3339Nano, msg); pingerErr == nil {
				rtt := time.Since(sentAt)
				logger.Sugar.Infof("%d bytes from from %s time=%d ms\n", n, from.String(), int(rtt.Seconds()*1000))
			}
		}
	}()

	// Start read-loop on relayConn
	go func() {
		buf := make([]byte, 1500)
		for {
			n, from, readerErr := this.relayConn.ReadFrom(buf)
			if readerErr != nil {
				logger.Sugar.Errorf(readerErr.Error())
				break
			}

			// Echo back
			if _, readerErr = this.relayConn.WriteTo(buf[:n], from); readerErr != nil {
				logger.Sugar.Errorf(readerErr.Error())
				break
			}
		}
	}()

	time.Sleep(500 * time.Millisecond)

	// Send 10 packets from relayConn to the echo server
	for i := 0; i < 10; i++ {
		msg := time.Now().Format(time.RFC3339Nano)
		_, err = pingerConn.WriteTo([]byte(msg), this.relayConn.LocalAddr())
		if err != nil {
			logger.Sugar.Errorf(err.Error())
			return err
		}

		// For simplicity, this example does not wait for the pong (reply).
		// Instead, sleep 1 second.
		time.Sleep(time.Second)
	}

	return nil
}
