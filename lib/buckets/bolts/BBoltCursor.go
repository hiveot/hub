package bolts

import (
	"go.etcd.io/bbolt"
	"log/slog"
)

// BBoltCursor is a wrapper around the bbolt cursor to map to the IBucketCursor API and to
// ensure its transaction is released after the cursor is no longer used.
// This implements the IBucketCursor API
type BBoltCursor struct {
	bucket *bbolt.Bucket
	cursor *bbolt.Cursor
}

// First moves the cursor to the first item
func (bbc *BBoltCursor) First() (key string, value []byte, valid bool) {
	if bbc.cursor == nil {
		return "", nil, false
	}
	k, v := bbc.cursor.First()
	valid = k != nil
	return string(k), v, valid
}

// Last moves the cursor to the last item
func (bbc *BBoltCursor) Last() (key string, value []byte, valid bool) {
	if bbc.cursor == nil {
		return "", nil, false
	}
	k, v := bbc.cursor.Last()
	valid = k != nil
	return string(k), v, valid
}

// Next iterates to the next key from the current cursor
func (bbc *BBoltCursor) Next() (key string, value []byte, valid bool) {
	if bbc.cursor == nil {
		return "", nil, false
	}
	k, v := bbc.cursor.Next()
	valid = k != nil
	return string(k), v, valid
}

// NextN increases the cursor position N times and return the encountered key-value pairs
func (bbc *BBoltCursor) NextN(steps uint) (docs map[string][]byte, itemsRemaining bool) {
	docs = make(map[string][]byte)
	if bbc.cursor == nil {
		return nil, false
	}
	itemsRemaining = true
	for i := uint(0); i < steps; i++ {
		key, value := bbc.cursor.Next()
		if key == nil {
			itemsRemaining = false
			break
		}
		docs[string(key)] = value
	}
	return
}

// Prev iterations to the previous key from the current cursor
func (bbc *BBoltCursor) Prev() (key string, value []byte, valid bool) {
	if bbc.cursor == nil {
		return "", nil, false
	}
	k, v := bbc.cursor.Prev()
	valid = k != nil
	return string(k), v, valid
}

// PrevN decreases the cursor position N times and return the encountered key-value pairs
func (bbc *BBoltCursor) PrevN(steps uint) (docs map[string][]byte, itemsRemaining bool) {
	docs = make(map[string][]byte)
	if bbc.cursor == nil {
		return nil, false
	}
	itemsRemaining = true

	for i := uint(0); i < steps; i++ {
		key, value := bbc.cursor.Prev()
		if key == nil {
			itemsRemaining = false
			break
		}
		docs[string(key)] = value
	}
	return
}

// Release the cursor
// This ends the bbolt bucket transaction
func (bbc *BBoltCursor) Release() {
	slog.Info("releasing bucket cursor")
	if bbc.bucket != nil {
		bbc.bucket.Tx().Rollback()
		bbc.cursor = nil
		bbc.bucket = nil
	}
}

// Seek returns a cursor with Next() and Prev() iterators
func (bbc *BBoltCursor) Seek(searchKey string) (key string, value []byte, valid bool) {
	if bbc.cursor == nil {
		return "", nil, false
	}
	k, v := bbc.cursor.Seek([]byte(searchKey))
	valid = k != nil
	return string(k), v, valid
}

func NewBBoltCursor(bucket *bbolt.Bucket) *BBoltCursor {
	var bbCursor *bbolt.Cursor = nil
	if bucket != nil {
		bbCursor = bucket.Cursor()
	}
	bbc := &BBoltCursor{
		bucket: bucket,
		cursor: bbCursor,
	}

	return bbc
}
