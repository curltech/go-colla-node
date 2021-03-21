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

type udpClient struct {
	client    *turn.Client
	relayConn net.PacketConn
}

var UdpClient = &udpClient{}

func (this *udpClient) Dial(host string, port string, user string, realm string) {
	cred := strings.SplitN(user, "=", 2)

	// TURN client won't create a local listening socket by itself.
	conn, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		logger.Sugar.Errorf(err.Error())
		return
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			panic(closeErr)
		}
	}()

	addr := fmt.Sprintf("%s:%d", host, port)

	cfg := &turn.ClientConfig{
		STUNServerAddr: addr,
		TURNServerAddr: addr,
		Conn:           conn,
		Username:       cred[0],
		Password:       cred[1],
		Realm:          realm,
		LoggerFactory:  logging.NewDefaultLoggerFactory(),
	}

	client, err := turn.NewClient(cfg)
	if err != nil {
		logger.Sugar.Errorf(err.Error())
		return
	}

	// Start listening on the conn provided.
	err = client.Listen()
	if err != nil {
		logger.Sugar.Errorf(err.Error())
		return
	}

	// Allocate a relay socket on the TURN server. On success, it
	// will return a net.PacketConn which represents the remote
	// socket.
	relayConn, err := client.Allocate()
	if err != nil {
		logger.Sugar.Errorf(err.Error())
		return
	}

	// The relayConn's local address is actually the transport
	// address assigned on the TURN server.
	logger.Sugar.Infof("relayed-address=%s", relayConn.LocalAddr().String())
}

func (this *udpClient) Ping() error {
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
