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
	closed     bool
}

// Close the bucket
func (bucket *PebbleBucket) Close() (err error) {
	if bucket.closed {
		err = fmt.Errorf("bucket '%s' is already closed", bucket.bucketID)
	}
	bucket.closed = true
	return err
}

// Commit changes to the bucket
//func (bucket *PebbleBucket) Commit() (err error) {
//	// this is just for error detection
//	if !bucket.writable {
//		err = fmt.Errorf("cant commit as bucket '%s' is not writable",bucket.bucketID)
//	}
//	if bucket.closed {
//		err = fmt.Errorf("bucket '%s' is already closed", bucket.bucketID)
//	}
//	return err
//}

// Cursor provides an iterator for the bucket using a pebble iterator with prefix bounds
//
//	optional name for use by application
func (bucket *PebbleBucket) Cursor() (buckets.IBucketCursor, error) {
	// bucket prefix is {bucketID}$
	// range bounds end at {bucketID}@
	opts := &pebble.IterOptions{
		LowerBound: []byte(bucket.bucketID + "$"),
		// a bucketID that is longer would be included when using @. Is this a bug?
		//UpperBound: []byte(bucket.bucketID + "@"), // this key never exists
		// FIXME: add testcase
		UpperBound:      []byte(bucket.bucketID + "%"), // this key never exists
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
		return nil, fmt.Errorf("Error getting cursor: %w", err)
	}
	cursor := NewPebbleCursor(bucket.bucketID, bucket.rangeStart, bucketIterator)
	return cursor, nil
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
		// return error if not found
		err = fmt.Errorf("key '%s' not found", key)
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
		err := fmt.Errorf("empty key '%s' for bucket '%s'",
			key, bucket.bucketID)
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
			slog.Error("failed set multiple documents", "err", err)
			_ = batch.Close()
			return err
		}
	}
	err = bucket.db.Apply(batch, &pebble.WriteOptions{})
	_ = batch.Close()
	return err
}

// NewPebbleBucket creates a new bucket
//
//	bucketID identifies the bucket
//	pebbleDB backend storage
func NewPebbleBucket(bucketID string, pebbleDB *pebble.DB) *PebbleBucket {
	if pebbleDB == nil {
		slog.Error("pebbleDB is nil", "bucketID", bucketID)
	}
	srv := &PebbleBucket{
		bucketID:   bucketID,
		db:         pebbleDB,
		rangeStart: bucketID + "$",
		rangeEnd:   bucketID + "@",
	}
	return srv
}
