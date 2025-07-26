package service

import (
	"fmt"
	"github.com/hiveot/hub/bindings/weather/providers"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	jsoniter "github.com/json-iterator/go"
	"sync"
)

// LocationStore stores configured locationStore in a bucket store
type LocationStore struct {
	bucketStore     buckets.IBucketStore
	locationsBucket buckets.IBucket
	locations       []providers.WeatherLocationConfig
	// lock location updates
	mux sync.RWMutex
}

// Add a location to the store
func (svc *LocationStore) Add(loc *providers.WeatherLocationConfig) error {
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
			svc.locations[i] = *loc
			err = svc.locationsBucket.Set(loc.ID, locJSON)
			return err
		}
	}
	// add new location
	svc.locations = append(svc.locations, *loc)
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

// ForEach invokes the callback for each enabled location
// this is concurrent safe
func (svc *LocationStore) ForEach(cb func(providers.WeatherLocationConfig)) {
	svc.mux.RLock()
	iter := append([]providers.WeatherLocationConfig{}, svc.locations...)
	svc.mux.RUnlock()
	for _, cfg := range iter {
		if cfg.ForecastEnabled {
			cb(cfg)
		}
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
		var weatherLocation providers.WeatherLocationConfig
		err = jsoniter.Unmarshal(v, &weatherLocation)
		svc.locations = append(svc.locations, weatherLocation)
	}
	return err
}

// Remove a location from the store
func (svc *LocationStore) Remove(loc *providers.WeatherLocationConfig) {
	svc.mux.Lock()
	defer svc.mux.Unlock()
	panic("remove not yet implemented")
}
func NewLocationStore(storePath string) *LocationStore {
	bucketStore := kvbtree.NewKVStore(storePath)

	store := &LocationStore{
		bucketStore: bucketStore,
	}
	return store
}
