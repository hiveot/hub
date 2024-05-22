package pebble

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/cockroachdb/pebble"
)

type PebbleCursor struct {
	//db       *pebble.DB
	//bucket   *PebbleBucket
	bucketPrefix string // prefix to remove from keys returned by get/set/seek/first/last
	bucketID     string
	iterator     *pebble.Iterator
}

func (cursor *PebbleCursor) BucketID() string {
	return cursor.bucketID
}

// First moves the cursor to the first item
func (cursor *PebbleCursor) First() (key string, value []byte, valid bool) {
	valid = cursor.iterator.First()
	if !valid {
		return
	}
	return cursor.getKV()
}

// Return the iterator current key and value
// This removes the bucket prefix
func (cursor *PebbleCursor) getKV() (key string, value []byte, valid bool) {
	k := string(cursor.iterator.Key())
	v, err := cursor.iterator.ValueAndErr()
	if strings.HasPrefix(k, cursor.bucketPrefix) {
		key = k[len(cursor.bucketPrefix):]
		valid = cursor.iterator.Valid()
	} else {
		err = fmt.Errorf("bucket key '%s' has no prefix '%s'", k, cursor.bucketPrefix)
		valid = false
	}

	// what to do in case of error?
	_ = err
	return key, v, valid
}

// Last moves the cursor to the last item
func (cursor *PebbleCursor) Last() (key string, value []byte, valid bool) {
	valid = cursor.iterator.Last()
	if !valid {
		return
	}
	return cursor.getKV()
}

// Next iterates to the next key from the current cursor
func (cursor *PebbleCursor) Next() (key string, value []byte, valid bool) {
	valid = cursor.iterator.Next()
	if !valid {
		return
	}
	return cursor.getKV()
}

// NextN increases the cursor position N times and return the encountered key-value pairs
func (cursor *PebbleCursor) NextN(steps uint) (docs map[string][]byte, itemsRemaining bool) {
	docs = make(map[string][]byte)
	for i := uint(0); i < steps; i++ {
		itemsRemaining = cursor.iterator.Next()
		if !itemsRemaining {
			break
		}
		key, value, _ := cursor.getKV()
		docs[key] = value
	}
	return
}

// Prev iterations to the previous key from the current cursor
func (cursor *PebbleCursor) Prev() (key string, value []byte, valid bool) {
	valid = cursor.iterator.Prev()
	if !valid {
		return
	}
	return cursor.getKV()
}

// PrevN decreases the cursor position N times and return the encountered key-value pairs
func (cursor *PebbleCursor) PrevN(steps uint) (docs map[string][]byte, itemsRemaining bool) {
	docs = make(map[string][]byte)
	for i := uint(0); i < steps; i++ {
		itemsRemaining = cursor.iterator.Prev()
		if !itemsRemaining {
			break
		}
		key, value, _ := cursor.getKV()
		docs[key] = value
	}
	return
}

// Release the cursor
func (cursor *PebbleCursor) Release() {
	err := cursor.iterator.Close()
	if err != nil {
		slog.Error("unexpected error releasing cursor", "err", err.Error())
	}
}

// Seek returns a cursor with Next() and Prev() iterators
func (cursor *PebbleCursor) Seek(searchKey string) (key string, value []byte, valid bool) {
	bucketKey := cursor.bucketPrefix + searchKey
	valid = cursor.iterator.SeekGE([]byte(bucketKey))
	if !valid {
		return
	}
	return cursor.getKV()
}

// NewPebbleCursor create a pebble storage iterator cursor for a pebble bucket
//
//	bucketID the cursor iterates
//	bucketPrefix
//	iterator is pebble's iterator
func NewPebbleCursor(bucketID string, bucketPrefix string, iterator *pebble.Iterator) *PebbleCursor {

	// TBD: use pebble range keys instead of bucket prefix
	cursor := &PebbleCursor{
		bucketPrefix: bucketPrefix,
		bucketID:     bucketID,
		iterator:     iterator,
	}
	return cursor
}
