package elastic

import (
	"context"
	"errors"
	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
)

/**
dht datastore的数据库实现和elasticsearch实现
*/

// ElasticDatastore stores nothing, but conforms to the API.
// Useful to test with.
type ElasticDatastore struct {
}

// NewElasticDatastore constructs a null datastoe
func NewElasticDatastore() *ElasticDatastore {
	return &ElasticDatastore{}
}

// Put implements Datastore.Put
func (d *ElasticDatastore) Put(ctx context.Context, key datastore.Key, value []byte) (err error) {
	return nil
}

// Sync implements Datastore.Sync
func (d *ElasticDatastore) Sync(ctx context.Context, prefix datastore.Key) error {
	return nil
}

// Get implements Datastore.Get
func (d *ElasticDatastore) Get(ctx context.Context, key datastore.Key) (value []byte, err error) {
	return nil, errors.New("ErrNotFound")
}

// Has implements Datastore.Has
func (d *ElasticDatastore) Has(ctx context.Context, key datastore.Key) (exists bool, err error) {
	return false, nil
}

// Has implements Datastore.GetSize
func (d *ElasticDatastore) GetSize(ctx context.Context, key datastore.Key) (size int, err error) {
	return -1, errors.New("ErrNotFound")
}

// Delete implements Datastore.Delete
func (d *ElasticDatastore) Delete(ctx context.Context, key datastore.Key) (err error) {
	return nil
}

// Query implements Datastore.Query
func (d *ElasticDatastore) Query(ctx context.Context, q dsq.Query) (dsq.Results, error) {
	return dsq.ResultsWithEntries(q, nil), nil
}

func (d *ElasticDatastore) Batch(ctx context.Context) (datastore.Batch, error) {
	return datastore.NewBasicBatch(d), nil
}

func (d *ElasticDatastore) Close() error {
	return nil
}
