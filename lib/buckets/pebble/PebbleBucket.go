package pebble

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/buckets"
	"log/slog"

	"github.com/cockroachdb/pebble"
)

// PebbleBucket represents a transactional bucket using Pebble
// Buckets are not supported in Pebble so these are simulated by prefixing all keys with "{bucketID}$"
// Each write operation is its own transaction.
type PebbleBucket struct {
	db *pebble.DB
	// key range for this bucket
	rangeStart string
	rangeEnd   string
	bucketID   string
	clientID   string
	closed     bool
}

// Close the bucket
func (bucket *PebbleBucket) Close() (err error) {
	if bucket.closed {
		err = fmt.Errorf("bucket '%s' of client '%s' is already closed", bucket.bucketID, bucket.clientID)
	}
	bucket.closed = true
	return err
}

// Commit changes to the bucket
//func (bucket *PebbleBucket) Commit() (err error) {
//	// this is just for error detection
//	if !bucket.writable {
//		err = fmt.Errorf("cant commit as bucket '%s' of client '%s' is not writable",
//			bucket.bucketID, bucket.clientID)
//	}
//	if bucket.closed {
//		err = fmt.Errorf("bucket '%s' of client '%s' is already closed", bucket.bucketID, bucket.clientID)
//	}
//	return err
//}

// Cursor provides an iterator for the bucket using a pebble iterator with prefix bounds
func (bucket *PebbleBucket) Cursor() buckets.IBucketCursor {
	// bucket prefix is {bucketID}$
	// range bounds end at {bucketID}@
	opts := &pebble.IterOptions{
		LowerBound:      []byte(bucket.bucketID + "$"),
		UpperBound:      []byte(bucket.bucketID + "@"), // this key never exists
		TableFilter:     nil,
		PointKeyFilters: nil,
		RangeKeyFilters: nil,
		KeyTypes:        0,
		RangeKeyMasking: pebble.RangeKeyMasking{
			Suffix: nil,
			Filter: nil,
		},
		OnlyReadGuaranteedDurable: false,
		UseL6Filters:              false,
	}
	bucketIterator, err := bucket.db.NewIter(opts)
	if err != nil {
		slog.Error("Error getting cursor", "err", err)
	}
	cursor := NewPebbleCursor(bucket.clientID, bucket.bucketID, bucket.rangeStart, bucketIterator)
	return cursor
}

// Delete removes the key-value pair from the bucket store
func (bucket *PebbleBucket) Delete(key string) (err error) {
	bucketKey := bucket.rangeStart + key
	opts := &pebble.WriteOptions{}
	err = bucket.db.Delete([]byte(bucketKey), opts)
	return err
}

// Get returns the document for the given key
func (bucket *PebbleBucket) Get(key string) (doc []byte, err error) {
	bucketKey := bucket.rangeStart + key
	if bucket.closed || bucket.db == nil {
		slog.Error("Bad state getting key. Bucket closed or DB nil.", "key", key, "isClosed", bucket.closed)
	}
	byteValue, closer, err := bucket.db.Get([]byte(bucketKey))
	if err == nil {
		doc = bytes.NewBuffer(byteValue).Bytes()
		err = closer.Close()
	} else if errors.Is(err, pebble.ErrNotFound) {
		// return doc nil if not found
		err = nil
		doc = nil
	}
	return doc, err
}

// GetMultiple returns a batch of documents with existing keys
func (bucket *PebbleBucket) GetMultiple(keys []string) (docs map[string][]byte, err error) {

	docs = make(map[string][]byte)
	batch := bucket.db.NewIndexedBatch()
	for _, key := range keys {
		bucketKey := bucket.rangeStart + key
		value, closer, err2 := batch.Get([]byte(bucketKey))
		if err2 == nil {
			docs[key] = bytes.NewBuffer(value).Bytes()
			err = closer.Close()
		}
	}
	err = batch.Close()
	return docs, err
}

// ID returns the bucket ID
func (bucket *PebbleBucket) ID() string {
	return bucket.bucketID
}

// Info returns bucket information
// FIXME: Unable to determine the number of records in a bucket (or even in the DB)
func (bucket *PebbleBucket) Info() (info *buckets.BucketStoreInfo) {

	//metrics := bucket.db.Metrics()
	// bucket key range
	//size, _ := bucket.db.EstimateDiskUsage([]byte(bucket.rangeStart), []byte(bucket.rangeEnd))
	//size = uint64(0)
	//sstables, err := bucket.db.SSTables()
	//if err == nil {
	//	for _, tblList := range sstables {
	//		for _, tbl := range tblList {
	//			size += tbl.Size
	//		}
	//	}
	//}

	info = &buckets.BucketStoreInfo{
		Id:     bucket.bucketID,
		Engine: buckets.BackendPebble,
		// FIXME: get bucket metrics
		DataSize:  -1, //int64(metrics.WAL.Size),
		NrRecords: -1,
		//Size: size,
	}

	return info
}

// Set sets a document with the given key
func (bucket *PebbleBucket) Set(key string, doc []byte) error {
	if key == "" {
		err := fmt.Errorf("empty key '%s' for bucket '%s' and client '%s'",
			key, bucket.bucketID, bucket.clientID)
		return err
	}
	bucketKey := bucket.rangeStart + key
	opts := &pebble.WriteOptions{}
	err := bucket.db.Set([]byte(bucketKey), doc, opts)
	return err

}

// SetMultiple sets multiple documents in a batch update
func (bucket *PebbleBucket) SetMultiple(docs map[string][]byte) (err error) {

	batch := bucket.db.NewBatch()
	for key, value := range docs {
		bucketKey := bucket.rangeStart + key
		opts := &pebble.WriteOptions{}
		err = batch.Set([]byte(bucketKey), value, opts)
		if err != nil {
			slog.Error("failed set multiple documents for client", "clientID", bucket.clientID, "err", err)
			_ = batch.Close()
			return err
		}
	}
	err = bucket.db.Apply(batch, &pebble.WriteOptions{})
	_ = batch.Close()
	return err
}

// NewPebbleBucket creates a new bucket
func NewPebbleBucket(clientID, bucketID string, pebbleDB *pebble.DB) *PebbleBucket {
	if pebbleDB == nil {
		slog.Error("pebbleDB is nil", "clientID", clientID, "bucketID", bucketID)
	}
	srv := &PebbleBucket{
		clientID:   clientID,
		bucketID:   bucketID,
		db:         pebbleDB,
		rangeStart: bucketID + "$",
		rangeEnd:   bucketID + "@",
	}
	return srv
}
