package kvbtree

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/lib/buckets"
	"log/slog"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"time"
)

// KVBTreeStore is an embedded, file backed, in-memory, lightweight, and very fast key-value bucket
// store.
// This is intended for simple use-cases with up to 100K records.
// Interestingly, this simple brute-force store using maps is faster than anything else I've tested and even
// scales up to 1M records. Pretty much all you need for basic databases.
//
// Changes are periodically persisted to file in the background.
//
// Limitations:
//   - No transaction support (basic usage, remember)
//   - Changes are periodically (default 3 second) written to disk
//
// Improvements for future considerations:
//   - Append-only to reduce disk writes for larger databases
//
// --- about jsonpath ---
// This was experimental because of the W3C WoT recommendation, and seems to work well.
// However this is shelved as the Hub has no use-case for it and the other stores don't support it.
//
// Query: jsonPath: `$[?(@.properties.title.name=="title1")]`
//
//	Approx 1.8 msec with a dataset of 1K records (1 result)
//	Approx 23 msec with a dataset of 10K records (1 result)
//	Approx 260 msec with a dataset of 100K records (1 result)
//
// A good overview of jsonpath implementations can be found here:
// > https://cburgmer.github.io/json-path-comparison/
// Two good options for jsonpath queries:
//
//	> github.com/ohler55/ojg/jp
//	> github.com/PaesslerAG/jsonpath
//
// Note that future implementations of this service can change the storage media used while
// maintaining API compatibility.
type KVBTreeStore struct {
	// collection of buckets, one for each Thing, each being a map.
	buckets              map[string]*KVBTreeBucket
	storePath            string       // for file backed storage or "" for in-memory only
	mutex                sync.RWMutex // simple locking is still fast enough
	updateCount          int32        // nr of updates since last save
	backgroundLoopEnded  chan bool
	backgroundLoopEnding chan bool
	writeDelay           time.Duration // delay before writing changes
	// cache for parsed json strings for faster query
	//jsonCache map[string]interface{}
}

// importStoreFile loads the store content into a map and converts it to a map of buckets
// returns an error if the file does not exist
// not concurrent safe
func importStoreFile(storePath string) (docs map[string]*KVBTreeBucket, err error) {
	imported, err := readStoreFile(storePath)
	docs = make(map[string]*KVBTreeBucket)
	if err != nil {
		return nil, err
	}
	for bucketID, bucketData := range imported {
		bucket := NewKVMemBucketFromMap(bucketID, bucketData)
		docs[bucketID] = bucket
	}
	// if the store didn't exist it must be writable successfully in order to continue
	return docs, err
}

// readStoreFile loads the store JSON content into a map
// Returns empty data if storePath is "" (eg an memory-only store)
// Returns the OS error if loading fails
// not concurrent safe
func readStoreFile(storePath string) (docs map[string]map[string][]byte, err error) {
	//docs = make(map[string]*KVBTreeBucket)
	docs = make(map[string]map[string][]byte)
	if storePath == "" {
		return docs, nil
	}
	var rawData []byte
	rawData, err = os.ReadFile(storePath)
	if err == nil {
		err = json.Unmarshal(rawData, &docs)

		if err != nil {
			// todo: chain errors
			slog.Warn("failed read store. Recover with an empty store. Sorry.", "storePath", storePath, "err", err)
		}
	}
	return docs, err
}

// writeStoreFile writes the store to file.
// This creates the folder if it doesn't exist. (the parent must exist)
// Note this is not concurrent safe. Callers must lock or create a shallow copy of the buckets.
//
//	storePath is the full path to the file or "" when ignored
//	docs contains an object map of the store objects
func writeStoreFile(storePath string, docs map[string]map[string][]byte) error {
	slog.Info("writeStoreFile: Flush changes to json store", "storePath", storePath)
	if storePath == "" {
		return nil
	}

	// create the folder if needed
	storeFolder := path.Dir(storePath)
	_, err := os.Stat(storeFolder)
	if os.IsNotExist(err) {
		// folder doesn't exist. Attempt to create it
		slog.Info("Store folder does not exist. Creating it now.", "storeFolder", storeFolder)
		err = os.Mkdir(storeFolder, 0700)
	}
	// If the folder can't be created we're dead in the water
	if err != nil {
		err = fmt.Errorf("unable to create the store folder at '%s': %s", storeFolder, err)
	}
	if err != nil {
		return err
	}

	// serialize the data to json for writing. Use indent for testing and debugging
	//rawData, err := oj.Marshal(docs)
	rawData, err := json.MarshalIndent(docs, "  ", "  ")
	if err != nil {
		// yeah this is pretty fatal too
		err = fmt.Errorf("Unable to marshal documents while saving store to %s: %w", storePath, err)
		slog.Error(err.Error())
		return err
	}
	// First write content to temp file
	// The temp file is opened with 0600 permissions
	tmpName := storePath + ".tmp"
	err = os.WriteFile(tmpName, rawData, 0600)
	if err != nil {
		// ouch, wth?
		err = fmt.Errorf("error while creating tempfile for jsonstore: %w", err)
		slog.Error(err.Error())
		return err
	}

	// move the temp file to the final store file.
	// this replaces the file if it already exists
	err = os.Rename(tmpName, storePath)
	if err != nil {
		err = fmt.Errorf("error while moving tempfile to jsonstore '%s': %w", storePath, err)
		slog.Error(err.Error())
		return err
	}
	return nil
}

// autoSaveLoop periodically saves changes to the store
func (store *KVBTreeStore) autoSaveLoop() {
	slog.Info("auto-save loop started")

	defer close(store.backgroundLoopEnded)

	for {
		select {
		case <-store.backgroundLoopEnding:
			slog.Info("Autosave loop ended")
			return
		case <-time.After(store.writeDelay):
			//store.mutex.Lock()
			if atomic.LoadInt32(&store.updateCount) > int32(0) {
				// make a shallow copy for writing to avoid a lock during write to disk
				exportedCopy := store.Export()
				atomic.StoreInt32(&store.updateCount, 0)
				//store.mutex.Unlock()

				// nothing we can do here. error is already logged
				// FIXME: use separate write lock
				_ = writeStoreFile(store.storePath, exportedCopy)
			} else {
				//store.mutex.Unlock()
			}
		}
	}
}

// Close the store and stop the background update.
// If any changes are remaining then write to disk now.
func (store *KVBTreeStore) Close() error {
	var err error
	slog.Info("closing store for client", "storePath", store.storePath)

	if store.buckets == nil || store.backgroundLoopEnding == nil {
		return fmt.Errorf("store already closed")
	}

	store.backgroundLoopEnding <- true

	// wait for the background loop to end
	<-store.backgroundLoopEnded

	// flush any remaining changes
	if atomic.LoadInt32(&store.updateCount) > int32(0) {
		// note Export does an rlock
		exportedCopy := store.Export()
		err = writeStoreFile(store.storePath, exportedCopy)
	}
	store.buckets = nil
	slog.Info("Store close completed. Background loop ended", "storePath", store.storePath)
	return err
}

// export returns a map of the given bucket
// this copies the keys and values into a new map
func (store *KVBTreeStore) Export() map[string]map[string][]byte {
	store.mutex.RLock()
	defer store.mutex.RUnlock()
	var exportedCopy = make(map[string]map[string][]byte)

	for bucketID, bucket := range store.buckets {
		bucketExport := bucket.Export()
		exportedCopy[bucketID] = bucketExport
	}
	return exportedCopy
}

// GetBucket returns a bucket and creates it if it doesn't exist
func (store *KVBTreeStore) GetBucket(bucketID string) (bucket buckets.IBucket) {

	if store.buckets == nil {
		panic("store is not open")
	}
	store.mutex.Lock()
	kvBucket, _ := store.buckets[bucketID]
	if kvBucket == nil {
		kvBucket = NewKVMemBucket(bucketID)
		kvBucket.setUpdateHandler(store.onBucketUpdated)
		store.buckets[bucketID] = kvBucket
		bucket = kvBucket
	}
	if kvBucket != nil {
		kvBucket.incrRefCounter()
	}
	store.mutex.Unlock()
	return kvBucket
}

// callback handler for notification that a bucket has been modified
func (store *KVBTreeStore) onBucketUpdated(bucket *KVBTreeBucket) {
	// at this point we don't need the bucket but this might change with more fine grained update tracking
	_ = bucket
	atomic.AddInt32(&store.updateCount, 1)
}

// Open the store and start the background loop for saving changes
func (store *KVBTreeStore) Open() error {
	slog.Info("Opening store", "storePath", store.storePath)
	var err error
	store.mutex.Lock()
	defer store.mutex.Unlock()

	if store.buckets != nil {
		return fmt.Errorf("store already open")
	}
	store.buckets, err = importStoreFile(store.storePath)
	// recover from bad file. Missing file is okay.
	if err != nil {
		if os.IsNotExist(err) {
			// store doesn't yet exist. This is okay
		} else {
			return fmt.Errorf("unknown error reading store '%s': %w", store.storePath, err)
		}
		// write an empty store to make sure the location is writable
		store.buckets = make(map[string]*KVBTreeBucket)
		dummy := make(map[string]map[string][]byte)
		err = writeStoreFile(store.storePath, dummy)
		if err != nil {
			// unable to recover. Hitting a dead end
			return fmt.Errorf("failed creating store file: '%w'", err)
		}
	}
	// after loading set the handler for all buckets
	for _, kvBucket := range store.buckets {
		kvBucket.setUpdateHandler(store.onBucketUpdated)
	}

	store.backgroundLoopEnding = make(chan bool)
	store.backgroundLoopEnded = make(chan bool)
	if err == nil {
		go store.autoSaveLoop()
	}
	// allow a context switch to start the autoSaveLoop to avoid problems
	// if the store is closed immediately.
	//time.Sleep(time.Millisecond)
	return err
}

//// Size returns the number of items in the store
//func (store *KVBTreeStore) Size(context.Context, *emptypb.Empty) (*svc.SizeResult, error) {
//	store.mutex.RLock()
//	defer store.mutex.RUnlock()
//	res := &svc.SizeResult{
//		Count: int32(len(store.docs)),
//	}
//	return res, nil
//}

// SetWriteDelay sets the delay for writing after a change
func (store *KVBTreeStore) SetWriteDelay(delay time.Duration) {
	store.writeDelay = delay
}

// NewKVStore creates a store instance and load it with saved documents.
// Run Connect to start the background loop and Stop to end it.
//
//	storeFile path to storage file or "" for in-memory only
func NewKVStore(storePath string) (store *KVBTreeStore) {
	writeDelay := time.Duration(3000) * time.Millisecond
	store = &KVBTreeStore{
		//jsonDocs:             make(map[string]string),
		buckets:              nil, // will be set after open
		storePath:            storePath,
		backgroundLoopEnding: nil,
		backgroundLoopEnded:  nil,
		mutex:                sync.RWMutex{},
		writeDelay:           writeDelay,
		//jsonCache:            make(map[string]interface{}),
	}
	return store
}
