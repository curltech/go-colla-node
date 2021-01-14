package handler

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/ipfs/go-datastore"
)

var dsServiceContainer = make(map[string]datastore.Datastore)

var keynamePool = make(map[string]string)

func RegistKeyname(name string, keyname string) {
	keynamePool[name] = keyname
}

func RegistDatastore(name string, ds datastore.Datastore) {
	var c = dsServiceContainer
	_, ok := c[name]
	if !ok {
		c[name] = ds
		logger.Infof("bean:%v registed", name)
	} else {
		logger.Warnf("bean:%v exist", name)
	}
}

func GetDatastore(name string) datastore.Datastore {
	var c = dsServiceContainer
	old, ok := c[name]
	if ok {
		return old
	}

	return nil
}

func init() {

}
