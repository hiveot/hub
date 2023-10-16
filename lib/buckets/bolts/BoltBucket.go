package bolts

import (
	"bytes"
	"fmt"
	"github.com/hiveot/hub/lib/buckets"
	"log/slog"

	"go.etcd.io/bbolt"
)

// BoltBucket implements the IBucket API using the embedded bolt database
// This bbBucket differs from bolt buckets in that transactions are created for read/write operations.
// This means that bbolt buckets are created and released continuously as needed.
type BoltBucket struct {
	// the underlying DB
	db *bbolt.DB
	// client this bbBucket is for. Intended for debugging and logging.
	clientID string
	// ID of the storage bbBucket
	bucketID string
	// callback for reporting bbBucket is released
	onRelease func(bucket buckets.IBucket)
}

// bucketTransaction creates a transaction with a bbBucket and invokes a callback
// If the transaction is writable and successful it will be committed, otherwise rolled back.
// Note: bbolt warns that opening a read transaction and write transaction in the same goroutine causes deadlock.
func (bb *BoltBucket) bucketTransaction(writable bool, cb func(bucket *bbolt.Bucket) error) error {
	var bboltBucket *bbolt.Bucket
	tx, err := bb.db.Begin(writable)
	slog.Debug("starting transaction for bbBucket. ", "bucketID", bb.bucketID, "writable", writable)
	if err != nil {
		slog.Error("unable to create transaction for bbBucket for client",
			"bucketID", bb.bucketID, "clientID", bb.clientID, "err", err.Error())
		return err
	}
	if writable {
		bboltBucket, err = tx.CreateBucketIfNotExists([]byte(bb.bucketID))
	} else {
		bboltBucket = tx.Bucket([]byte(bb.bucketID))
	}
	if bboltBucket == nil {
		slog.Debug("Nothing to read, bbBucket for client doesn't yet exist", "bucketID", bb.bucketID, "clientID", bb.clientID)
		// This bbBucket has never been written to so ignore the rest
	} else {
		err = cb(bboltBucket)
	}
	if writable && err == nil {
		slog.Debug("closing write transaction for bbBucket. commit", "bucketID", bb.bucketID)
		err = tx.Commit()
	} else {
		slog.Debug("closing readonly transaction for bbBucket. rollback", "bucketID", bb.bucketID)
		_ = tx.Rollback()
	}
	return err
}

// Close the bbBucket
func (bb *BoltBucket) Close() (err error) {
	//slog.Infof("Closing bbBucket '%s' of client '%s", bb.bucketID, bb.clientID)
	bb.onRelease(bb)
	return err
}

// Cursor returns a new cursor for iterating the bbBucket.
// This creates a read-only bbolt bbBucket for iteration.
// The cursor MUST be closed after use to release the bbolt bbBucket.
// Do not write to the database while iterating.
//
// This returns a cursor with Next() and Prev() iterators or an error if the bbBucket doesn't exist
func (bb *BoltBucket) Cursor() (cursor buckets.IBucketCursor) {

	tx, err := bb.db.Begin(false)
	if err != nil {
		slog.Error("unable to create transaction for bbBucket for client",
			"bucketID", bb.bucketID, "clientID", bb.clientID, "err", err)
	}
	bboltBucket := tx.Bucket([]byte(bb.bucketID))
	if bboltBucket == nil {
		// nothing to iterate, the bbBucket doesn't exist
		_ = tx.Rollback()
		err = fmt.Errorf("bbBucket '%s' no longer exist for client '%s'", bb.bucketID, bb.clientID)
		slog.Info(err.Error())
	}
	// always return a cursor, although it might be empty without a boltbucket
	// cursor MUST end the transaction when done
	cursor = NewBBoltCursor(bb.bucketID, bboltBucket)
	return cursor
}

// Delete a key in the bbBucket
func (bb *BoltBucket) Delete(key string) (err error) {

	//
	err = bb.bucketTransaction(true, func(bboltBucket *bbolt.Bucket) error {
		return bboltBucket.Delete([]byte(key))
	})

	return err
}

// Get reads a document with the given key
// returns nil if the key doesn't exist
func (bb *BoltBucket) Get(key string) (val []byte, err error) {
	var byteValue []byte
	err = bb.bucketTransaction(false, func(bboltBucket *bbolt.Bucket) error {
		v := bboltBucket.Get([]byte(key))
		if v != nil {
			byteValue = bytes.NewBuffer(v).Bytes() //copy the buffer
		}
		return nil
	})
	return byteValue, err
}

// GetMultiple returns a batch of documents with existing keys
func (bb *BoltBucket) GetMultiple(keys []string) (docs map[string][]byte, err error) {
	docs = make(map[string][]byte)

	err = bb.bucketTransaction(false, func(bboltBucket *bbolt.Bucket) error {
		for _, key := range keys {
			byteValue := bboltBucket.Get([]byte(key))
			// simply ignore non existing keys and log as info
			if byteValue == nil {
				//slog.Infof("key '%s' in bbBucket '%s' for client '%s' doesn't exist", key, bb.bucketID, bb.clientID)
			} else {
				// byteValue is only valid within the transaction
				val := bytes.NewBuffer(byteValue).Bytes()
				docs[key] = val
			}
		}
		return nil
	})
	return docs, err
}

// ID returns the bbBucket's ID
func (bb *BoltBucket) ID() string {
	return bb.bucketID
}

// Info returns the bbBucket info
func (bb *BoltBucket) Info() (info *buckets.BucketStoreInfo) {
	info = &buckets.BucketStoreInfo{}
	tx, err := bb.db.Begin(false)
	info.Id = bb.bucketID
	info.Engine = buckets.BackendBBolt
	info.DataSize = -1
	info.NrRecords = -1
	if err == nil {
		bucket := tx.Bucket([]byte(bb.bucketID))
		bucketStats := bucket.Stats()
		info.NrRecords = int64(bucketStats.KeyN)
		// not sure if this is correct
		info.DataSize = int64(bucketStats.LeafInuse)
		_ = tx.Rollback()
	}
	return
}

// Set writes a document with the given key
func (bb *BoltBucket) Set(key string, value []byte) (err error) {
	err = bb.bucketTransaction(true, func(bboltBucket *bbolt.Bucket) error {
		err = bboltBucket.Put([]byte(key), value)
		return err
	})
	return err
}

// SetMultiple writes a multiple documents in a single transaction
// This returns an error as soon as an invalid key is encountered.
// Cancel this bbBucket with Close(false) if this returns an error.
func (bb *BoltBucket) SetMultiple(docs map[string][]byte) (err error) {
	slog.Info("SetMultiple docs", "len", len(docs))
	err = bb.bucketTransaction(true, func(bboltBucket *bbolt.Bucket) error {

		for key, value := range docs {
			err = bboltBucket.Put([]byte(key), value)
			if err != nil {
				err = fmt.Errorf("error put client '%s' value for key '%s' in bbBucket '%s': %w", bb.clientID, key, bb.bucketID, err)
				//_ = bb.bbBucket.Tx().Rollback()
				return err
			}
		}
		return nil
	})
	return err
}

// NewBoltBucket creates a new bbBucket
//
//	clientID that owns the bbBucket. Used for logging
//	bucketID used to create transactional buckets
//	db bbolt database used to create transactions
//	onRelease callback to track reference for detecting unreleased buckets on close
func NewBoltBucket(clientID, bucketID string, db *bbolt.DB, onRelease func(bucket buckets.IBucket)) *BoltBucket {
	srv := &BoltBucket{
		db:        db,
		clientID:  clientID,
		bucketID:  bucketID,
		onRelease: onRelease,
	}
	return srv
}
