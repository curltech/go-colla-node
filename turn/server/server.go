package server

import (
   "github.com/curltech/go-colla-core/config"
   "github.com/curltech/go-colla-core/logger"
   "github.com/curltech/go-colla-core/util/message"
   "github.com/curltech/go-colla-node/libp2p/dht"
   "github.com/curltech/go-colla-node/libp2p/ns"
   "github.com/curltech/go-colla-node/p2p/dht/entity"
   "github.com/pion/stun"
   "github.com/pion/turn/v2"
   "net"
   "regexp"
)

var server *turn.Server

// Cache -users flag for easy lookup later
// If passwords are stored they should be saved to your DB hashed using turn.GenerateAuthKey
var usersMap map[string][]byte

/**
turn验证
*/
func authHandler(username string, realm string, srcAddr net.Addr) ([]byte, bool) {
   key := ns.GetPeerClientKey(username)
   recvdVals, err := dht.PeerEndpointDHT.GetValues(key, config.Libp2pParams.Nvals)
   if err == nil {
      for _, recvdVal := range recvdVals {
         pcs := make([]*entity.PeerClient, 0)
         err = message.TextUnmarshal(string(recvdVal.Val), &pcs)
         if err == nil {
            for _, pc := range pcs {
               credential := turn.GenerateAuthKey(pc.PeerId, realm, pc.PeerPublicKey)
               return credential, true
            }
         }
      }
   }
   if credential, ok := usersMap[username]; ok {
      return credential, true
   }
   return nil, false
}

/**
启动turn server
*/
func Start() {
   host := config.TurnParams.Host
   if len(host) == 0 {
      logger.Sugar.Errorf("'host' is required")
      return
   }
   udpport := config.TurnParams.UdpPort
   // Create a UDP listener to pass into pion/turn
   // pion/turn itself doesn't allocate any UDP sockets, but lets the user pass them in
   // this allows us to add logging, storage or modify inbound/outbound traffic
   packetConn, err := net.ListenPacket("udp4", host+":"+udpport)
   if err != nil {
      logger.Sugar.Errorf("Failed to create TURN server listener: %s", err)
      return
   }
   tcpport := config.TurnParams.TcpPort
   tcpListener, err := net.Listen("tcp4", host+":"+tcpport)
   if err != nil {
      logger.Sugar.Errorf("Failed to create TURN server listener: %s", err)
      return
   }
   realm := config.TurnParams.Realm
   // NewLongTermAuthHandler takes a pion.LeveledLogger. This allows you to intercept messages
   // and process them yourself.
   //logger := logging.NewDefaultLeveledLoggerForScope("lt-creds", logging.LogLevelTrace, os.Stdout)
   publicIp := config.TurnParams.Ip
   if len(publicIp) == 0 {
      logger.Sugar.Errorf("'publicIp' is required")
      return
   }
   users := config.TurnParams.Credentials
   usersMap = map[string][]byte{}
   for _, kv := range regexp.MustCompile(`(\w+)=(\w+)`).FindAllStringSubmatch(users, -1) {
      usersMap[kv[1]] = turn.GenerateAuthKey(kv[1], realm, kv[2])
   }
   /**
     创建新的turn服务器
   */
   server, err = turn.NewServer(turn.ServerConfig{
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
      logger.Sugar.Errorf("Failed to create TURN server: %s", err)
      return
   }
   defer func() {
      if err == nil {
         if p := recover(); p != nil {
            logger.Sugar.Errorf("recover failed", p)
            Close()
            panic(p) // re-throw panic when recover failed
         }
      }
   }()
}

func Close() {
   if err := server.Close(); err != nil {
      logger.Sugar.Errorf("Failed to close TURN server: %s", err)
      return
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
         logger.Sugar.Errorf(err.Error())
         logger.Sugar.Errorf("Outbound STUN: %s \n", msg.String())
         return
      }

      logger.Sugar.Debugf("Outbound STUN: %s \n", msg.String())
   }

   return
}

func (s *stunLogger) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
   if n, addr, err = s.PacketConn.ReadFrom(p); err == nil && stun.IsMessage(p) {
      msg := &stun.Message{Raw: p}
      if err = msg.Decode(); err != nil {
         logger.Sugar.Errorf(err.Error())
         logger.Sugar.Errorf("Inbound STUN: %s \n", msg.String())
         return
      }

      logger.Sugar.Debugf("Inbound STUN: %s \n", msg.String())
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
