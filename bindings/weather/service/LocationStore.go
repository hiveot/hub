package service

import (
	"fmt"
	"sync"

	"github.com/hiveot/hivekit/go/buckets"
	"github.com/hiveot/hivekit/go/buckets/kvbtree"
	"github.com/hiveot/hub/bindings/weather/config"
	jsoniter "github.com/json-iterator/go"
)

// LocationStore stores configured locationStore in a bucket store
type LocationStore struct {
	bucketStore     buckets.IBucketStore
	locationsBucket buckets.IBucket
	locations       []config.WeatherLocation
	// lock location updates
	mux sync.RWMutex
}

// Add a location to the store
// If the location exists then it is updated.
func (svc *LocationStore) Add(loc config.WeatherLocation) error {
	svc.mux.Lock()
	defer svc.mux.Unlock()
	if loc.Latitude == "" || loc.Longitude == "" || loc.ID == "" {
		return fmt.Errorf("missing location or ID")
	}
	locJSON, err := jsoniter.Marshal(loc)
	if err != nil {
		return err
	}
	// update if location exists
	for i, loc2 := range svc.locations {
		if loc2.ID == loc.ID {
			svc.locations[i] = loc
			err = svc.locationsBucket.Set(loc.ID, locJSON)
			return err
		}
	}
	// add new location
	svc.locations = append(svc.locations, loc)
	err = svc.locationsBucket.Set(loc.ID, locJSON)
	return err
}

// Close the location store
func (svc *LocationStore) Close() {
	svc.mux.Lock()
	defer svc.mux.Unlock()
	if svc.locationsBucket != nil {
		_ = svc.locationsBucket.Close()
		_ = svc.bucketStore.Close()
	}
}

// Get returns the configuration of a location
func (svc *LocationStore) Get(id string) (loc config.WeatherLocation, found bool) {
	for _, loc = range svc.locations {
		if loc.ID == id {
			return loc, true
		}
	}
	return loc, false
}

// ForEach invokes the callback for each enabled location
// this is concurrent safe
func (svc *LocationStore) ForEach(cb func(location config.WeatherLocation)) {
	svc.mux.RLock()
	iter := append([]config.WeatherLocation{}, svc.locations...)
	svc.mux.RUnlock()
	for _, cfg := range iter {
		cb(cfg)
	}
}

// Open the storage bucket
func (svc *LocationStore) Open() error {
	svc.mux.Lock()
	defer svc.mux.Unlock()
	// load locationStore
	err := svc.bucketStore.Open()
	if err != nil {
		return err
	}
	// load locationStore from store
	svc.locationsBucket = svc.bucketStore.GetBucket(weatherLocationsKey)
	cursor, err := svc.locationsBucket.Cursor()
	if err != nil {
		return err
	}
	for _, v, valid := cursor.First(); valid; _, v, valid = cursor.Next() {
		var weatherLocation config.WeatherLocation
		err = jsoniter.Unmarshal(v, &weatherLocation)
		svc.locations = append(svc.locations, weatherLocation)
	}
	return err
}

// Remove a location from the store
func (svc *LocationStore) Remove(id string) {
	svc.mux.Lock()
	defer svc.mux.Unlock()
	panic("remove not yet implemented")
}

// Update a location in the store
func (svc *LocationStore) Update(loc config.WeatherLocation) {
	_ = svc.Add(loc)
}

func NewLocationStore(storePath string) *LocationStore {
	bucketStore := kvbtree.NewKVStore(storePath)

	store := &LocationStore{
		bucketStore: bucketStore,
	}
	return store
}
