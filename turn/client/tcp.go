package client

import (
	"fmt"
	"github.com/curltech/go-colla-core/logger"
	"log"
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
	// Dial TURN Server
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err)
	}

	cred := strings.SplitN(user, "=", 2)
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

	client, err := turn.NewClient(cfg)
	if err != nil {
		panic(err)
	}
	this.client = client

	// Start listening on the conn provided.
	err = client.Listen()
	if err != nil {
		panic(err)
	}

	// Allocate a relay socket on the TURN server. On success, it
	// will return a net.PacketConn which represents the remote
	// socket.
	relayConn, err := client.Allocate()
	if err != nil {
		panic(err)
	}
	this.relayConn = relayConn

	// The relayConn's local address is actually the transport
	// address assigned on the TURN server.
	logger.Sugar.Infof("relayed-address=%s", relayConn.LocalAddr().String())
}

func (this *tcpClient) Close() {
	this.client.Close()
	if closeErr := this.relayConn.Close(); closeErr != nil {
		panic(closeErr)
	}
}

func (this *tcpClient) Ping() error {
	// Send BindingRequest to learn our external IP
	mappedAddr, err := this.client.SendBindingRequest()
	if err != nil {
		return err
	}

	// Set up pinger socket (pingerConn)
	pingerConn, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		panic(err)
	}
	defer func() {
		if closeErr := pingerConn.Close(); closeErr != nil {
			panic(closeErr)
		}
	}()

	// Punch a UDP hole for the relayConn by sending a data to the mappedAddr.
	// This will trigger a TURN client to generate a permission request to the
	// TURN server. After this, packets from the IP address will be accepted by
	// the TURN server.
	_, err = this.relayConn.WriteTo([]byte("Hello"), mappedAddr)
	if err != nil {
		return err
	}

	// Start read-loop on pingerConn
	go func() {
		buf := make([]byte, 1500)
		for {
			n, from, pingerErr := pingerConn.ReadFrom(buf)
			if pingerErr != nil {
				break
			}

			msg := string(buf[:n])
			if sentAt, pingerErr := time.Parse(time.RFC3339Nano, msg); pingerErr == nil {
				rtt := time.Since(sentAt)
				log.Printf("%d bytes from from %s time=%d ms\n", n, from.String(), int(rtt.Seconds()*1000))
			}
		}
	}()

	// Start read-loop on relayConn
	go func() {
		buf := make([]byte, 1500)
		for {
			n, from, readerErr := this.relayConn.ReadFrom(buf)
			if readerErr != nil {
				break
			}

			// Echo back
			if _, readerErr = this.relayConn.WriteTo(buf[:n], from); readerErr != nil {
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
			return err
		}

		// For simplicity, this example does not wait for the pong (reply).
		// Instead, sleep 1 second.
		time.Sleep(time.Second)
	}

	return nil
}
