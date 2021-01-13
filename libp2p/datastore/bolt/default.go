package bolt

import (
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-ds-bolt"
)

type Datastore struct {
	boltds.BoltDatastore
}

func (d Datastore) Sync(prefix datastore.Key) error {
	panic("implement me")
}

func NewDatastore(path string, bucket string, noSync bool) (*Datastore, error) {
	ds, err := boltds.NewBoltDatastore(path, bucket, noSync)
	if err != nil {
		return nil, err
	}
	datastore := Datastore{BoltDatastore: *ds}

	return &datastore, nil
}
