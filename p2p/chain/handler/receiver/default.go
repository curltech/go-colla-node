package receiver

import (
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/pipe"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/chain/service"
	msg1 "github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
)

/**
p2p的chain协议处理handler，将原始数据还原成ChainMessage，然后根据消息类型进行分支处理
*/
func HandleChainMessage(data []byte, p *pipe.Pipe) ([]byte, error) {
	var response *msg1.ChainMessage
	remotePeerId := p.GetStream().Conn().RemotePeer().Pretty()
	remoteAddr := p.GetStream().Conn().RemoteMultiaddr().String()
	chainMessage := &msg1.ChainMessage{}
	err := message.Unmarshal(data, chainMessage)
	if err != nil {
		response = handler.Error(msgtype.ERROR, err)
		goto responseProcess
	}
	chainMessage.LocalConnectPeerId = remotePeerId
	chainMessage.LocalConnectAddress = remoteAddr
	if chainMessage.SrcPeerId == "" {
		chainMessage.SrcPeerId = remotePeerId
	}
	if chainMessage.SrcAddress == "" {
		chainMessage.SrcAddress = remoteAddr
	}
	if chainMessage.SrcConnectPeerId == "" {
		chainMessage.SrcConnectPeerId = global.Global.PeerId.Pretty()
	}
	if chainMessage.SrcConnectSessionId == "" {
		chainMessage.SrcConnectSessionId = p.GetStream().Conn().ID()
	}
	chainMessage.ConnectSessionId = p.GetStream().Conn().ID()
	response, err = service.Receive(chainMessage)

responseProcess:
	if response != nil {
		_, err = handler.Encrypt(response)
		if err != nil {
			response = handler.Error(chainMessage.MessageType, err)
		}
		handler.SetResponse(chainMessage, response)
		data, _ := message.Marshal(response)

		return data, nil
	}

	return nil, nil
}
