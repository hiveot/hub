package pebble

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/buckets"
	"log/slog"
	"os"

	"github.com/cockroachdb/pebble"
	"golang.org/x/sys/unix"
)

// PebbleStore implements the IBucketStore API using the embedded CockroachDB pebble database
//
// The following benchmark are made using BucketBench_test.go
// Performance is stellar! Fast, efficient data storage and low memory usage compared to the others.
// Estimates are made using a i5-4570S @2.90GHz cpu. Document size is 100 bytes.
//
// Create&commit bucket, no data changes  (fast since pebbles doesn't use transactions for this)
//
//	Dataset 1K,        0.1 us/op
//	Dataset 10K,       0.1 us/op
//	Dataset 100K       0.1 us/op
//	Dataset 1M         0.1 us/op
//
// Get bucket 1 record
//
//	Dataset 1K,        1.0 us/op
//	Dataset 10K,       1.6 us/op
//	Dataset 100K       1.6 us/op
//	Dataset 1M         3.2 us/op
//
// Set bucket 1 record
//
//	Dataset 1K,         2.2 us/op
//	Dataset 10K,        2.2 us/op
//	Dataset 100K        2.5 us/op
//	Dataset 1M          3.0 us/op
//	Dataset 10M        40   us/op
//
// Seek, 1 record
//
//	Dataset 1K,         5 us/op
//	Dataset 10K,        3 us/op
//	Dataset 100K        3 us/op
//	Dataset 1M         14 us/op
//	Dataset 10M       144 us/op
//
// See https://pkg.go.dev/github.com/cockroachdb/pebble for Pebble's documentation.
type PebbleStore struct {
	storeDirectory string
	db             *pebble.DB
}

func (store *PebbleStore) Close() error {
	err := store.db.Close()
	return err
}

// GetBucket returns a bucket with the given ID.
// If the bucket doesn't yet exist it will be created.
func (store *PebbleStore) GetBucket(bucketID string) (bucket buckets.IBucket) {
	pb := NewPebbleBucket(bucketID, store.db)
	return pb
}

// Open the store
func (store *PebbleStore) Open() (err error) {
	options := &pebble.Options{}
	// pebble.Open will panic if the store directory is readonly, so check ahead to return an error
	stat, err := os.Stat(store.storeDirectory)
	// if the path exists, it must be a directory
	if err == nil {
		if !stat.IsDir() {
			err = fmt.Errorf("can't open store. '%s' is not a directory", store.storeDirectory)
		}
	} else if errors.Is(err, os.ErrNotExist) {
		// if the path doesn't exist, create a directory with mode 0700
		err = os.MkdirAll(store.storeDirectory, 0700)
	}
	// path must be writable to avoid a panic
	if err == nil {
		err = unix.Access(store.storeDirectory, unix.W_OK)
	}
	if err == nil {
		store.db, err = pebble.Open(store.storeDirectory, options)
	} else {
		slog.Error("failed to open bucket store", "directory", store.storeDirectory, "err", err)
	}
	return err
}

// NewPebbleStore creates a storage database with bucket support.
//
//	storeDirectory is the directory (not file) holding the database
func NewPebbleStore(storeDirectory string) *PebbleStore {
	srv := &PebbleStore{
		storeDirectory: storeDirectory,
	}
	return srv
}
