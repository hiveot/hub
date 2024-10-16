package bolts

import (
	"fmt"
	"github.com/hiveot/hub/lib/buckets"
	"log/slog"
	"os"
	"path"
	"sync/atomic"
	"time"

	"go.etcd.io/bbolt"
)

// BoltStore implements the IBucketStore API using the embedded bolt database
// This uses the BBolt package which is a derivative of BoltDB
// Estimates using a i5-4570S @2.90GHz cpu.
//
// Create & close bbBucket
//   Dataset 1K,        0.8 us/op
//   Dataset 10K,       0.8 us/op
//   Dataset 100K       0.8 us/op
//   Dataset 1M         0.7 us/op
//
// Bucket Get 1 record
//   Dataset 1K,        1.3 us/op
//   Dataset 10K,       1.3 us/op
//   Dataset 100K       1.4 us/op
//   Dataset 1M         1.3 us/op
//
// Bucket Set 1 record
//   Dataset 1K,          5.1 ms/op
//   Dataset 10K,         5.1 ms/op
//   Dataset 100K        12   ms/op
//   Dataset 1M          47   ms/op
//   Dataset 10M         62   ms/op
//
// Seek
//   Dataset 1K,        1.6 us/op
//   Dataset 10K,       1.8 us/op
//   Dataset 100K       2.0 us/op
//   Dataset 1M         1.7 us/op
//

type BoltStore struct {
	// the underlying database
	boltDB *bbolt.DB
	// storePath with the location of the database
	storePath string
	// for preventing deadlocks when closing the store. panic instead
	bucketRefCount int32
}

// Close the store and flush changes to disk
// Since boltDB locks transactions on close, this runs in the background.
// Close() returns before closing is completed.
func (store *BoltStore) Close() (err error) {
	br := atomic.LoadInt32(&store.bucketRefCount)
	slog.Info("closing store", "storePath", store.storePath, "refCnt", br)
	//close with wait until all transactions are completed ...
	// so it might hang forever if not all transactions are released.
	//err = store.boltDB.Remove()
	err2 := make(chan error, 1)
	go func() {
		err2 <- store.boltDB.Close()
	}()
	select {
	case <-time.After(10 * time.Second):
		panic("BoltDB is not closing")
	case err = <-err2:
		return err
	}
	return err
}

// GetBucket returns a bbBucket to use for writing to storage.
// This does not yet create the bbBucket in the database until an operation takes place on the bbBucket.
func (store *BoltStore) GetBucket(bucketID string) (bucket buckets.IBucket) {

	//logrus.Infof("Opening BoltBucket '%s", bucketID, store.clientID)
	bucket = NewBoltBucket(bucketID, store.boltDB, store.onBucketReleased)

	atomic.AddInt32(&store.bucketRefCount, 1)
	return bucket
}

// track bbBucket references
func (store *BoltStore) onBucketReleased(bucket buckets.IBucket) {
	atomic.AddInt32(&store.bucketRefCount, -1)
}

// Open the store
func (store *BoltStore) Open() (err error) {
	slog.Info("Opening BoltDB store", "storePath", store.storePath)

	// make sure the folder exists
	storeDir := path.Dir(store.storePath)
	err = os.MkdirAll(storeDir, 0700)
	if err != nil {
		slog.Error("Failed ensuring folder exists", "err", err)
	}

	options := &bbolt.Options{
		Timeout:        10,                    // wait max 1 sec for a file lock
		NoFreelistSync: false,                 // consider true for increased write performance
		FreelistType:   bbolt.FreelistMapType, // performant even for large DB
		//InitialMmapSize: 0,  // when is this useful to set?
	}
	store.boltDB, err = bbolt.Open(store.storePath, 0600, options)

	if err != nil {
		err = fmt.Errorf("error opening BoltDB at %s: %w", store.storePath, err)
	}
	return err
}

// NewBoltStore creates a bbBucket store supporting the IBucketStore API using the embedded BBolt database
//
//	storePath is the file holding the database
func NewBoltStore(storePath string) *BoltStore {
	srv := &BoltStore{
		storePath: storePath,
	}
	return srv
}
