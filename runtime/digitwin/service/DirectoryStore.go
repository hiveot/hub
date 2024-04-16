package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
	"sync"
)

const TDBucketName = "td"

// DirectoryService stores, reads and queries TD documents
// This service keeps an in-memory Thing Description instance backed by a persistent bucket store.
//
// When a TDD is updated, its form entries are modified to point to the Hub, after
// which the new TDD is persisted in the bucket store under the name {TDBucketName}
// with the key of thingID.
//
// Read and query methods using the in-memory cache for fast performance.
type DirectoryService struct {
	store    buckets.IBucketStore
	tdBucket buckets.IBucket

	// tdCache holds an in-memory cache version of stored TDs
	tdCache map[string]*things.TD
	// list of thingIDs used as a consistent iterator for reading batches
	thingKeys []string
	// mutex for accessing the cache
	cachemux sync.RWMutex
}

// LoadCacheFromStore loads the cache from store
func (svc *DirectoryService) LoadCacheFromStore() error {
	svc.cachemux.Lock()
	defer svc.cachemux.Unlock()
	cursor, err := svc.tdBucket.Cursor(context.Background())
	if err != nil {
		return err
	}
	svc.thingKeys = make([]string, 0, 1000)
	svc.tdCache = make(map[string]*things.TD)
	//k, v, valid := cursor.First()
	//_ = k
	//_ = v
	//_ = valid
	for {
		// read in batches of 1000 TD documents
		tdmap, itemsRemaining := cursor.NextN(1000)
		for thingID, tddjson := range tdmap {
			svc.thingKeys = append(svc.thingKeys, thingID)
			tdd := &things.TD{}
			err = json.Unmarshal(tddjson, tdd)
			if err != nil {
				slog.Error("LoadCacheFromStore: Unmarshalling TDDoc failed. TD ignored",
					slog.String("thingID", thingID),
					slog.String("err", err.Error()))
			} else {
				svc.tdCache[thingID] = tdd
			}
		}
		if !itemsRemaining {
			break
		}
	}
	return nil
}

// QueryTDs the collection of TD documents
//func (svc *DirectoryService) QueryTDs(query string) (tddList []string, err error) {
//	// TBD: query based on what?
//	return nil, fmt.Errorf("not yet implemented")
//}

// ReadThing returns the TD document in json format for the given Thing ID
func (svc *DirectoryService) ReadThing(thingID string) (td *things.TD, err error) {
	svc.cachemux.RLock()
	defer svc.cachemux.RUnlock()
	td, found := svc.tdCache[thingID]
	if !found {
		err = fmt.Errorf("Thing with ID '%s' not found", thingID)
	}
	return td, err
}

// ReadThings returns a list of TD documents
//
//	offset is the offset in the list
//	limit is the maximum number of records to return
func (svc *DirectoryService) ReadThings(offset, limit int) (tdList []*things.TD, err error) {
	tdList = make([]*things.TD, 0, limit)
	svc.cachemux.RLock()
	defer svc.cachemux.RUnlock()
	// Use the thingKeys index to ensure consistent iteration and to quickly
	// skip offset items (maps are not consistent between iterations)
	if offset >= len(svc.thingKeys) {
		// empty result
		return []*things.TD{}, nil
	}
	if offset+limit > len(svc.thingKeys) {
		limit = len(svc.thingKeys) - offset
	}
	tdKeys := svc.thingKeys[offset:limit]
	// add the TD documents
	for _, k := range tdKeys {
		v := svc.tdCache[k]
		tdList = append(tdList, v)
	}
	return tdList, nil
}

// RemoveThing deletes the TD document from the given agent with the ThingID
func (svc *DirectoryService) RemoveThing(senderID string, thingID string) error {
	slog.Info("RemoveThing",
		slog.String("thingID", thingID),
		slog.String("senderID", senderID))
	// remove from both cache and bucket
	err := svc.tdBucket.Delete(thingID)
	svc.cachemux.Lock()
	defer svc.cachemux.Unlock()
	delete(svc.tdCache, thingID)
	// fast delete from the index array
	for i, key := range svc.thingKeys {
		if key == thingID {
			svc.thingKeys[i] = svc.thingKeys[len(svc.thingKeys)-1]
			svc.thingKeys = svc.thingKeys[:len(svc.thingKeys)-1]
		}
	}
	return err
}

// Start the directory service and open the directory stored TD bucket
func (svc *DirectoryService) Start() (err error) {
	slog.Info("Starting DirectoryService")
	// fill the in-memory cache
	err = svc.LoadCacheFromStore()
	if err != nil {
		return err
	}
	return err
}

// Stop the service
func (svc *DirectoryService) Stop() {
	slog.Info("Stopping DirectoryService")
	if svc.tdBucket != nil {
		_ = svc.tdBucket.Close()
	}
}

// UpdateThing adds or updates the Thing Description document
// Added things are written to the store.
func (svc *DirectoryService) UpdateThing(senderID string, thingID string, tdd *things.TD) error {
	slog.Info("UpdateThing",
		slog.String("senderID", senderID),
		slog.String("thingID", thingID))

	// TODO: update the forms to point to the Hub instead of the device
	svc.cachemux.Lock()
	defer svc.cachemux.Unlock()
	_, exists := svc.tdCache[thingID]
	// append the key if it doesn't yet exist
	if !exists {
		svc.thingKeys = append(svc.thingKeys, thingID)
	}
	svc.tdCache[thingID] = tdd

	// serialize to persist
	updatedTDD, _ := json.Marshal(tdd)
	err := svc.tdBucket.Set(thingID, updatedTDD)
	return err
}

// NewDirectoryStore creates a new service instance for the directory of Thing TD documents.
//
//	store is an instance of the bucket store to store the directory data. This is opened by 'Start' and closed by 'Stop'
func NewDirectoryStore(store buckets.IBucketStore) *DirectoryService {
	tdBucket := store.GetBucket(TDBucketName)
	svc := &DirectoryService{
		store:    store,
		tdBucket: tdBucket,
	}
	return svc
}
