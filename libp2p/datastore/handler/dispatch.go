package handler

import (
	"errors"
	"github.com/curltech/go-colla-core/container"
	"github.com/curltech/go-colla-core/logger"
	service2 "github.com/curltech/go-colla-core/service"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	"github.com/multiformats/go-base32"
	"strings"
)

/**
dht datastore的数据库实现和elasticsearch实现
*/

// DispatchDatastore uses a standard Go map for internal storage.
type DispatchDatastore struct {
}

type DispatchRequest struct {
	Name      string
	Keyname   string
	Datastore datastore.Datastore
	Service   service2.BaseService
	Keyvalue  map[string]string
}

func NewKeyRequest(key datastore.Key) (*DispatchRequest, error) {
	keyId := strings.TrimPrefix(key.String(), "/")
	buf, err := base32.RawStdEncoding.DecodeString(keyId)
	if err != nil {
		return nil, nil
	}
	path := string(buf)
	segs := strings.Split(path, "/")
	prefix := segs[1]
	request, err := NewPrefixRequest(prefix)
	if err != nil {
		return nil, err
	}
	/**
	在delete，put操作的时候，segs[2]也就是key必须是唯一确定数据的主键和唯一索引，因为它决定了要删除和保存的数据存放的节点
	最近节点是根据这个决定的
	在get的时候，最好也是主键或者唯一索引，并且与put的时候的值相同，假如需要作更复杂的搜索，可以让key成为一个json
	但是，网络在递归搜索节点的时候会根据key的值来决定搜索的节点，因此是盲目的，搜索量会很大，
	现在的实现是如果不是json默认是主键或者唯一索引的模式
	如果自己实现递归搜索也是一种方案
	*/
	Keyvalue := make(map[string]string, 0)
	err = message.TextUnmarshal(segs[2], &Keyvalue)
	if err != nil {
		Keyvalue[request.Keyname] = segs[2]
	}
	request.Keyvalue = Keyvalue

	return request, nil
}

func NewPrefixRequest(prefix string) (*DispatchRequest, error) {
	ds := GetDatastore(prefix)
	if ds == nil {
		logger.Sugar.Errorf("No datastore:%v", prefix)
		return nil, errors.New("No datastore")
	}
	s := container.GetService(prefix)
	var svc service2.BaseService
	if s == nil {
		logger.Sugar.Warnf("No service:%v", prefix)
	} else {
		svc = s.(service2.BaseService)
	}
	keyname, ok := keynamePool[prefix]
	if !ok {
		logger.Sugar.Errorf("No keyname:%v", prefix)
		return nil, errors.New("No keyname")
	}

	return &DispatchRequest{Name: prefix, Keyname: keyname, Datastore: ds, Service: svc}, nil
}

func NewDispatchDatastore() (this *DispatchDatastore) {
	return &DispatchDatastore{}
}

// Put implements Datastore.Put
func (this *DispatchDatastore) Put(key datastore.Key, value []byte) (err error) {
	request, err := NewKeyRequest(key)
	if err != nil {
		return err
	}
	return request.Datastore.Put(key, value)
}

// Sync implements Datastore.Sync
func (this *DispatchDatastore) Sync(prefix datastore.Key) error {
	request, err := NewKeyRequest(prefix)
	if err != nil {
		return err
	}
	return request.Datastore.Sync(prefix)
}

/**
GetValue其实可以支持返回多条记录和全文检索结果
一般Key的格式是/peerEndpoint/12D3KooWG59NPEuY1dseFzXMSyYbHQb1pfpPiMq5fk7c48exxNJp
如果需要支持条件查询，第二个/后的格式就不是这样的，可以用=表示条件，类似url，甚至类似elastic的查询条件
*/
func (this *DispatchDatastore) Get(key datastore.Key) (value []byte, err error) {
	request, err := NewKeyRequest(key)
	if err != nil {
		return nil, err
	}
	return request.Datastore.Get(key)
}

// Has implements Datastore.Has
func (this *DispatchDatastore) Has(key datastore.Key) (exists bool, err error) {
	request, err := NewKeyRequest(key)
	if err != nil {
		return false, err
	}
	return request.Datastore.Has(key)
}

// GetSize implements Datastore.GetSize
func (this *DispatchDatastore) GetSize(key datastore.Key) (size int, err error) {
	request, err := NewKeyRequest(key)
	if err != nil {
		return 0, err
	}
	return request.Datastore.GetSize(key)
}

// Delete implements Datastore.Delete
func (this *DispatchDatastore) Delete(key datastore.Key) (err error) {
	request, err := NewKeyRequest(key)
	if err != nil {
		return err
	}
	return request.Datastore.Delete(key)
}

// Query implements Datastore.Query
func (this *DispatchDatastore) Query(q dsq.Query) (dsq.Results, error) {
	logger.Sugar.Warnf("query trigger:%v:%v", q.Prefix, q.String())
	request, err := NewPrefixRequest(q.Prefix)
	if err != nil {
		return nil, err
	}
	return request.Datastore.Query(q)
}

func (this *DispatchDatastore) Batch() (datastore.Batch, error) {
	return datastore.NewBasicBatch(this), nil
}

func (this *DispatchDatastore) Close() error {
	return nil
}
