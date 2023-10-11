package mongohs

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type MongoCursor struct {
	//bucket      bucketstore.IBucket
	bucketID    string
	mongoCursor *mongo.Cursor
}

// First moves the cursor to the first item
func (cursor *MongoCursor) First() (key string, value []byte, valid bool) {
	//valid = cursor.mongoCursor.First()
	//if !valid {
	//	return
	//}
	//key = cursor.mongoCursor.Key()
	//value = cursor.mongoCursor.Value()
	return
}

// Last moves the cursor to the last item
func (cursor *MongoCursor) Last() (key string, value []byte, valid bool) {
	//valid = cursor.mongoCursor.Last()
	//if !valid {
	//	return
	//}
	//key = cursor.mongoCursor.Key()
	//value = cursor.mongoCursor.Value()
	return
}

// Next increases the cursor position and return the next key and value
// If the end is reached the returned key is empty
func (cursor *MongoCursor) Next() (key string, value []byte, valid bool) {
	//ctx := context.Background()
	//valid = cursor.mongoCursor.Next(ctx)
	//
	//err := cursor.mongoCursor.Decode(&value)
	//if err != nil {
	//	return "", nil, false
	//}
	//valid = true
	//key = cursor.mongoCursor.Key()
	return
}

// NextN increases the cursor position N times and return the encountered key-value pairs
func (cursor *MongoCursor) NextN(steps uint) (docs map[string][]byte, itemsRemaining bool) {
	//docs = make(map[string][]byte)
	//for i := uint(0); i < steps; i++ {
	//	itemsRemaining = cursor.mongoCursor.Next()
	//	if !itemsRemaining {
	//		break
	//	}
	//	key := cursor.mongoCursor.Key()
	//	value := cursor.mongoCursor.Value()
	//	docs[key] = value
	//}
	return
}

// Prev decreases the cursor position and return the previous key and value
// If the head is reached the returned key is empty
func (cursor *MongoCursor) Prev() (key string, value []byte, valid bool) {
	//valid = cursor.mongoCursor.Prev()
	//if !valid {
	//	return
	//}
	//key = cursor.mongoCursor.Key()
	//value = cursor.mongoCursor.Value()
	return
}

// PrevN decreases the cursor position N times and return the encountered key-value pairs
func (cursor *MongoCursor) PrevN(steps uint) (docs map[string][]byte, itemsRemaining bool) {
	//docs = make(map[string][]byte)
	//for i := uint(0); i < steps; i++ {
	//	itemsRemaining = cursor.mongoCursor.Prev()
	//	if !itemsRemaining {
	//		break
	//	}
	//	key := cursor.mongoCursor.Key()
	//	value := cursor.mongoCursor.Value()
	//	docs[key] = value
	//}
	return
}

// Release the cursor capability
func (cursor *MongoCursor) Release() {
	ctx := context.Background()
	cursor.mongoCursor.Close(ctx)
}

// Seek positions the cursor at the given searchKey.
// This implementation is brute force. It generates a sorted list of orderedKeys for use by the cursor.
// This should still be fast enough for most cases. (test shows around 500msec for 1 million orderedKeys).
//
//	BucketID to seach for. Returns and error if the bucket is not found
//	key is the starting point. If key doesn't exist, the next closest key will be used.
//
// This returns a cursor with Next() and Prev() iterators
func (cursor *MongoCursor) Seek(searchKey string) (key string, value []byte, valid bool) {
	// tbd use FindOne
	//var err error
	//valid = cursor.mongoCursor.Find(searchKey)
	//if !valid {
	//	return
	//}
	//key = cursor.mongoCursor.Key()
	//value = cursor.mongoCursor.Value()
	return
}

// NewMongoCursor create a new bucket cursor for the mongo client.
// Cursor.Close() must be called to release the resources.
//
// func NewKVCursor(bucket bucketstore.IBucket, orderedKeys []string, kvbtree btree.Map[string, []byte]) *MongoCursor {
func NewMongoCursor(bucketID string, mongoCursor *mongo.Cursor) *MongoCursor {
	cursor := &MongoCursor{
		bucketID:    bucketID,
		mongoCursor: mongoCursor,
	}
	return cursor
}
