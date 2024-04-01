package valuestore

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"log/slog"
	"sync"
	"time"

	"github.com/hiveot/hub/lib/things"
)

// ThingValueStore holds the most recent property and event values of things.
// It persists a record for each Thing containing a map of the most recent properties.
type ThingValueStore struct {
	// bucket to persist things properties with a serialized property map for each things
	store buckets.IBucket

	// in-memory cache of the latest things values by things address
	cache map[string]things.ThingValueMap
	// mutex for read/writing the cache
	cacheMux sync.RWMutex // mutex for the following two fields
	// map o things addresses and their change status
	changedThings map[string]bool
}

// LoadProps loads the cached value of a Thing properties on demand.
// To be invoked before reading and writing Thing properties to ensure the cache is loaded.
// This immediately returns if a record for the Thing was already loaded.
// Returns true if a cache value exists, false if the things address was added to the cache
func (svc *ThingValueStore) LoadProps(thingAddr string) (found bool) {
	svc.cacheMux.Lock()
	props, found := svc.cache[thingAddr]
	defer svc.cacheMux.Unlock()

	if found {
		return
	}
	val, _ := svc.store.Get(thingAddr)

	if val == nil {
		// create a new record with things properties
		props = things.NewThingValueMap()
	} else {
		// decode the record with things properties
		err := json.Unmarshal(val, &props)
		if err != nil {
			slog.Error("stored 'latest' properties of things can't be unmarshalled. Clean start.",
				slog.String("thingAddr", thingAddr), slog.String("err", err.Error()))
			props = things.NewThingValueMap()
		}
	}
	svc.cache[thingAddr] = props
	return
}

// GetProperties returns the latest value of things properties and events as a list of properties
//
//	thingAddr is the address the things is reachable at. Usually the agentID/thingID.
//	names is optional and can be used to limit the resulting array of values. Use nil to get all properties.
func (svc *ThingValueStore) GetProperties(thingAddr string, names []string) (props things.ThingValueMap) {
	props = things.NewThingValueMap()

	// ensure this thing has its properties cache loaded
	svc.LoadProps(thingAddr)

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	thingCache, _ := svc.cache[thingAddr]
	if names != nil && len(names) > 0 {
		// filter the requested property/event names
		for _, name := range names {
			tv := thingCache.Get(name)
			if tv != nil {
				props.Set(name, tv)
			}
		}
		return props
	}

	// default: get all available property/event names
	props = thingCache
	//for _, value := range thingCache {
	//	propList = append(propList, value)
	//}
	return props
}

// HandleAddValue is the handler of update to a things's event/property values
// used to update the properties cache.
// isAction indicates the value is an action.
func (svc *ThingValueStore) HandleAddValue(addtv *things.ThingValue) {
	// ensure the Thing has its properties cache loaded
	thingAddr := addtv.AgentID + "/" + addtv.ThingID
	if addtv.CreatedMSec <= 0 {
		addtv.CreatedMSec = time.Now().UnixMilli()
	}

	svc.LoadProps(thingAddr)
	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()
	thingCache, _ := svc.cache[thingAddr]

	if addtv.Key == transports.EventNameProps {
		// the value holds a map of property name:value pairs, add each one individually
		// in order to retain the sender and created timestamp.
		props := make(map[string]any)
		err := json.Unmarshal(addtv.Data, &props)
		if err != nil {
			return // data is not used
		}
		// turn each value into a ThingValue object
		for propName, propValue := range props {
			propValueString := fmt.Sprint(propValue)
			tv := things.NewThingValue(transports.MessageTypeEvent,
				addtv.AgentID, addtv.ThingID, propName, []byte(propValueString), addtv.SenderID)
			tv.CreatedMSec = addtv.CreatedMSec

			// in case events arrive out of order, only update if the addtv is newer
			existingLatest := thingCache.Get(propName)
			if existingLatest == nil || tv.CreatedMSec > existingLatest.CreatedMSec {
				thingCache.Set(propName, tv)
			}
		}
	} else {
		// events or action messages
		// in case events arrive out of order, only update if the addtv is newer
		existingLatest := thingCache.Get(addtv.Key)
		if existingLatest == nil || addtv.CreatedMSec > existingLatest.CreatedMSec {
			thingCache.Set(addtv.Key, addtv)
		}
	}
	svc.changedThings[thingAddr] = true
}

// SaveChanges writes modified cached properties to the underlying store.
// this returns the last encountered error, although writing is attempted for all changes
func (svc *ThingValueStore) SaveChanges() (err error) {

	// try to minimize the lock time for each Thing
	// start with using a read lock to collect the addresses of Things that changed
	var changedThings = make([]string, 0)
	svc.cacheMux.RLock()
	for thingAddr, hasChanged := range svc.changedThings {
		if hasChanged {
			changedThings = append(changedThings, thingAddr)
		}
	}
	svc.cacheMux.RUnlock()

	// next, iterate the changes and lock only to serialize the properties record
	for _, thingAddr := range changedThings {
		var propsJSON []byte
		// lock only for marshalling the properties
		svc.cacheMux.Lock()
		props, found := svc.cache[thingAddr]
		if !found {
			// Should never happen
			err = fmt.Errorf("thingsChanged is set for address '%s' but no properties are present. Ignored", thingAddr)
			slog.Error(err.Error())
		} else {
			propsJSON, _ = json.Marshal(props)
		}
		svc.changedThings[thingAddr] = false
		svc.cacheMux.Unlock()

		// buckets manage their own locks
		if propsJSON != nil {
			err2 := svc.store.Set(thingAddr, propsJSON)
			if err2 != nil {
				err = err2
			}
		}
	}
	return err
}

// NewThingValueStore creates a new instance of the storage for Thing's latest property values
func NewThingValueStore(storage buckets.IBucket) *ThingValueStore {

	svc := &ThingValueStore{
		store:         storage,
		cache:         make(map[string]things.ThingValueMap),
		cacheMux:      sync.RWMutex{},
		changedThings: make(map[string]bool),
	}
	return svc
}
