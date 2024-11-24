package service

import (
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/utils"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"strconv"
	"time"
)

const DefaultMaxMessageSize = 30

// AddHistory adds events and actions of any Thing
type AddHistory struct {
	// store with a bucket for each Thing
	store buckets.IBucketStore
	// onAddedValue is a callback to invoke after a value is added. Intended for tracking most recent values.
	//onAddedValue func(ev *transports.IConsumer)
	//
	retentionMgr *ManageHistory
	// Maximum message size in bytes.
	MaxMessageSize int
}

// encode a ThingMessage into a single storage key value pair for easy storage and filtering.
// Encoding generates a key as: timestampMsec/name/a|e|p/sender,
// where a|e|p indicates message type "action", "event" or "property"
func (svc *AddHistory) encodeValue(msg *transports.ThingMessage) (storageKey string, data []byte) {
	var err error
	createdTime := time.Now()
	if msg.Created != "" {
		createdTime, err = dateparse.ParseAny(msg.Created)
		if err != nil {
			slog.Warn("Invalid Created time. Using current time instead", "created", msg.Created)
			createdTime = time.Now()
		}
	}

	// the index uses milliseconds for timestamp
	timestamp := createdTime.UnixMilli()
	storageKey = strconv.FormatInt(timestamp, 10) + "/" + msg.Name
	if msg.Operation == vocab.OpInvokeAction {
		// TODO: actions subscriptions are currently not supported. This would be useful though.
		storageKey = storageKey + "/a"
	} else if msg.Operation == vocab.HTOpUpdateProperty {
		storageKey = storageKey + "/p"
	} else {
		storageKey = storageKey + "/e"
	}
	storageKey = storageKey + "/" + msg.SenderID
	//if msg.Data != nil {
	data, _ = jsoniter.Marshal(msg.Data)
	//}
	return storageKey, data
}

// AddAction adds a Thing action with the given name and value to the action history
func (svc *AddHistory) AddAction(actionValue *transports.ThingMessage) error {
	slog.Info("AddAction",
		slog.String("senderID", actionValue.SenderID),
		slog.String("thingID", actionValue.ThingID),
		slog.String("name", actionValue.Name))
	// actions are always aimed at the digital twin ID
	dThingID := actionValue.ThingID

	retain, err := svc.validateValue(actionValue)
	if err != nil {
		slog.Info("AddAction value error", "err", err.Error())
		return err
	}
	if !retain {
		slog.Info("action value not retained",
			slog.String("name", actionValue.Name))
		return nil
	}
	storageKey, val := svc.encodeValue(actionValue)
	bucket := svc.store.GetBucket(dThingID)
	err = bucket.Set(storageKey, val)
	_ = bucket.Close()
	//if svc.onAddedValue != nil {
	//	svc.onAddedValue(actionValue)
	//}
	return err
}

// AddEvent adds an event to the event history
// Only events that pass retention rules are stored.
// If the event has no created time, it will be set to 'now'
// These events must contain the digitwin thingID
func (svc *AddHistory) AddEvent(msg *transports.ThingMessage) error {

	retain, err := svc.validateValue(msg)
	if err != nil {
		slog.Warn("invalid event", "name", msg.Name, "err", err)
		return err
	}
	if !retain {
		slog.Debug("event value not retained", slog.String("name", msg.Name))
		return nil
	}

	storageKey, data := svc.encodeValue(msg)
	if len(data) > svc.MaxMessageSize {
		data = data[:svc.MaxMessageSize]
	}

	slog.Debug("AddEvent",
		slog.String("senderID", msg.SenderID),
		slog.String("thingID", msg.ThingID),
		slog.String("name", msg.Name),
		slog.Any("data", data),
		slog.String("storageKey", storageKey))

	thingAddr := msg.ThingID // the digitwin ID with the agent prefix
	bucket := svc.store.GetBucket(thingAddr)

	err = bucket.Set(storageKey, data)
	if err != nil {
		slog.Error("AddEvent storage error", "err", err)
	}
	_ = bucket.Close()
	//if svc.onAddedValue != nil {
	//	svc.onAddedValue(msg)
	//}
	return err
}

// AddMessage adds an event, action or property message to the history store
func (svc *AddHistory) AddMessage(msg *transports.ThingMessage) error {
	if msg.Operation == vocab.OpInvokeAction {
		return svc.AddAction(msg)
	}
	if msg.Operation == vocab.HTOpUpdateProperty || msg.Operation == vocab.HTOpUpdateProperties {
		return svc.AddProperty(msg)
	}
	if msg.Operation == vocab.HTOpPublishEvent {
		return svc.AddEvent(msg)
	}
	// anything else is added as an event
	return svc.AddEvent(msg)
}

// AddMessages provides a bulk-add of event/action messages to the history
// Events that are invalid are skipped.
func (svc *AddHistory) AddMessages(msgList []*transports.ThingMessage) (err error) {
	if msgList == nil || len(msgList) == 0 {
		return nil
	} else if len(msgList) == 1 {
		err = svc.AddMessage(msgList[0])
		return err
	}
	// encode events as K,V pair and group them by thingAddr
	kvpairsByThingAddr := make(map[string]map[string][]byte)
	for _, eventValue := range msgList {
		// kvpairs hold a map of storage encoded value name and value
		thingAddr := eventValue.ThingID
		kvpairs, found := kvpairsByThingAddr[thingAddr]
		if !found {
			kvpairs = make(map[string][]byte, 0)
			kvpairsByThingAddr[thingAddr] = kvpairs
		}
		retain, err := svc.validateValue(eventValue)
		if err != nil {
			slog.Warn("AddMessages: Invalid event value", slog.String("name", eventValue.Name))
			return err
		}
		if retain {
			key, value := svc.encodeValue(eventValue)
			kvpairs[key] = []byte(value)
			// notify owner to update things properties
			//if svc.onAddedValue != nil {
			//	svc.onAddedValue(eventValue)
			//}
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

// AddProperty adds a single property value to the history
//
// If property name is empty then expect a property key-value map.
// This splits the property map and adds then as individual name-value pairs
func (svc *AddHistory) AddProperty(msg *transports.ThingMessage) (err error) {

	propMap := make(map[string]any)
	if msg.Name == "" {
		err = utils.DecodeAsObject(msg.Data, &propMap)
		if err != nil {
			return err
		}
	} else {
		propMap[msg.Name] = msg.Data
	}
	if msg.Created == "" {
		msg.Created = time.Now().Format(utils.RFC3339Milli)
	}
	thingAddr := msg.ThingID // the digitwin ID with the agent prefix
	bucket := svc.store.GetBucket(thingAddr)

	// turn each property into a ThingMessage object so they can be queried separately
	for propName, propValue := range propMap {
		tv := clients.NewThingMessage(vocab.HTOpUpdateProperty,
			msg.ThingID, propName, propValue, msg.SenderID)
		tv.Created = msg.Created

		retain, err := svc.validateValue(tv)
		if err != nil {
			slog.Info("AddProperty value error", "err", err.Error())
			return err
		}
		// only store properties marked as retained. (default all)
		if retain {
			//
			storageKey, val := svc.encodeValue(tv)
			err = bucket.Set(storageKey, val)
		}
	}
	_ = bucket.Close()
	return err
}

// validateValue checks the event has the right things address, adds a timestamp if missing and returns if it is retained
// an error will be returned if the agentID, thingID or name are empty.
// retained returns true if the value is valid and passes the retention rules
func (svc *AddHistory) validateValue(tv *transports.ThingMessage) (retained bool, err error) {
	if tv.ThingID == "" {
		return false, fmt.Errorf("missing thingID in value with value name '%s'", tv.Name)
	}
	if tv.Name == "" {
		return false, fmt.Errorf("missing name for event or action for things '%s'", tv.ThingID)
	}
	if tv.SenderID == "" && tv.Operation == vocab.OpInvokeAction {
		return false, fmt.Errorf("missing sender for action on thing '%s'", tv.ThingID)
	}
	if tv.Created == "" {
		tv.Created = time.Now().Format(utils.RFC3339Milli)
	}
	if svc.retentionMgr != nil {
		retain, rule := svc.retentionMgr._IsRetained(tv)
		if rule == nil {
			slog.Debug("no retention rule found for event",
				slog.String("name", tv.Name), slog.Bool("retain", retain))
		}
		return retain, nil
	}

	return true, nil
}

// NewAddHistory provides the capability to add values to Thing history buckets
//
//	store with a bucket for each Thing
//	retentionMgr is optional and used to apply constraints to the events to add
func NewAddHistory(
	store buckets.IBucketStore, retentionMgr *ManageHistory) *AddHistory {
	svc := &AddHistory{
		store:          store,
		retentionMgr:   retentionMgr,
		MaxMessageSize: DefaultMaxMessageSize,
	}

	return svc
}
