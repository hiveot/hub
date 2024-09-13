package service

import (
	"encoding/json"
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"sync"
)

// DigiTwinLatestStore is the digital twin storage for storing the current
// state of Things.
// When used by the outbox it holds the last event and property value.
// This store is intended for obtaining the current state of things.
type DigiTwinLatestStore struct {
	// The message storage bucket
	bucket buckets.IBucket

	// in-memory cache of the latest messages by digitwin thingID
	cache map[string]hubclient.ThingMessageMap
	// mutex for read/writing the cache
	cacheMux sync.RWMutex // mutex for the following two fields
	// map of thingsIDs and their change status
	changedThings map[string]bool
}

// AddEvent stores the event, sent by agent, in the bucket
//func (store *DigiTwinLatestStore) AddEvent(msg *things.ThingMessage) {
//
//	addr := fmt.Sprintf("%s", msg.SenderID, msg.ThingID, msg.MessageType, msg.Name)
//	msgJSON, _ := json.Marshal(msg)
//	err := store.bucket.Set(addr, msgJSON)
//	if err != nil {
//		slog.Error("Unexpected error storing event in outbox",
//			"thingID", msg.ThingID, "name", msg.Name, "err", err.Error())
//		return
//	}
//}

// LoadLatest loads the cached value of a Thing properties on demand.
// To be invoked before reading and writing Thing properties to ensure the cache is loaded.
// This immediately returns if a record for the Thing was already loaded.
// Returns true if a cache value exists, false if the thingID was added to the cache
func (svc *DigiTwinLatestStore) LoadLatest(thingID string) (cached bool) {
	svc.cacheMux.Lock()
	props, found := svc.cache[thingID]
	defer svc.cacheMux.Unlock()

	if found {
		return true
	}
	val, _ := svc.bucket.Get(thingID)

	if val == nil {
		// create a new record with things messages
		props = hubclient.NewThingMessageMap()
	} else {
		// decode the record with things properties
		err := json.Unmarshal(val, &props)
		if err != nil {
			slog.Error("stored 'latest' properties of things can't be unmarshalled. Clean start.",
				slog.String("thingID", thingID), slog.String("err", err.Error()))
			props = hubclient.NewThingMessageMap()
		}
	}
	svc.cache[thingID] = props
	return false
}

// ReadLatest returns the latest values send to digital twin Things.
//
//	msgType type of message to read, MessageTypeEvent, MessageTypeProperties
//	thingID whose events to return
//	keys  optional keys of message types to filter on
//	since optional ISO timestamp with time since which to return the messages
//
//	keys optional filter for the values to read or nil to read all
func (svc *DigiTwinLatestStore) ReadLatest(msgType string, thingID string, keys []string, since string) (
	messages hubclient.ThingMessageMap, err error) {

	messages = hubclient.NewThingMessageMap()
	svc.LoadLatest(thingID)

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	cachedMessages, found := svc.cache[thingID]
	if !found {
		return nil, fmt.Errorf("ReadLatest. Unknown thingID '%s'", thingID)
	}
	// get each specified value
	if keys != nil && len(keys) > 0 {
		// filter the requested property/event keys
		for _, name := range keys {
			tm := cachedMessages.Get(name)
			if tm != nil {
				if tm.Name == "" {
					slog.Error("ReadLatest. TM has no message name")
				}
				messages.Set(name, tm)
			}
		}
	} else {
		// filter by message type
		for k, v := range cachedMessages {
			if v.MessageType == msgType {
				messages[k] = v
			}
		}
	}
	// filter on since
	if since != "" {
		sinceTime, err := dateparse.ParseAny(since)
		if err != nil {
			return nil, fmt.Errorf(
				"ReadLatest: invalid since time '%s' for thingID '%s'", since, thingID)
		}
		validMessages := hubclient.NewThingMessageMap()
		for k, v := range messages {
			createdTime, _ := dateparse.ParseAny(v.Created)
			if sinceTime.UnixMilli() <= createdTime.UnixMilli() {
				validMessages.Set(k, v)
			}
		}
		messages = validMessages
	}
	return messages, nil
}

// Remove removes a value from the latest values send to digital twin Things.
//
//	messageID to remove
func (svc *DigiTwinLatestStore) Remove(thingID string, key string) (err error) {
	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()
	thingCache, _ := svc.cache[thingID]
	if thingCache != nil {
		delete(thingCache, key)
	}
	return err
}

// SaveChanges writes modified cached messages to the underlying store.
// this returns the last encountered error, although writing is attempted for all changes
func (svc *DigiTwinLatestStore) SaveChanges() (err error) {

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
func (svc *DigiTwinLatestStore) Start() error {
	return nil
}
func (svc *DigiTwinLatestStore) Stop() {
	_ = svc.SaveChanges()
	_ = svc.bucket.Close()
}

// StoreMessage stores the latest event or property values
func (svc *DigiTwinLatestStore) StoreMessage(msg *hubclient.ThingMessage) {

	svc.LoadLatest(msg.ThingID)
	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()
	if msg.Name == "" {
		slog.Error("StoreMessage. msg has no Name")
	}
	thingCache, _ := svc.cache[msg.ThingID]
	if msg.MessageType == vocab.MessageTypeEvent {
		if msg.Name == vocab.EventNameProperties {
			// the value holds a map of property name:value pairs, add each one individually
			// in order to retain the sender and created timestamp.
			props := make(map[string]any)
			err := utils.DecodeAsObject(msg.Data, &props)
			if err != nil {
				slog.Warn("StoreEvent; Error unmarshalling props. Ignored.",
					slog.String("err", err.Error()),
					slog.String("senderID", msg.SenderID),
					slog.String("thingID", msg.ThingID))
				return
			}
			// turn each value into a ThingMessage object
			for propName, propValue := range props {
				//propValueString := fmt.Sprint(propValue)
				tm := hubclient.NewThingMessage(vocab.MessageTypeEvent,
					msg.ThingID, propName, propValue, msg.SenderID)
				tm.Created = msg.Created

				// in case events arrive out of order, only update if the msg is newer
				existingLatest := thingCache.Get(propName)
				if existingLatest == nil || tm.Created > existingLatest.Created {
					thingCache.Set(propName, tm)
				}
			}
			svc.changedThings[msg.ThingID] = true
		} else if msg.Name == vocab.EventNameTD {
			// TD documents are handled by the directory
		} else {
			// Thing events
			// in case events arrive out of order, only update if the msg is newer
			existingLatest := thingCache.Get(msg.Name)
			if existingLatest == nil || msg.Created > existingLatest.Created {
				thingCache.Set(msg.Name, msg)
				svc.changedThings[msg.ThingID] = true
			}
		}
	} else {
		// TODO: split namespace as currently events and properties share the same namespace
		// in case messages arrive out of order, only update if the msg is newer
		existingLatest := thingCache.Get(msg.Name)
		if existingLatest == nil || msg.Created > existingLatest.Created {
			thingCache.Set(msg.Name, msg)
			svc.changedThings[msg.ThingID] = true
		}
	}
}

// NewDigiTwinLatestStore returns a new instance of the latest-messages store using the
// given storage bucket for persistence.
// the bucket will be closed on stop
func NewDigiTwinLatestStore(bucket buckets.IBucket) *DigiTwinLatestStore {
	svc := &DigiTwinLatestStore{
		bucket:        bucket,
		cache:         make(map[string]hubclient.ThingMessageMap),
		changedThings: make(map[string]bool),
	}
	return svc
}
