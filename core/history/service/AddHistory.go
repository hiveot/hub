package service

import (
	"context"
	"fmt"
	"github.com/hiveot/hub/lib/vocab"
	"strconv"
	"time"

	"github.com/araddon/dateparse"
	"github.com/sirupsen/logrus"

	"github.com/hiveot/hub/lib/thing"
	"github.com/hiveot/hub/pkg/bucketstore"
)

// AddHistory adds events and actions of any Thing
// this is not restricted to one Thing and only intended for services that are authorized to do so.
type AddHistory struct {
	clientID string
	// store with buckets for Things
	store bucketstore.IBucketStore
	// onAddedValue is a callback to invoke after a value is added. Intended for tracking most recent values.
	onAddedValue func(ev thing.ThingValue, isAction bool)
	//
	retentionMgr *ManageRetention
}

// encode a ThingValue into a single key value pair
// Encoding generates a key as: timestampMsec/name/a|e, where a|e indicates action or event
func (svc *AddHistory) encodeValue(thingValue thing.ThingValue, isAction bool) (key string, val []byte) {
	var err error
	ts := time.Now()
	if thingValue.Created != "" {
		ts, err = dateparse.ParseAny(thingValue.Created)
		if err != nil {
			logrus.Infof("Invalid Created time '%s'. Using current time instead", thingValue.Created)
			ts = time.Now()
		}
	}

	// the index uses milliseconds for timestamp
	timestamp := ts.UnixMilli()
	key = strconv.FormatInt(timestamp, 10) + "/" + thingValue.ID
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
func (svc *AddHistory) AddAction(_ context.Context, actionValue thing.ThingValue) error {
	logrus.Infof("clientID=%s, thingID=%s, name=%s", svc.clientID, actionValue.ThingID, actionValue.ID)

	if err := svc.validateValue(actionValue); err != nil {
		logrus.Info(err)
		return err
	}
	key, val := svc.encodeValue(actionValue, true)
	thingAddr := actionValue.PublisherID + "/" + actionValue.ThingID
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
func (svc *AddHistory) AddEvent(ctx context.Context, eventValue thing.ThingValue) error {

	valueStr := eventValue.Data
	if len(valueStr) > 20 {
		valueStr = valueStr[:20]
	}
	logrus.Infof("clientID=%s, thingID=%s, name=%s, value=%s", svc.clientID, eventValue.ThingID, eventValue.ID, valueStr)
	if err := svc.validateValue(eventValue); err != nil {
		logrus.Info(err)
		return err
	}

	key, val := svc.encodeValue(eventValue, false)
	thingAddr := eventValue.PublisherID + "/" + eventValue.ThingID
	bucket := svc.store.GetBucket(thingAddr)

	logrus.Infof("adding: [%s %s] %s", eventValue.PublisherID, eventValue.ThingID, key)

	err := bucket.Set(key, val)
	_ = bucket.Close()
	if svc.onAddedValue != nil {
		svc.onAddedValue(eventValue, false)
	}
	return err
}

// AddEvents provides a bulk-add of events to the event history
// Events that are invalid are skipped.
func (svc *AddHistory) AddEvents(ctx context.Context, eventValues []thing.ThingValue) (err error) {
	logrus.Infof("clientID=%s, nrEvents=%d", svc.clientID, len(eventValues))
	if eventValues == nil || len(eventValues) == 0 {
		return nil
	} else if len(eventValues) == 1 {
		err = svc.AddEvent(ctx, eventValues[0])
		return err
	}
	// encode events as K,V pair and group them by thingAddr
	kvpairsByThingAddr := make(map[string]map[string][]byte)
	for _, eventValue := range eventValues {
		// kvpairs hold a map of storage encoded value key and value
		thingAddr := eventValue.PublisherID + "/" + eventValue.ThingID
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

// Release the capability and its resources
func (svc *AddHistory) Release() {

}

// validateValue checks the event has the right thing address and adds a timestamp if missing
func (svc *AddHistory) validateValue(thingValue thing.ThingValue) error {
	if thingValue.ThingID == "" || thingValue.PublisherID == "" {
		return fmt.Errorf("missing publisher/thing address in value with name '%s'", thingValue.ID)
	}
	if thingValue.ID == "" {
		return fmt.Errorf("missing name for event or action for thing '%s/%s'", thingValue.PublisherID, thingValue.ThingID)
	}
	if thingValue.Created == "" {
		thingValue.Created = time.Now().Format(vocab.ISO8601Format)
	}
	if svc.retentionMgr != nil {
		isValid, err := svc.retentionMgr.TestEvent(context.Background(), thingValue)
		if !isValid || err != nil {
			return fmt.Errorf("no retention for event '%s'", thingValue.ID)
		}
	}

	return nil
}

// NewAddHistory provides the capability to add values to Thing history buckets
//
//	retentionMgr is optional and used to apply constraints to the events to add
//	onAddedValue is optional and invoked after the value is added to the bucket.
func NewAddHistory(
	clientID string,
	store bucketstore.IBucketStore,
	retentionMgr *ManageRetention,
	onAddedValue func(value thing.ThingValue, isAction bool)) *AddHistory {
	svc := &AddHistory{
		clientID:     clientID,
		store:        store,
		retentionMgr: retentionMgr,
		onAddedValue: onAddedValue,
	}

	return svc
}
