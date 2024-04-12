package valueservice

import (
	"encoding/json"
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/runtime/api"
	"log/slog"
	"sync"
	"time"

	"github.com/hiveot/hub/lib/things"
)

const PropertiesBucketName = "properties"

// ValueService holds the most recent property and event values of things.
// It persists a record for each Thing containing a map of the most recent properties.
type ValueService struct {
	cfg *ValueStoreConfig

	// bucket to persist things properties with a serialized property map for each thing
	store  buckets.IBucketStore
	bucket buckets.IBucket

	// in-memory cache of the latest things values by thingID
	cache map[string]things.ThingMessageMap
	// mutex for read/writing the cache
	cacheMux sync.RWMutex // mutex for the following two fields
	// map of thingsIDs and their change status
	changedThings map[string]bool
}

// GetProperties returns the latest value of things properties and events as a list of properties
//
//	thingID of the thing to read.
//	names is optional and can be used to limit the resulting array of values. Use nil to get all properties.
func (svc *ValueService) GetProperties(thingID string, keys []string) (props things.ThingMessageMap) {
	props = things.NewThingMessageMap()

	// ensure this thing has its properties cache loaded
	svc.LoadProps(thingID)

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	thingCache, _ := svc.cache[thingID]
	if keys != nil && len(keys) > 0 {
		// filter the requested property/event keys
		for _, name := range keys {
			tv := thingCache.Get(name)
			if tv != nil {
				props.Set(name, tv)
			}
		}
		return props
	}

	// default: get all available property/event keys
	props = thingCache
	//for _, value := range thingCache {
	//	propList = append(propList, value)
	//}
	return props
}

// HandleMessage handles thing events and value service action requests
// used to update the properties cache.
// isAction indicates the value is an action.
func (svc *ValueService) HandleMessage(msg *things.ThingMessage) ([]byte, error) {
	// ensure the Thing has its properties cache loaded
	if msg.CreatedMSec <= 0 {
		msg.CreatedMSec = time.Now().UnixMilli()
	}

	if msg.MessageType == vocab.MessageTypeEvent {
		return nil, svc.HandleEvent(msg)
	} else {
		return svc.HandleAction(msg)
	}
}

// HandleAction requests, including reading latest value
func (svc *ValueService) HandleAction(action *things.ThingMessage) (reply []byte, err error) {
	if action.Key == api.ValueServiceGetActionsMethod {
		return nil, fmt.Errorf("not yet implemented")
	} else if action.Key == api.ValueServiceGetPropertiesMethod {
		args := api.GetPropertiesArgs{}
		err = json.Unmarshal(action.Data, &args)
		if err != nil {
			return nil, err
		}
		resp := api.GetPropertiesResp{}
		resp.Props = svc.GetProperties(args.ThingID, args.Keys)
		reply, err = json.Marshal(resp)
		return reply, err
	}
	return nil, fmt.Errorf("unknown action '%s'", action.Key)
}

// HandleEvent stores the latest event and property event values
func (svc *ValueService) HandleEvent(msg *things.ThingMessage) error {
	svc.LoadProps(msg.ThingID)
	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()
	thingCache, _ := svc.cache[msg.ThingID]

	if msg.Key == vocab.EventTypeProperties {

		// the value holds a map of property name:value pairs, add each one individually
		// in order to retain the sender and created timestamp.
		props := make(map[string]any)
		err := json.Unmarshal(msg.Data, &props)
		if err != nil {
			slog.Warn("HandleEvent; Error unmarshalling props",
				slog.String("err", err.Error()),
				slog.String("senderID", msg.SenderID))
			return err
		}
		// turn each value into a ThingMessage object
		for propName, propValue := range props {
			propValueString := fmt.Sprint(propValue)
			tv := things.NewThingMessage(vocab.MessageTypeEvent,
				msg.ThingID, propName, []byte(propValueString), msg.SenderID)
			tv.CreatedMSec = msg.CreatedMSec

			// in case events arrive out of order, only update if the msg is newer
			existingLatest := thingCache.Get(propName)
			if existingLatest == nil || tv.CreatedMSec > existingLatest.CreatedMSec {
				thingCache.Set(propName, tv)
			}
		}
		svc.changedThings[msg.ThingID] = true
	} else if msg.Key == vocab.EventTypeTD {
		// TD documents are handled by the directory
	} else {
		// Thing events
		// in case events arrive out of order, only update if the msg is newer
		existingLatest := thingCache.Get(msg.Key)
		if existingLatest == nil || msg.CreatedMSec > existingLatest.CreatedMSec {
			thingCache.Set(msg.Key, msg)
			svc.changedThings[msg.ThingID] = true
		}
	}
	return nil
}

// LoadProps loads the cached value of a Thing properties on demand.
// To be invoked before reading and writing Thing properties to ensure the cache is loaded.
// This immediately returns if a record for the Thing was already loaded.
// Returns true if a cache value exists, false if the thingID was added to the cache
func (svc *ValueService) LoadProps(thingID string) (cached bool) {
	svc.cacheMux.Lock()
	props, found := svc.cache[thingID]
	defer svc.cacheMux.Unlock()

	if found {
		return true
	}
	val, _ := svc.bucket.Get(thingID)

	if val == nil {
		// create a new record with things properties
		props = things.NewThingMessageMap()
	} else {
		// decode the record with things properties
		err := json.Unmarshal(val, &props)
		if err != nil {
			slog.Error("stored 'latest' properties of things can't be unmarshalled. Clean start.",
				slog.String("thingID", thingID), slog.String("err", err.Error()))
			props = things.NewThingMessageMap()
		}
	}
	svc.cache[thingID] = props
	return false
}

// SaveChanges writes modified cached properties to the underlying store.
// this returns the last encountered error, although writing is attempted for all changes
func (svc *ValueService) SaveChanges() (err error) {

	// try to minimize the lock time for each Thing
	// start with using a read lock to collect the IDs of Things that changed
	var changedThings = make([]string, 0)
	svc.cacheMux.RLock()
	for thingID, hasChanged := range svc.changedThings {
		if hasChanged {
			changedThings = append(changedThings, thingID)
		}
	}
	svc.cacheMux.RUnlock()

	// next, iterate the changes and lock only to serialize the properties record
	for _, thingID := range changedThings {
		var propsJSON []byte
		// lock only for marshalling the properties
		svc.cacheMux.Lock()
		props, found := svc.cache[thingID]
		if !found {
			// Should never happen
			err = fmt.Errorf("thingsChanged is set for thingID '%s' but no properties are present. Ignored", thingID)
			slog.Error(err.Error())
		} else {
			propsJSON, _ = json.Marshal(props)
		}
		svc.changedThings[thingID] = false
		svc.cacheMux.Unlock()

		// buckets manage their own locks
		if propsJSON != nil {
			err = svc.bucket.Set(thingID, propsJSON)
		}
	}
	return err
}

// Start the value store
func (svc *ValueService) Start() (err error) {
	slog.Info("Starting ValueService")
	// start with empty cache
	svc.cache = make(map[string]things.ThingMessageMap)
	svc.changedThings = make(map[string]bool)

	return err
}

// Stop the value store
func (svc *ValueService) Stop() {
	slog.Info("Stopping ValueService")
	_ = svc.SaveChanges()
}

// NewThingValueService creates a new instance of the storage for Thing's latest property values
func NewThingValueService(cfg *ValueStoreConfig, store buckets.IBucketStore) *ValueService {
	propsbucket := store.GetBucket(PropertiesBucketName)

	svc := &ValueService{
		cfg:           cfg,
		store:         store,
		bucket:        propsbucket,
		cache:         make(map[string]things.ThingMessageMap),
		cacheMux:      sync.RWMutex{},
		changedThings: make(map[string]bool),
	}
	return svc
}
