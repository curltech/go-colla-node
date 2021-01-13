package controller

import (
	"curltech.io/camsi/camsi-biz/controller"
	"github.com/curltech/go-colla-core/container"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/p2p/chain/entity"
	service2 "github.com/curltech/go-colla-node/p2p/chain/service"
)

/**
控制层代码需要做数据转换，调用服务层的代码，由于数据转换的结构不一致，因此每个实体（外部rest方式访问）的控制层都需要写一遍
*/
type DataBlockController struct {
	controller.BaseController
}

var dataBlockController *DataBlockController

func (this *DataBlockController) ParseJSON(json []byte) (interface{}, error) {
	var entities = make([]*entity.DataBlock, 0)
	err := message.Unmarshal(json, &entities)

	return &entities, err
}

/**
注册bean管理器，注册序列
*/
func init() {
	dataBlockController = &DataBlockController{
		BaseController: controller.BaseController{
			BaseService: service2.GetDataBlockService(),
		},
	}
	dataBlockController.BaseController.ParseJSON = dataBlockController.ParseJSON
	container.RegistController("datablock", dataBlockController)
}
