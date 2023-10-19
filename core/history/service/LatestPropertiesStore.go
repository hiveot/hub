package service

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/vocab"
	"log/slog"
	"sync"

	"github.com/hiveot/hub/lib/thing"
)

// ThingPropertyValues is a map of Thing property name to value
type ThingPropertyValues map[string]*thing.ThingValue

// LatestPropertiesStore holds the most recent property and event values of things.
// It persists a record for each Thing containing a map of the most recent properties.
type LatestPropertiesStore struct {
	// bucket to persist thing properties with a serialized property map for each thing
	store buckets.IBucket

	// in-memory cache of the latest thing values by thing address
	cache map[string]ThingPropertyValues
	// mutex for read/writing the cache
	cacheMux sync.RWMutex // mutex for the following two fields
	// map o thing addresses and their change status
	changedThings map[string]bool
}

// LoadProps loads the cached value of a Thing properties on demand.
// To be invoked before reading and writing Thing properties to ensure the cache is loaded.
// This immediately returns if a record for the Thing was already loaded.
// Returns true if a cache value exists, false if the thing address was added to the cache
func (srv *LatestPropertiesStore) LoadProps(thingAddr string) (found bool) {
	srv.cacheMux.Lock()
	props, found := srv.cache[thingAddr]
	defer srv.cacheMux.Unlock()

	if found {
		return
	}
	val, _ := srv.store.Get(thingAddr)

	if val == nil {
		// create a new record with thing properties
		props = make(ThingPropertyValues)
	} else {
		// decode the record with thing properties
		err := json.Unmarshal(val, &props)
		if err != nil {
			slog.Error("stored 'latest' properties of thing can't be unmarshalled. Clean start.",
				slog.String("thingAddr", thingAddr), slog.String("err", err.Error()))
			props = make(ThingPropertyValues)
		}
	}
	srv.cache[thingAddr] = props
	return
}

// GetProperties returns the latest value of thing properties and events as a list of properties
//
//	 thingAddr is the address the thing is reachable at. Usually the agentID/thingID.
//		names is optional and can be used to limit the resulting array of values. Use nil to get all properties.
func (srv *LatestPropertiesStore) GetProperties(thingAddr string, names []string) (propList []*thing.ThingValue) {
	propList = make([]*thing.ThingValue, 0)

	// ensure this thing has its properties cache loaded
	srv.LoadProps(thingAddr)

	srv.cacheMux.RLock()
	defer srv.cacheMux.RUnlock()
	thingCache, _ := srv.cache[thingAddr]
	if names != nil && len(names) > 0 {
		// get the requested property/event names
		for _, name := range names {
			value, found := thingCache[name]
			if found {
				propList = append(propList, value)
			}
		}
		return propList
	}

	// default: get all available property/event names
	for _, value := range thingCache {
		propList = append(propList, value)
	}
	return propList
}

// HandleAddValue is the handler of update to a thing's event/property values
// used to update the properties cache.
// isAction indicates the value is an action.
func (srv *LatestPropertiesStore) HandleAddValue(event *thing.ThingValue, isAction bool) {
	// ensure the Thing has its properties cache loaded
	thingAddr := event.AgentID + "/" + event.ThingID
	srv.LoadProps(thingAddr)

	srv.cacheMux.Lock()
	defer srv.cacheMux.Unlock()

	// TODO: differentiate between action and event values
	// right now actions are not added.
	if isAction {
		return
	}
	thingCache, _ := srv.cache[thingAddr]

	if event.Name == vocab.EventNameProps {
		// this is a properties event that holds a map of property name:values
		props := make(map[string][]byte)
		err := json.Unmarshal(event.Data, &props)
		if err != nil {
			return // data is not used
		}
		// turn each value into a ThingValue object
		for propName, propValue := range props {
			tv := thing.NewThingValue(event.AgentID, event.ThingID, propName, propValue)
			tv.CreatedMSec = event.CreatedMSec

			// in case events arrive out of order, only update if the event is newer
			existingLatest, found := thingCache[propName]
			// FIXME. This will be wrong with different timezones
			if !found || tv.CreatedMSec > existingLatest.CreatedMSec {
				thingCache[propName] = tv
			}
		}
	} else {
		// in case events arrive out of order, only update if the event is newer
		existingLatest, found := thingCache[event.Name]
		if !found || event.CreatedMSec > existingLatest.CreatedMSec {
			thingCache[event.Name] = event
		}
	}
	srv.changedThings[thingAddr] = true
}

// SaveChanges writes modified cached properties to the underlying store.
// this returns the last encountered error, although writing is attempted for all changes
func (srv *LatestPropertiesStore) SaveChanges() (err error) {

	// try to minimize the lock time for each Thing
	// start with using a read lock to collect the addresses of Things that changed
	var changedThings = make([]string, 0)
	srv.cacheMux.RLock()
	for thingAddr, hasChanged := range srv.changedThings {
		if hasChanged {
			changedThings = append(changedThings, thingAddr)
		}
	}
	srv.cacheMux.RUnlock()

	// next, iterate the changes and lock only to serialize the properties record
	for _, thingAddr := range changedThings {
		var propsJSON []byte
		// lock only for marshalling the properties
		srv.cacheMux.Lock()
		props, found := srv.cache[thingAddr]
		if !found {
			// Should never happen
			err = fmt.Errorf("thingsChanged is set for address '%s' but no properties are present. Ignored", thingAddr)
			slog.Error(err.Error())
		} else {
			propsJSON, _ = json.Marshal(props)
		}
		srv.changedThings[thingAddr] = false
		srv.cacheMux.Unlock()

		// buckets manage their own locks
		if propsJSON != nil {
			err2 := srv.store.Set(thingAddr, propsJSON)
			if err2 != nil {
				err = err2
			}
		}
	}
	return err
}

// NewPropertiesStore creates a new instance of the storage for Thing's latest property values
func NewPropertiesStore(storage buckets.IBucket) *LatestPropertiesStore {

	propsStore := &LatestPropertiesStore{
		store:         storage,
		cache:         make(map[string]ThingPropertyValues),
		cacheMux:      sync.RWMutex{},
		changedThings: make(map[string]bool),
	}
	return propsStore
}
