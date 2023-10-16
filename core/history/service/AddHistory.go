package service

import (
	"fmt"
	"github.com/hiveot/hub/lib/buckets"
	"log/slog"
	"strconv"
	"time"

	"github.com/hiveot/hub/lib/thing"
)

// AddHistory adds events and actions of any Thing
type AddHistory struct {
	// store with a bucket for each Thing
	store buckets.IBucketStore
	// onAddedValue is a callback to invoke after a value is added. Intended for tracking most recent values.
	onAddedValue func(ev *thing.ThingValue, isAction bool)
	//
	retentionMgr *HistoryRetention
}

// encode a ThingValue into a single key value pair
// Encoding generates a key as: timestampMsec/name/a|e, where a|e indicates action or event
func (svc *AddHistory) encodeValue(thingValue *thing.ThingValue, isAction bool) (key string, val []byte) {
	var err error
	ts := time.Now()
	if thingValue.CreatedMSec > 0 {
		ts = time.UnixMilli(thingValue.CreatedMSec)
		if err != nil {
			slog.Warn("Invalid CreatedMSec time. Using current time instead", "created", thingValue.CreatedMSec)
			ts = time.Now()
		}
	}

	// the index uses milliseconds for timestamp
	timestamp := ts.UnixMilli()
	key = strconv.FormatInt(timestamp, 10) + "/" + thingValue.Name
	if isAction {
		key = key + "/a"
	} else {
		key = key + "/e"
	}
	// TODO: reorganize data to store. Remove duplication. Timestamp in msec since epoc
	val = thingValue.Data
	return key, val
}

// AddAction adds a Thing action with the given name and value to the action history
// value is json encoded. Optionally include a 'created' ISO8601 timestamp
func (svc *AddHistory) AddAction(actionValue *thing.ThingValue) error {
	slog.Info("AddAction",
		slog.String("agentID", actionValue.AgentID),
		slog.String("thingID", actionValue.ThingID),
		slog.String("actionName", actionValue.Name))

	if err := svc.validateValue(actionValue); err != nil {
		slog.Info("AddAction error", "err", err.Error())
		return err
	}
	key, val := svc.encodeValue(actionValue, true)
	thingAddr := actionValue.AgentID + "/" + actionValue.ThingID
	bucket := svc.store.GetBucket(thingAddr)
	err := bucket.Set(key, val)
	_ = bucket.Close()
	if svc.onAddedValue != nil {
		svc.onAddedValue(actionValue, true)
	}
	return err
}

// AddEvent adds an event to the event history
// If the event has no created time, it will be set to 'now'
func (svc *AddHistory) AddEvent(eventMsg *thing.ThingValue) error {

	valueStr := eventMsg.Data
	if len(valueStr) > 20 {
		valueStr = valueStr[:20]
	}
	slog.Info("AddEvent",
		slog.String("agentID", eventMsg.AgentID),
		slog.String("thingID", eventMsg.ThingID),
		slog.String("name", eventMsg.Name),
		slog.String("value", string(valueStr)))

	if err := svc.validateValue(eventMsg); err != nil {
		slog.Warn("invalid value", "err", err)
		return err
	}

	key, val := svc.encodeValue(eventMsg, false)
	thingAddr := eventMsg.AgentID + "/" + eventMsg.ThingID
	bucket := svc.store.GetBucket(thingAddr)

	err := bucket.Set(key, val)
	_ = bucket.Close()
	if svc.onAddedValue != nil {
		svc.onAddedValue(eventMsg, false)
	}
	return err
}

// AddEvents provides a bulk-add of events to the event history
// Events that are invalid are skipped.
func (svc *AddHistory) AddEvents(eventValues []*thing.ThingValue) (err error) {
	if eventValues == nil || len(eventValues) == 0 {
		return nil
	} else if len(eventValues) == 1 {
		err = svc.AddEvent(eventValues[0])
		return err
	}
	// encode events as K,V pair and group them by thingAddr
	kvpairsByThingAddr := make(map[string]map[string][]byte)
	for _, eventValue := range eventValues {
		// kvpairs hold a map of storage encoded value key and value
		thingAddr := eventValue.AgentID + "/" + eventValue.ThingID
		kvpairs, found := kvpairsByThingAddr[thingAddr]
		if !found {
			kvpairs = make(map[string][]byte, 0)
			kvpairsByThingAddr[thingAddr] = kvpairs
		}
		if err := svc.validateValue(eventValue); err == nil {
			key, value := svc.encodeValue(eventValue, false)
			kvpairs[key] = value
			// notify owner to update thing properties
			if svc.onAddedValue != nil {
				svc.onAddedValue(eventValue, false)
			}
		}
	}
	// adding in bulk, opening and closing buckets only once for each thing address
	for thingAddr, kvpairs := range kvpairsByThingAddr {
		bucket := svc.store.GetBucket(thingAddr)
		_ = bucket.SetMultiple(kvpairs)
		err = bucket.Close()
	}
	return nil
}

// validateValue checks the event has the right thing address and adds a timestamp if missing
func (svc *AddHistory) validateValue(evMsg *thing.ThingValue) error {
	if evMsg.ThingID == "" || evMsg.AgentID == "" {
		return fmt.Errorf("missing agent/thing address in value with name '%s'", evMsg.Name)
	}
	if evMsg.Name == "" {
		return fmt.Errorf("missing name for event or action for thing '%s/%s'", evMsg.AgentID, evMsg.ThingID)
	}
	if evMsg.CreatedMSec == 0 {
		evMsg.CreatedMSec = time.Now().UnixMilli()
	}
	if svc.retentionMgr != nil {
		isValid, err := svc.retentionMgr.CheckRetention(evMsg)
		if !isValid || err != nil {
			return fmt.Errorf("no retention for event '%s'", evMsg.Name)
		}
	}

	return nil
}

// NewAddHistory provides the capability to add values to Thing history buckets
//
//	store with a bucket for each Thing
//	retentionMgr is optional and used to apply constraints to the events to add
//	onAddedValue is optional and invoked after the value is added to the bucket.
func NewAddHistory(
	store buckets.IBucketStore,
	retentionMgr *HistoryRetention,
	onAddedValue func(value *thing.ThingValue, isAction bool)) *AddHistory {
	svc := &AddHistory{
		store:        store,
		retentionMgr: retentionMgr,
		onAddedValue: onAddedValue,
	}

	return svc
}
