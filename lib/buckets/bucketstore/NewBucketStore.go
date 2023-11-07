package bucketstore

import (
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/buckets/bolts"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/buckets/pebble"
	"log/slog"
	"path"
)

// NewBucketStore creates a new bucket store of a given type
// The store will be created in the given directory using the
// backend as the name. The directory is typically the name of the service that
// uses the store. Different databases can co-exist.
//
//	directory is the directory in which to create the store
//	name of the store database file or folder without extension
//	backend is the type of store to create: BackendKVBTree, BackendBBolt, BackendPebble
func NewBucketStore(directory, name string, backend string) (store buckets.IBucketStore) {
	if backend == buckets.BackendKVBTree {
		// kvbtree stores data into a single file
		storePath := path.Join(directory, name+".kvbtree")
		store = kvbtree.NewKVStore(storePath)
	} else if backend == buckets.BackendBBolt {
		// bbolt stores data into a single file
		storePath := path.Join(directory, name+".boltdb")
		store = bolts.NewBoltStore(storePath)
	} else if backend == buckets.BackendPebble {
		// Pebbles stores data into a directory
		storePath := path.Join(directory, name+".pebble")
		store = pebble.NewPebbleStore(storePath)
	} else {
		slog.Error("Unknown backend", "backend", backend)
	}
	return store
}
