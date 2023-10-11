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
//
//		directory is the directory to create the store
//	 clientID is used to name the store
//		backend is the type of store to create: BackendKVBTree, BackendBBolt, BackendPebble
func NewBucketStore(directory, clientID, backend string) (store buckets.IBucketStore) {
	if backend == buckets.BackendKVBTree {
		// kvbtree stores data into a single file
		storePath := path.Join(directory, clientID+".json")
		store = kvbtree.NewKVStore(clientID, storePath)
	} else if backend == buckets.BackendBBolt {
		// bbolt stores data into a single file
		storePath := path.Join(directory, clientID+".boltdb")
		store = bolts.NewBoltStore(clientID, storePath)
	} else if backend == buckets.BackendPebble {
		// Pebbles stores data into a directory
		storePath := path.Join(directory, clientID)
		store = pebble.NewPebbleStore(clientID, storePath)
	} else {
		slog.Error("Unknown backend", "backend", backend)
	}
	return store
}
