package handler

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/pipe"
	"github.com/curltech/go-colla-node/p2p/handler"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
)

/*
*
libp2p的流处理模块把接收的数据分发到这里，通用的消息处理分发器根据stream的protocolID作下一步处理分发
比如，chain协议的将进一步分发到p2p的chain协议处理handler
*/
func HandleRaw(data []byte, p *pipe.Pipe) ([]byte, error) {
	stream := p.GetStream()
	protocolID := stream.Protocol()
	protocolMessageHandler, err := handler.GetProtocolMessageHandler(string(protocolID))
	if err != nil {
		logger.Sugar.Errorf(err.Error())
	}
	//调用Receive函数或者Response函数处理
	sessId := p.GetStream().Conn().ID()
	remotePeerId := p.GetStream().Conn().RemotePeer().Pretty()
	remoteAddr := p.GetStream().Conn().RemoteMultiaddr().String()
	ResponsePipePool[sessId] = p
	data, err = protocolMessageHandler.MessageHandler(data, remotePeerId, "", sessId, remoteAddr)
	//如果处理器返回了Response，则写回到原来的管道，并关闭管道
	if data != nil {
		logger.Sugar.Debugf("read data:%v", string(data))
		logger.Sugar.Debugf("read protocolID:%v", protocolID)
		_, _, err := p.Write(data, false)
		if err != nil {
			logger.Sugar.Errorf("HandleRaw-pipe.Write failure: %v", err)
		}
	}
	p.Reset()

	return data, nil
}

/*
*
根据配置的协议编号自定义流协议，其他peer连接自己的时候，用于在节点间接收和发送数据
*/
func ProtocolStream(protocolID protocol.ID) {
	global.Global.Host.SetStreamHandler(protocolID, func(stream network.Stream) {
		CreatePipe(stream, msgtype.MsgDirect_Response)
	})
}
