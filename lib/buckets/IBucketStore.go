// Package bucketstore is a storage library for use by the services.
// The bucket store is primarily used by the state service which provides this store as a service to multiple clients.
// This package defines an API to use the store with several implementations.
package buckets

// Available embedded bucket store implementations with low memory overhead
const (
	BackendKVBTree = "kvbtree" // fastest and best for small to medium amounts of data (dependent on available memory)
	BackendPebble  = "pebble"  // a good middle ground between performance and memory
)

// BucketStoreInfo information of the bucket or the store
type BucketStoreInfo struct {
	// DataSize contains the size of data in the store or bucket.
	// -1 if not available.
	DataSize int64

	// Engine describes the storage engine of the store, eg kvbtree, pebble
	Engine string

	// The store or bucket identifier, eg thingID, appID
	Id string

	// NrRecords holds the number of records in the store or bucket.
	// -1 if not available.
	NrRecords int64
}

// IBucketStore defines the interface to a simple key-value embedded bucket store.
//   - organizes data into buckets
//   - open/close buckets as a transaction, if transactions are available
//   - get/set single or multiple key/value pairs
//   - delete key/value
//   - cursor based seek and iteration
//     Streaming data into a bucket is not supported
//     Various implementations are available to the services to use.
//
// TODO: add refcount for multiple consumers of the store so it can be closed when done.
type IBucketStore interface {
	// GetBucket opens and returns a bucket to use.
	// This creates the bucket if it doesn't exist.
	// Use bucket.Close() to close the bucket and release its resources.
	GetBucket(bucketID string) (bucket IBucket)

	// Close the store and release its resources
	Close() error

	// Open the store
	Open() error

	// Info returns bucket store information
	//Info() *BucketStoreInfo
}

// IBucket defines the interface to a store key-value bucket
type IBucket interface {

	// Close the bucket and release its resources
	Close() error

	// Cursor creates a new bucket cursor for iterating the bucket
	// cursor.Remove must be called after use to release any read transactions
	// The cursor points before the first item to iterate.
	Cursor() (cursor IBucketCursor, err error)

	// Delete removes the key-value pair from the bucket store
	// Returns nil if the key is deleted or doesn't exist.
	// Returns an error if the database cannot be updated.
	Delete(key string) (err error)

	// Get returns the document for the given key
	// Returns an error if the key isn't found in the bucket or if the database cannot be read
	Get(key string) (value []byte, err error)

	// GetMultiple returns a batch of documents with existing keys
	// if a key does not exist it will not be included in the result.
	// An error is return if the database cannot be read.
	GetMultiple(keys []string) (keyValues map[string][]byte, err error)

	// ID returns the bucket's ID
	ID() string

	// Info returns the bucket information, when available
	Info() *BucketStoreInfo

	// Set sets a document with the given key
	// This stores a copy of value.
	// An error is returned if either the bucketID or the key is empty
	Set(key string, value []byte) error

	// SetMultiple sets multiple documents in a batch update
	// This stores a copy of docs.
	// If the transaction fails an error is returned and no changes are made.
	SetMultiple(docs map[string][]byte) (err error)

	// Status returns the bucket status
	//Status() BucketStoreStatus
}

// IBucketCursor provides the prev/next cursor based iterator on a range
type IBucketCursor interface {
	// BucketID is the ID of the bucket this cursor iterates
	BucketID() string

	// First positions the cursor at the first key in the ordered list
	// valid is false if the bucket is empty
	First() (key string, value []byte, valid bool)

	// Last positions the cursor at the last key in the ordered list
	// valid is false if the bucket is empty
	Last() (key string, value []byte, valid bool)

	// Next moves the cursor to the next key from the current cursor
	// First() or Seek must have been called first.
	// valid is false if the iterator has reached the end and no valid value is returned.
	Next() (key string, value []byte, valid bool)

	// NextN moves the cursor to the next N places from the current cursor
	// and return a map with the N key-value pairs.
	// If the iterator reaches the end it returns the remaining items and itemsRemaining is false
	// If the cursor is already at the end, the resulting map is empty and itemsRemaining is also false.
	// Intended to speed up with batch iterations over rpc.
	NextN(steps uint) (docs map[string][]byte, itemsRemaining bool)

	// Prev moves the cursor to the previous key from the current cursor
	// Last() or Seek must have been called first.
	// valid is false if the iterator has reached the beginning and no valid value is returned.
	Prev() (key string, value []byte, valid bool)

	// PrevN moves the cursor back N places from the current cursor and returns a map with
	// the N key-value pairs.
	// Intended to speed up with batch iterations over rpc.
	// If the iterator reaches the beginning it returns the remaining items and itemsRemaining is false
	// If the cursor is already at the beginning, the resulting map is empty and itemsRemaining is also false.
	PrevN(steps uint) (docs map[string][]byte, itemsRemaining bool)

	// Release close the cursor and release its resources.
	// This invalidates all values obtained from the cursor
	Release()

	// Seek positions the cursor at the given searchKey and corresponding value.
	// If the key is not found, the next key is returned.
	// valid is false if the iterator has reached the end and no valid value is returned.
	Seek(searchKey string) (key string, value []byte, valid bool)

	// Skip N items without reading them
	// If N is negative it skips backwards.
	Skip(steps int) (itemsRemaining bool)

	// Stream the content of the cursor to the provided function
	//Stream(cb func(key string, value []byte, done bool) error)
}
