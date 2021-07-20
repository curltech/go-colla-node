module github.com/curltech/go-colla-node

go 1.15

require (
	github.com/ProtonMail/gopenpgp/v2 v2.0.1
	github.com/at-wat/ebml-go v0.11.0
	github.com/curltech/go-colla-core v0.1.6
	github.com/davidlazar/go-crypto v0.0.0-20190912175916-7055855a373f // indirect
	github.com/dgryski/go-farm v0.0.0-20200201041132-a6ae2369ad13 // indirect
	github.com/emersion/go-imap v1.0.6
	github.com/emersion/go-imap-idle v0.0.0-20201224103203-6f42b9020098
	github.com/emersion/go-message v0.14.1
	github.com/emersion/go-sasl v0.0.0-20200509203442-7bfe0ed36a21
	github.com/emersion/go-smtp v0.14.0
	github.com/fasthttp/websocket v1.4.1
	github.com/foxcpp/maddy v0.4.3
	github.com/gogo/protobuf v1.3.2
	github.com/google/uuid v1.2.0
	github.com/gorilla/websocket v1.4.2
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-ds-badger2 v0.1.1-0.20200708190120-187fc06f714e
	github.com/ipfs/go-ds-bolt v0.0.1
	github.com/ipfs/go-ds-flatfs v0.4.5
	github.com/ipfs/go-ds-leveldb v0.4.2
	github.com/ipfs/go-ds-redis v0.1.0
	github.com/ipfs/go-ds-sql v0.2.0
	github.com/ipfs/go-filestore v1.0.0 // indirect
	github.com/ipfs/go-ipfs v0.8.0
	github.com/ipfs/go-ipfs-blockstore v1.0.1 // indirect
	github.com/ipfs/go-ipfs-config v0.12.0
	github.com/ipfs/go-ipfs-files v0.0.8
	github.com/ipfs/go-ipfs-util v0.0.2
	github.com/ipfs/interface-go-ipfs-core v0.4.0
	github.com/jbenet/goprocess v0.1.4
	github.com/libp2p/go-libp2p v0.13.0
	github.com/libp2p/go-libp2p-autonat v0.4.0
	github.com/libp2p/go-libp2p-circuit v0.4.0
	github.com/libp2p/go-libp2p-connmgr v0.2.4
	github.com/libp2p/go-libp2p-core v0.8.5
	github.com/libp2p/go-libp2p-discovery v0.5.0
	github.com/libp2p/go-libp2p-gostream v0.3.0
	github.com/libp2p/go-libp2p-kad-dht v0.12.0
	github.com/libp2p/go-libp2p-kbucket v0.4.7
	github.com/libp2p/go-libp2p-mplex v0.4.1
	github.com/libp2p/go-libp2p-noise v0.1.2
	github.com/libp2p/go-libp2p-peerstore v0.2.7
	github.com/libp2p/go-libp2p-pubsub v0.4.1
	github.com/libp2p/go-libp2p-quic-transport v0.10.0
	github.com/libp2p/go-libp2p-record v0.1.3
	github.com/libp2p/go-libp2p-secio v0.2.2
	github.com/libp2p/go-libp2p-tls v0.1.3
	github.com/libp2p/go-libp2p-transport-upgrader v0.4.0
	github.com/libp2p/go-tcp-transport v0.2.1
	github.com/libp2p/go-ws-transport v0.4.0
	github.com/multiformats/go-base32 v0.0.3
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/multiformats/go-multiaddr-fmt v0.1.0
	github.com/multiformats/go-multiaddr-net v0.2.0
	github.com/panjf2000/gnet v1.3.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/phachon/fasthttpsession v0.0.0-20201116052147-eb400f0616cb
	github.com/pion/ion-log v1.0.0
	github.com/pion/ion-sdk-go v0.1.4
	github.com/pion/ion-sfu v1.9.8
	github.com/pion/logging v0.2.2
	github.com/pion/rtcp v1.2.6
	github.com/pion/rtp v1.6.2
	github.com/pion/sdp/v2 v2.4.0
	github.com/pion/stun v0.3.5
	github.com/pion/turn/v2 v2.0.5
	github.com/pion/webrtc/v3 v3.0.20
	github.com/valyala/fasthttp v1.16.0
	github.com/whyrusleeping/cbor-gen v0.0.0-20200826160007-0b9f6c5fb163 // indirect
	github.com/yeqown/fasthttp-reverse-proxy/v2 v2.1.4
	github.com/yeqown/log v1.0.5 // indirect
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/image v0.0.0-20200927104501-e162460cd6b5
)

replace golang.org/x/crypto => github.com/ProtonMail/crypto v0.0.0-20200416114516-1fa7f403fb9c
