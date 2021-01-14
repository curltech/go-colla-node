package server

import (
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/pion/stun"
	"github.com/pion/turn/v2"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func authHandler(username string, realm string, srcAddr net.Addr) ([]byte, bool) {
	//cred := strings.SplitN(username, "=", 2)
	//user, err := service.GetUserService().Auth(cred[0], cred[1])
	//if err == nil {
	//	return []byte(user.UserId), true
	//}
	return nil, false
}

func Start() {
	host := config.TurnParams.Host
	if len(host) == 0 {
		logger.Errorf("'host' is required")
	}
	udpport := config.TurnParams.UdpPort
	// Create a UDP listener to pass into pion/turn
	// pion/turn itself doesn't allocate any UDP sockets, but lets the user pass them in
	// this allows us to add logging, storage or modify inbound/outbound traffic
	packetConn, err := net.ListenPacket("udp4", host+":"+udpport)
	if err != nil {
		logger.Errorf("Failed to create TURN server listener: %s", err)
	}
	tcpport := config.TurnParams.TcpPort
	tcpListener, err := net.Listen("tcp4", host+":"+tcpport)
	if err != nil {
		logger.Errorf("Failed to create TURN server listener: %s", err)
	}
	realm := config.TurnParams.Realm
	// NewLongTermAuthHandler takes a pion.LeveledLogger. This allows you to intercept messages
	// and process them yourself.
	//logger := logging.NewDefaultLeveledLoggerForScope("lt-creds", logging.LogLevelTrace, os.Stdout)
	publicIp := config.TurnParams.Ip
	if len(publicIp) == 0 {
		publicIp = host
		logger.Errorf("'host' is required")
	}
	s, err := turn.NewServer(turn.ServerConfig{
		Realm: realm,
		// Set AuthHandler callback
		// This is called everytime a user tries to authenticate with the TURN server
		// Return the key for that user, or false when no user is found
		AuthHandler: authHandler,
		//AuthHandler: turn.NewLongTermAuthHandler(*authSecret, logger),
		// PacketConnConfigs is a list of UDP Listeners and the configuration around them
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: &stunLogger{packetConn},
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(publicIp), // Claim that we are listening on IP passed by user (This should be your Public IP)
					Address:      "0.0.0.0",             // But actually be listening on every interface
				},
			},
		},
		// ListenerConfig is a list of Listeners and the configuration around them
		ListenerConfigs: []turn.ListenerConfig{
			{
				Listener: tcpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(publicIp),
					Address:      "0.0.0.0",
				},
			},
		},
	})
	if err != nil {
		logger.Errorf("Failed to create TURN server: %s", err)
	}

	// Block until user sends SIGINT or SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	if err = s.Close(); err != nil {
		logger.Errorf("Failed to close TURN server: %s", err)
	}

	//turn.GenerateLongTermCredentials(*authSecret, time.Minute)
}

// stunLogger wraps a PacketConn and prints incoming/outgoing STUN packets
// This pattern could be used to capture/inspect/modify data as well
type stunLogger struct {
	net.PacketConn
}

func (s *stunLogger) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if n, err = s.PacketConn.WriteTo(p, addr); err == nil && stun.IsMessage(p) {
		msg := &stun.Message{Raw: p}
		if err = msg.Decode(); err != nil {
			return
		}

		logger.Infof("Outbound STUN: %s \n", msg.String())
	}

	return
}

func (s *stunLogger) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if n, addr, err = s.PacketConn.ReadFrom(p); err == nil && stun.IsMessage(p) {
		msg := &stun.Message{Raw: p}
		if err = msg.Decode(); err != nil {
			return
		}

		logger.Infof("Inbound STUN: %s \n", msg.String())
	}

	return
}

// attributeAdder wraps a PacketConn and appends the SOFTWARE attribute to STUN packets
// This pattern could be used to capture/inspect/modify data as well
type attributeAdder struct {
	net.PacketConn
}

func (s *attributeAdder) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if stun.IsMessage(p) {
		m := &stun.Message{Raw: p}
		if err = m.Decode(); err != nil {
			return
		}

		if err = stun.NewSoftware("CustomTURNServer").AddTo(m); err != nil {
			return
		}

		m.Encode()
		p = m.Raw
	}

	return s.PacketConn.WriteTo(p, addr)
}

func init() {
	enable := config.TurnParams.Enable
	if enable {
		go Start()
	}
}
