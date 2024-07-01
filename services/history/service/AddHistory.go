package service

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/buckets"
	"log/slog"
	"strconv"
	"time"

	"github.com/hiveot/hub/lib/things"
)

const DefaultMaxMessageSize = 30

// AddHistory adds events and actions of any Thing
type AddHistory struct {
	// store with a bucket for each Thing
	store buckets.IBucketStore
	// onAddedValue is a callback to invoke after a value is added. Intended for tracking most recent values.
	onAddedValue func(ev *things.ThingMessage)
	//
	retentionMgr *ManageHistory
	// Maximum message size in bytes.
	MaxMessageSize int
}

// encode a ThingMessage into a single key value pair for easy storage and filtering.
// Encoding generates a key as: timestampMsec/name/a|e|p/sender,
// where a|e|p indicates message type "action", "event" or "property"
func (svc *AddHistory) encodeValue(msg *things.ThingMessage) (key string, data []byte) {
	var err error
	ts := time.Now()
	if msg.CreatedMSec > 0 {
		ts = time.UnixMilli(msg.CreatedMSec)
		if err != nil {
			slog.Warn("Invalid CreatedMSec time. Using current time instead", "created", msg.CreatedMSec)
			ts = time.Now()
		}
	}

	// the index uses milliseconds for timestamp
	timestamp := ts.UnixMilli()
	key = strconv.FormatInt(timestamp, 10) + "/" + msg.Key
	if msg.MessageType == vocab.MessageTypeAction {
		key = key + "/a"
	} else if msg.MessageType == vocab.MessageTypeProperty {
		key = key + "/p"
	} else {
		key = key + "/e"
	}
	key = key + "/" + msg.SenderID
	data, _ = json.Marshal(msg.Data)
	return key, data
}

// AddAction adds a Thing action with the given name and value to the action history
// value is json encoded. Optionally include a 'created' ISO8601 timestamp
func (svc *AddHistory) AddAction(actionValue *things.ThingMessage) error {
	slog.Info("AddAction",
		slog.String("senderID", actionValue.SenderID),
		slog.String("thingID", actionValue.ThingID),
		slog.String("key", actionValue.Key))
	// actions are always aimed at the digital twin ID
	dThingID := actionValue.ThingID
	retain, err := svc.validateValue(actionValue)
	if err != nil {
		slog.Info("AddAction value error", "err", err.Error())
		return err
	}
	if !retain {
		slog.Info("action value not retained",
			slog.String("name", actionValue.Key))
		return nil
	}
	key, val := svc.encodeValue(actionValue)
	bucket := svc.store.GetBucket(dThingID)
	err = bucket.Set(key, val)
	_ = bucket.Close()
	if svc.onAddedValue != nil {
		svc.onAddedValue(actionValue)
	}
	return err
}

// AddProperties adds individual property values to the history
// This splits the property map and adds then as individual key-values
func (svc *AddHistory) AddProperties(msg *things.ThingMessage) error {
	propMap := make(map[string]any)
	err := msg.Decode(&propMap)
	if err != nil {
		return err
	}
	thingAddr := msg.ThingID // the digitwin ID with the agent prefix
	bucket := svc.store.GetBucket(thingAddr)

	// turn each property into a ThingMessage object so they can be queried separately
	for propName, propValue := range propMap {
		propValueString := fmt.Sprint(propValue)
		// store this as a property message to differentiate from events
		tv := things.NewThingMessage(vocab.MessageTypeProperty,
			msg.ThingID, propName, propValueString, msg.SenderID)
		tv.CreatedMSec = msg.CreatedMSec
		//

		storageKey, val := svc.encodeValue(msg)

		err = bucket.Set(storageKey, val)
	}
	_ = bucket.Close()
	return err
}

// AddEvent adds an event to the event history
// Only events that pass retention rules are stored.
// If the event has no created time, it will be set to 'now'
// These events must contain the digitwin thingID
func (svc *AddHistory) AddEvent(msg *things.ThingMessage) error {

	if msg.Key == vocab.EventTypeProperties {
		return svc.AddProperties(msg)
	}

	retain, err := svc.validateValue(msg)
	if err != nil {
		slog.Warn("invalid event", "name", msg.Key, "err", err)
		return err
	}
	if !retain {
		slog.Debug("event value not retained", slog.String("name", msg.Key))
		return nil
	}

	storageKey, data := svc.encodeValue(msg)
	if len(data) > svc.MaxMessageSize {
		data = data[:svc.MaxMessageSize]
	}

	slog.Debug("AddEvent",
		slog.String("senderID", msg.SenderID),
		slog.String("thingID", msg.ThingID),
		slog.String("key", msg.Key),
		slog.Any("data", data),
		slog.String("storageKey", storageKey))

	thingAddr := msg.ThingID // the digitwin ID with the agent prefix
	bucket := svc.store.GetBucket(thingAddr)

	err = bucket.Set(storageKey, data)
	if err != nil {
		slog.Error("AddMessage storage error", "err", err)
	}
	_ = bucket.Close()
	if svc.onAddedValue != nil {
		svc.onAddedValue(msg)
	}
	return err
}

// AddMessage adds an event, action or properties to the history store
func (svc *AddHistory) AddMessage(msg *things.ThingMessage) error {
	if msg.MessageType == vocab.MessageTypeAction {
		return svc.AddAction(msg)
	}
	if msg.MessageType == vocab.MessageTypeProperty {
		return svc.AddAction(msg)
	}
	if msg.Key == vocab.EventTypeProperties {
		return svc.AddProperties(msg)
	}
	return svc.AddEvent(msg)
}

// AddMessages provides a bulk-add of event/action messages to the history
// Events that are invalid are skipped.
func (svc *AddHistory) AddMessages(msgList []*things.ThingMessage) (err error) {
	if msgList == nil || len(msgList) == 0 {
		return nil
	} else if len(msgList) == 1 {
		err = svc.AddMessage(msgList[0])
		return err
	}
	// encode events as K,V pair and group them by thingAddr
	kvpairsByThingAddr := make(map[string]map[string][]byte)
	for _, eventValue := range msgList {
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
			kvpairs[key] = []byte(value)
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
	if tv.SenderID == "" {
		return false, fmt.Errorf("missing sender for event or action for things '%s'", tv.ThingID)
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
		store:          store,
		retentionMgr:   retentionMgr,
		onAddedValue:   onAddedValue,
		MaxMessageSize: DefaultMaxMessageSize,
	}

	return svc
}
