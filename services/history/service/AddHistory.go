package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/buckets"
	"log/slog"
	"strconv"
	"time"

	"github.com/hiveot/hub/lib/things"
)

// AddHistory adds events and actions of any Thing
type AddHistory struct {
	// store with a bucket for each Thing
	store buckets.IBucketStore
	// onAddedValue is a callback to invoke after a value is added. Intended for tracking most recent values.
	onAddedValue func(ev *things.ThingMessage)
	//
	retentionMgr *ManageHistory
}

// encode a ThingMessage into a single key value pair for easy storage and filtering.
// Encoding generates a key as: timestampMsec/name/a|e|c/sender,
// where a|e|c indicates action, event or config
func (svc *AddHistory) encodeValue(tv *things.ThingMessage) (key string, val []byte) {
	var err error
	ts := time.Now()
	if tv.CreatedMSec > 0 {
		ts = time.UnixMilli(tv.CreatedMSec)
		if err != nil {
			slog.Warn("Invalid CreatedMSec time. Using current time instead", "created", tv.CreatedMSec)
			ts = time.Now()
		}
	}

	// the index uses milliseconds for timestamp
	timestamp := ts.UnixMilli()
	key = strconv.FormatInt(timestamp, 10) + "/" + tv.Key
	if tv.MessageType == vocab.MessageTypeAction {
		key = key + "/a"
	} else if tv.MessageType == MessageTypeConfig {
		key = key + "/c"
	} else {
		key = key + "/e"
	}
	key = key + "/" + tv.SenderID
	val = tv.Data
	return key, val
}

//
//// AddAction adds a Thing action with the given name and value to the action history
//// value is json encoded. Optionally include a 'created' ISO8601 timestamp
//func (svc *AddHistory) AddAction(actionValue *things.ThingMessage) error {
//	slog.Info("AddAction",
//		slog.String("agentID", actionValue.AgentID),
//		slog.String("thingID", actionValue.ThingID),
//		slog.String("actionName", actionValue.Name))
//
//	retain, err := svc.validateValue(actionValue)
//	if err != nil {
//		slog.Info("AddAction value error", "err", err.Error())
//		return err
//	}
//	if !retain {
//		slog.Info("action value not retained", slog.String("name", actionValue.Name))
//		return nil
//	}
//	key, val := svc.encodeValue(actionValue)
//	thingAddr := actionValue.AgentID + "/" + actionValue.ThingID
//	bucket := svc.store.GetBucket(thingAddr)
//	err = bucket.Set(key, val)
//	_ = bucket.Close()
//	if svc.onAddedValue != nil {
//		svc.onAddedValue(actionValue)
//	}
//	return err
//}

// AddMessage adds an event to the event history
// Only events that pass retention rules are stored.
// If the event has no created time, it will be set to 'now'
func (svc *AddHistory) AddMessage(newtm *things.ThingMessage) error {

	valueStr := newtm.Data
	if len(valueStr) > 20 {
		valueStr = valueStr[:20]
	}
	retain, err := svc.validateValue(newtm)
	if err != nil {
		slog.Warn("invalid event", "name", newtm.Key, "err", err)
		return err
	}
	if !retain {
		slog.Debug("event value not retained", slog.String("name", newtm.Key))
		return nil
	}

	storageKey, val := svc.encodeValue(newtm)

	slog.Info("AddMessage",
		slog.String("senderID", newtm.SenderID),
		slog.String("thingID", newtm.ThingID),
		slog.String("key", newtm.Key),
		slog.String("value", string(valueStr)),
		slog.String("storageKey", storageKey))

	thingAddr := newtm.ThingID // the digitwin ID with the agent prefix
	bucket := svc.store.GetBucket(thingAddr)

	err = bucket.Set(storageKey, val)
	if err != nil {
		slog.Error("AddMessage storage error", "err", err)
	}
	_ = bucket.Close()
	if svc.onAddedValue != nil {
		svc.onAddedValue(newtm)
	}
	return err
}

// AddMessages provides a bulk-add of event/action messages to the history
// Events that are invalid are skipped.
func (svc *AddHistory) AddMessages(tvList []*things.ThingMessage) (err error) {
	if tvList == nil || len(tvList) == 0 {
		return nil
	} else if len(tvList) == 1 {
		err = svc.AddMessage(tvList[0])
		return err
	}
	// encode events as K,V pair and group them by thingAddr
	kvpairsByThingAddr := make(map[string]map[string][]byte)
	for _, eventValue := range tvList {
		// kvpairs hold a map of storage encoded value key and value
		thingAddr := eventValue.ThingID
		kvpairs, found := kvpairsByThingAddr[thingAddr]
		if !found {
			kvpairs = make(map[string][]byte, 0)
			kvpairsByThingAddr[thingAddr] = kvpairs
		}
		retain, err := svc.validateValue(eventValue)
		if err != nil {
			slog.Warn("Invalid event value", slog.String("key", eventValue.Key))
			return err
		}
		if retain {
			key, value := svc.encodeValue(eventValue)
			kvpairs[key] = value
			// notify owner to update things properties
			if svc.onAddedValue != nil {
				svc.onAddedValue(eventValue)
			}
		}
	}
	// adding in bulk, opening and closing buckets only once for each things address
	for thingAddr, kvpairs := range kvpairsByThingAddr {
		bucket := svc.store.GetBucket(thingAddr)
		_ = bucket.SetMultiple(kvpairs)
		err = bucket.Close()
	}
	return nil
}

// validateValue checks the event has the right things address, adds a timestamp if missing and returns if it is retained
// an error will be returned if the agentID, thingID or name are empty.
// retained returns true if the value is valid and passes the retention rules
func (svc *AddHistory) validateValue(tv *things.ThingMessage) (retained bool, err error) {
	if tv.ThingID == "" {
		return false, fmt.Errorf("missing thingID in value with value key '%s'", tv.Key)
	}
	if tv.Key == "" {
		return false, fmt.Errorf("missing key for event or action for things '%s'", tv.ThingID)
	}
	if tv.CreatedMSec == 0 {
		tv.CreatedMSec = time.Now().UnixMilli()
	}
	if svc.retentionMgr != nil {
		retain, rule := svc.retentionMgr._IsRetained(tv)
		if rule == nil {
			slog.Debug("no retention rule found for event",
				slog.String("key", tv.Key), slog.Bool("retain", retain))
		}
		return retain, nil
	}

	return true, nil
}

// NewAddHistory provides the capability to add values to Thing history buckets
//
//	store with a bucket for each Thing
//	retentionMgr is optional and used to apply constraints to the events to add
//	onAddedValue is optional and invoked after the value is added to the bucket.
func NewAddHistory(
	store buckets.IBucketStore,
	retentionMgr *ManageHistory,
	onAddedValue func(value *things.ThingMessage)) *AddHistory {
	svc := &AddHistory{
		store:        store,
		retentionMgr: retentionMgr,
		onAddedValue: onAddedValue,
	}

	return svc
}
