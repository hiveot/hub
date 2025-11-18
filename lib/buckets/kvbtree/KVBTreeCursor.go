package kvbtree

import (
	"github.com/hiveot/hivehub/lib/buckets"
	"github.com/tidwall/btree"
)

type KVBTreeCursor struct {
	bucket buckets.IBucket
	kviter btree.MapIter[string, []byte]
}

func (cursor *KVBTreeCursor) BucketID() string {
	return cursor.bucket.ID()
}

// First moves the cursor to the first item
func (cursor *KVBTreeCursor) First() (key string, value []byte, valid bool) {
	valid = cursor.kviter.First()
	if !valid {
		return
	}
	key = cursor.kviter.Key()
	value = cursor.kviter.Value()
	return
}

// Last moves the cursor to the last item
func (cursor *KVBTreeCursor) Last() (key string, value []byte, valid bool) {
	valid = cursor.kviter.Last()
	if !valid {
		return
	}
	key = cursor.kviter.Key()
	value = cursor.kviter.Value()
	return
}

// Next increases the cursor position and return the next key and value
// If the end is reached the returned key is empty
func (cursor *KVBTreeCursor) Next() (key string, value []byte, valid bool) {
	valid = cursor.kviter.Next()
	if !valid {
		return
	}
	key = cursor.kviter.Key()
	value = cursor.kviter.Value()
	return
}

// NextN increases the cursor position N times and return the encountered key-value pairs
func (cursor *KVBTreeCursor) NextN(steps uint) (docs map[string][]byte, itemsRemaining bool) {
	docs = make(map[string][]byte)
	for i := uint(0); i < steps; i++ {
		itemsRemaining = cursor.kviter.Next()
		if !itemsRemaining {
			break
		}
		key := cursor.kviter.Key()
		value := cursor.kviter.Value()
		docs[key] = value
	}
	return
}

// Prev decreases the cursor position and return the previous key and value
// If the head is reached the returned key is empty
func (cursor *KVBTreeCursor) Prev() (key string, value []byte, valid bool) {
	valid = cursor.kviter.Prev()
	if !valid {
		return
	}
	key = cursor.kviter.Key()
	value = cursor.kviter.Value()
	return
}

// PrevN decreases the cursor position N times and return the encountered key-value pairs
func (cursor *KVBTreeCursor) PrevN(steps uint) (docs map[string][]byte, itemsRemaining bool) {
	docs = make(map[string][]byte)
	for i := uint(0); i < steps; i++ {
		itemsRemaining = cursor.kviter.Prev()
		if !itemsRemaining {
			break
		}
		key := cursor.kviter.Key()
		value := cursor.kviter.Value()
		docs[key] = value
	}
	return
}

// Release the cursor capability
func (cursor *KVBTreeCursor) Release() {
}

// Seek positions the cursor at the given searchKey.
// This implementation is brute force. It generates a sorted list of orderedKeys for use by the cursor.
// This should still be fast enough for most cases. (test shows around 500msec for 1 million orderedKeys).
//
//	BucketID to seach for. Returns and error if the bucket is not found
//	key is the starting point. If key doesn't exist, the next closest key will be used.
//
// This returns a cursor with Next() and Prev() iterators
func (cursor *KVBTreeCursor) Seek(searchKey string) (key string, value []byte, valid bool) {
	//var err error
	valid = cursor.kviter.Seek(searchKey)
	if !valid {
		return
	}
	key = cursor.kviter.Key()
	value = cursor.kviter.Value()
	return
}

// NewKVCursor create a new bucket cursor for the KV store.
// Cursor.Remove() must be called to release the resources.
//
//	bucket is the bucket holding the data
//	orderedKeys is a snapshot of the keys in ascending order
//
// func NewKVCursor(bucket bucketstore.IBucket, orderedKeys []string, kvbtree btree.Map[string, []byte]) *KVBTreeCursor {
func NewKVCursor(bucket buckets.IBucket, kvIter btree.MapIter[string, []byte]) *KVBTreeCursor {
	cursor := &KVBTreeCursor{
		bucket: bucket,
		kviter: kvIter,
	}
	return cursor
}
