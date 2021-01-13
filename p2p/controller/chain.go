package controller

import (
	"github.com/curltech/go-colla-core/container"
	"github.com/curltech/go-colla-node/p2p/chain/action/dht"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/chain/service"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/kataras/iris/v12"
	"time"
)

type ChainController struct {
}

var chainController = &ChainController{}

/**
从http控制层接收ChainMessage
*/
func (this *ChainController) Receive(ctx iris.Context) {
	chainMessage := &msg.ChainMessage{}
	ctx.ReadJSON(&chainMessage)
	chainMessage.LocalConnectAddress = ctx.RemoteAddr()
	response, err := service.Receive(chainMessage)
	handler.SetResponse(chainMessage, response)
	if err != nil {
		ctx.JSON(err.Error())
	} else {
		ctx.JSON(response)
	}
}

/**
从http控制层Ping
*/
func (this *ChainController) Ping(ctx iris.Context) {
	param := P2pParam{}
	ctx.ReadJSON(&param)
	if param.PeerId == "" {
		ctx.StopWithJSON(iris.StatusInternalServerError, "NoPeerId")

		return
	}
	start := time.Now()
	_, err := dht.PingAction.Ping(param.PeerId, param.TargetPeerId)
	interval := time.Since(start)
	if err != nil {
		ctx.StopWithJSON(iris.StatusInternalServerError, err.Error())

		return
	}
	result := make(map[string]interface{})
	result["start"] = start
	result["interval"] = interval
	ctx.JSON(&result)
}

/**
从http控制层Chat
*/
func (this *ChainController) Chat(ctx iris.Context) {
	param := P2pParam{}
	ctx.ReadJSON(&param)
	if param.PeerId == "" {
		ctx.StopWithJSON(iris.StatusInternalServerError, "NoPeerId")

		return
	}
	if param.Value == "" {
		ctx.StopWithJSON(iris.StatusInternalServerError, "NoValue")

		return
	}
	start := time.Now()
	response, err := dht.ChatAction.Chat(param.PeerId, param.PayloadType, param.Value, param.TargetPeerId)
	if err != nil {
		ctx.StopWithJSON(iris.StatusInternalServerError, err.Error())

		return
	}
	interval := time.Since(start)
	result := make(map[string]interface{})
	result["start"] = start
	result["interval"] = interval
	result["data"] = response
	ctx.JSON(&result)
}

func init() {
	container.RegistController("chain", chainController)
}
