// Package kvbtree
package kvbtree

import (
	"bytes"
	"context"
	"fmt"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"sync"

	"github.com/tidwall/btree"
)

// KVBTreeBucket is an in-memory bucket for the KVBTreeBucket
type KVBTreeBucket struct {
	BucketID string `json:"bucketID"`
	refCount int    // simple ref count for error detection
	kvtree   btree.Map[string, []byte]

	mutex sync.RWMutex
	// cache for parsed json strings for faster query
	//queryCache map[string]interface{}

	// update handler callback to notify bucket owner
	updated func(bucket *KVBTreeBucket)
}

// Close the bucket and release its resources
// commit is not used as this store doesn't handle transactions.
// This decreases the refCount and detects an error if below 0
func (bucket *KVBTreeBucket) Close() (err error) {

	slog.Debug("closing bucket", "bucketID", bucket.BucketID)
	// this just lowers the refCount to detect leaks
	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()

	bucket.refCount--
	if bucket.refCount < 0 {
		err = fmt.Errorf("bucket '%s' closed more often than opened",
			bucket.BucketID)
	}
	return err
}

// Cursor returns a new cursor for iterating the bucket.
// The cursor MUST be closed after use to release its memory.
//
// This implementation is brute force. It generates a sorted list of key/values for use by the cursor.
// The cursor makes a shallow copy of the store. Store mutations are not reflected in the cursor.
//
// This should be fast enough for many use-cases. 100K records takes around 27msec on an i5@2.9GHz
//
// This returns a cursor with Next() and Prev() iterators
func (bucket *KVBTreeBucket) Cursor(
	ctx context.Context) (buckets.IBucketCursor, error) {

	bucket.mutex.RLock()
	defer bucket.mutex.RUnlock()

	iter := bucket.kvtree.Iter()
	cursor := NewKVCursor(ctx, bucket, iter)
	return cursor, nil
}

// Delete a document from the bucket
// Also succeeds if the document doesn't exist
func (bucket *KVBTreeBucket) Delete(key string) error {
	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()

	slog.Info("Deleting key from bucket", "key", key, "bucketID", bucket.BucketID)
	bucket.kvtree.Delete(key)
	bucket.updated(bucket)
	return nil
}

// Export returns a shallow copy of the bucket content
func (bucket *KVBTreeBucket) Export() map[string][]byte {
	bucket.mutex.RLock()
	defer bucket.mutex.RUnlock()

	// shallow copy each bucket kv pairs as well
	exportedCopy := make(map[string][]byte)
	//exportedCopy := bucket.kvtree.Copy[string, []byte]()
	iter := bucket.kvtree.Iter()
	hasItem := iter.First()
	for hasItem {
		exportedCopy[iter.Key()] = iter.Value()
		hasItem = iter.Next()
	}
	return exportedCopy
}

// Get an object by its ID
// returns an error if the key does not exist.
func (bucket *KVBTreeBucket) Get(key string) (val []byte, err error) {
	var found bool
	bucket.mutex.RLock()
	defer bucket.mutex.RUnlock()

	val, found = bucket.kvtree.Get(key)
	if !found {
		err = fmt.Errorf("key '%s' not found", key)
		// return nil with no error
	}
	return val, err
}

// GetMultiple returns a batch of documents for the given key
// The document can be any text.
func (bucket *KVBTreeBucket) GetMultiple(keys []string) (docs map[string][]byte, err error) {

	bucket.mutex.RLock()
	defer bucket.mutex.RUnlock()
	docs = make(map[string][]byte)

	for _, key := range keys {
		val, found := bucket.kvtree.Get(key)
		if found {
			docs[key] = val
		}
	}
	return docs, err
}

// Query for documents using JSONPATH
//
// This returns a cursor for a set of parsed documents that match.
// Note that the orderedKeys of the cursor are index numbers, not actual document orderedKeys.
//
// This parses the value into a json document. The parsed document is cached so successive queries
// will be faster.
//
// Eg `$[? @.properties.deviceType=="sensor"]`
//
//  jsonPath contains the query for each document.
//  offset contains the offset in the list of results, sorted by ID
//  limit contains the maximum or of responses, 0 for the default 100
//  orderedKeys can be used to limit the result to documents with the given orderedKeys. Use nil to ignore
//func (bucket *KVBTreeBucket) Query(
//	BucketID string, jsonPath string, orderedKeys []string) (cursor bucketstore.IBucketCursor, err error) {
//
//	//  "github.com/PaesslerAG/jsonpath" - just works, amazing!
//	// Unfortunately no filter with bracket notation $[? @.["title"]=="my title"]
//	// res, err := jsonpath.Get(jsonPath, store.docs)
//	// github.com/ohler55/ojg/jp - seems to work with in-mem maps, no @token in bracket notation
//	//logrus.Infof("jsonPath='%s', limit=%d", args.JsonPathQuery, args.Limit)
//
//	jpExpr, err := jp.ParseString(jsonPath)
//	if err != nil {
//		return nil, err
//	}
//	store.mutex.RLock()
//
//	// build an object tree of potential documents to query
//	var potentialDocs = make(map[string][]byte)
//
//	// no use to continue if the bucket doesn't exist
//	bucket, found := store.getBucket(BucketID, false)
//	if !found {
//		return cursor, fmt.Errorf("bucket '%s' not found", BucketID)
//	}
//	// when the list of orderedKeys is given, reduce to those that actually exist
//	if orderedKeys != nil {
//		for _, key := range orderedKeys {
//			doc, exists := bucket[key]
//			if exists {
//				potentialDocs[key] = doc
//			}
//		}
//	} else {
//		// get all docs
//		for key, docString := range bucket {
//			potentialDocs[key] = docString
//		}
//	}
//
//	// unlock now we have a copy of the document list
//	store.mutex.RUnlock()
//
//	// the query requires a parsed version of json docs
//	var docsToQuery = make(map[string]interface{})
//	for key, jsonDoc := range potentialDocs {
//		doc, found := store.jsonCache[key]
//		if found {
//			// use cached doc
//			docsToQuery[key] = doc
//		} else {
//			// parse and store
//			doc, err = oj.ParseString(string(jsonDoc))
//			if err == nil {
//				docsToQuery[key] = doc
//				store.jsonCache[key] = doc
//			}
//		}
//	}
//	// A big problem with jp.Get is that it returns an interface and we lose the orderedKeys.
//	// The only option is to query each document in order to retain the orderedKeys. That however affects jsonPath formulation.
//	validDocs := jpExpr.Get(docsToQuery)
//
//	// return the json docs instead of the interface.
//	// FIXME: Unfortunately that means marshalling again as we lost the orderedKeys... :(
//	cursorMap := make(map[string][]byte, 0)
//	cursorKeys := make([]string, len(validDocs))
//	for i, validDoc := range validDocs {
//		key := strconv.Itoa(i)
//		cursorKeys[i] = key
//		jsonDoc, _ := oj.Marshal(validDoc)
//		cursorMap[key] = jsonDoc
//	}
//	cursor = NewKVCursor(cursorMap, cursorKeys, 0)
//	return cursor, err
//}

//// Size returns the number of items in the store
//func (bucket *KVBTreeBucket) Size(context.Context, *emptypb.Empty) (*svc.SizeResult, error) {
//	store.mutex.RLock()
//	defer bucket.mutex.RUnlock()
//	res := &svc.SizeResult{
//		Count: int32(len(bucket.kvPairs)),
//	}
//	return res, nil
//}

func (bucket *KVBTreeBucket) ID() string {
	return bucket.BucketID
}

// increment the ref counter when a new bucket is requested
func (bucket *KVBTreeBucket) incrRefCounter() {
	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()
	bucket.refCount++
}

// Info returns the bucket info
func (bucket *KVBTreeBucket) Info() (info *buckets.BucketStoreInfo) {
	info = &buckets.BucketStoreInfo{}
	// are these are full store sizes
	info.NrRecords = int64(bucket.kvtree.Len())
	info.DataSize = -1
	//
	info.Engine = buckets.BackendKVBTree
	info.Id = bucket.BucketID
	return
}

// Set writes a document to the store. If the document exists it is replaced.
// This will store a copy of doc
//
//	A background process periodically checks the change count. When increased:
//	1. Lock the store while copying the index. Unlock when done.
//	2. Stream the in-memory json documents to a temp file.
//	3. If success, move the temp file to the store file using the OS atomic move operation.
func (bucket *KVBTreeBucket) Set(key string, doc []byte) error {
	if key == "" {
		return fmt.Errorf("missing key")
	}

	//docCopy := bytes.NewBuffer(doc).Bytes()
	//docCopy := []byte(string(doc))
	// store the document and object
	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()
	bucket.kvtree.Set(key, utils.Clone(doc))
	bucket.updated(bucket)
	return nil
}

func (bucket *KVBTreeBucket) setUpdateHandler(handler func(bucket *KVBTreeBucket)) {
	bucket.updated = handler
}

// SetMultiple writes a batch of key-values
// Values are copied
func (bucket *KVBTreeBucket) SetMultiple(docs map[string][]byte) (err error) {
	// store the document and object
	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()
	for k, v := range docs {
		bucket.kvtree.Set(k, bytes.NewBuffer(v).Bytes())
	}
	bucket.updated(bucket)
	return nil
}

func NewKVMemBucket(bucketID string) *KVBTreeBucket {
	kvbucket := &KVBTreeBucket{
		BucketID: bucketID,
		refCount: 0,
		mutex:    sync.RWMutex{},
		updated:  nil,
	}
	return kvbucket
}

// NewKVMemBucketFromMap creates a new KVBTreeBucket from a map with bucket data
// Intended for loading a saved store.
func NewKVMemBucketFromMap(bucketID string, data map[string][]byte) *KVBTreeBucket {
	slog.Debug("creating bucket", "bucketID", bucketID)
	kvbucket := &KVBTreeBucket{
		BucketID: bucketID,
		refCount: 0,
		mutex:    sync.RWMutex{},
		updated:  nil,
	}
	for k, v := range data {
		kvbucket.kvtree.Set(k, v)
	}
	return kvbucket
}
