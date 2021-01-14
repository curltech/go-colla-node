package handler

import (
	"fmt"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/dht"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/pipe"
	"github.com/curltech/go-colla-node/p2p/handler"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"strings"
)

/**
libp2p的流处理模块把接收的数据分发到这里，通用的消息处理分发器根据stream的protocolID作下一步处理分发
比如，chain协议的将进一步分发到p2p的chain协议处理handler
*/
func HandleRaw(data []byte, p *pipe.Pipe) ([]byte, error) {
	stream := p.GetStream()
	protocolID := stream.Protocol()
	protocolMessageHandler, err := handler.GetProtocolMessageHandler(string(protocolID))
	if err != nil {

	}
	//调用Receive函数或者Response函数处理
	data, err = protocolMessageHandler.ReceiveHandler(data, p)
	//如果处理器返回了Response，则写回到原来的管道，并关闭管道
	if data != nil {
		p.Write(data, false)
	}
	p.Reset()
	logger.Infof("read data:%v", string(data))
	logger.Infof("read protocolID:%v", protocolID)

	return data, nil
}

type Sender interface {
	SendRaw(peerId string, address string, protocolId string, data []byte) ([]byte, error)
}

/**
如果输入peerId没有地址信息，通过路由表获取完整的地址信息
如果不在路由表中则原样返回，这地方有个问题，定位器有地址信息，客户端没有地址信息
*/
func GetAddrInfo(peerId string) string {
	ps := strings.Split(peerId, "/")
	if len(ps) == 1 {
		id, err := peer.Decode(peerId)
		if err == nil {
			addrInfo, err := dht.PeerEndpointDHT.FindPeer(id)
			if err == nil && len(addrInfo.Addrs) > 0 {
				peerId = fmt.Sprintf(global.GeneralP2pAddrFormat, addrInfo.Addrs[0], addrInfo.ID)
			}
		}
	}

	return peerId
}

/**
使用特定的协议，peerId和地址，利用原有的或者创建新的管道发送数据
这里有个冲突，就是发送给定位器，peerId带地址，发送给PeerClient则没有地址，如果分成两个方法也许更好
*/
func SendRaw(peerId string, protocolId string, data []byte) ([]byte, error) {
	peerId = GetAddrInfo(peerId)
	pipe, err := GetPipePool().GetRequestPipe(peerId, protocolId)
	if err != nil {
		logger.Errorf("createPipe failure")

		return nil, err
	}

	logger.Infof("Write data length:%v", len(data))
	pipe, _, err = pipe.Write(data, false)
	if err != nil {
		logger.Errorf("pipe.Write failure")

		return nil, err
	}

	return data, nil
}

/**
根据配置的协议编号自定义流协议，其他peer连接自己的时候，用于在节点间接收和发送数据
*/
func ProtocolStream(protocolID protocol.ID) {
	global.Global.Host.SetStreamHandler(protocolID, func(stream network.Stream) {
		CreatePipe(stream, msgtype.MsgDirect_Response)
	})
}
