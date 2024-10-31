package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"sync"
	"time"
)

type DigitwinStore struct {
	// The digital twin storage bucket
	dtwBucket buckets.IBucket

	// in-memory cache of the digital twin Things by dThingID
	dtwCache map[string]*DigitalTwinInstance

	// map of changed digital thing IDs values
	changedThings map[string]any

	// list of DTW thingIDs used as a consistent iterator for reading batches
	thingKeys []string

	// mutex for read/writing the cache
	cacheMux sync.RWMutex // mutex for the following two fields

	// strict forces event and property checks against the TD
	strict bool
}

// Close the digitwin store
func (svc *DigitwinStore) Close() {
	slog.Info("Closing DigitwinStore")
	_ = svc.SaveChanges()
	if svc.dtwBucket != nil {
		_ = svc.dtwBucket.Close()
		svc.dtwBucket = nil
	}
}

//

// LoadCacheFromStore saves the current changes and reloads the cache from
// store into memory.
func (store *DigitwinStore) LoadCacheFromStore() error {
	_ = store.SaveChanges()

	store.cacheMux.Lock()
	defer store.cacheMux.Unlock()
	cursor, err := store.dtwBucket.Cursor()
	if err != nil {
		return err
	}
	store.thingKeys = make([]string, 0, 1000)
	store.dtwCache = make(map[string]*DigitalTwinInstance)
	store.changedThings = make(map[string]any)
	for {
		// read in batches of 300 TD documents
		tdmap, itemsRemaining := cursor.NextN(300)
		for dThingID, dtwSer := range tdmap {
			dtwInstance := &DigitalTwinInstance{}
			err = jsoniter.Unmarshal(dtwSer, &dtwInstance)
			//err = msgpack.Unmarshal(dtwSer, &dtwInstance)
			if err == nil {
				store.thingKeys = append(store.thingKeys, dThingID)
				store.dtwCache[dThingID] = dtwInstance
			}
		}
		if !itemsRemaining {
			break
		}
	}
	return nil
}

// QueryAction returns the current status of the action
// This returns an empty value if no action value is available
func (svc *DigitwinStore) QueryAction(
	dThingID string, name string) (v digitwin.ActionValue, err error) {

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err = fmt.Errorf("ReadAction: dThing with ID '%s' not found", dThingID)
		return v, err
	}
	if svc.strict {
		// affordance must exist
		aff, found := dtw.DtwTD.Actions[name]
		_ = aff
		if !found {
			return v, fmt.Errorf("QueryAction: Action '%s' not found in digital twin '%s'", name, dThingID)
		}
	}
	v, found = dtw.ActionValues[name]
	return v, nil
}

// QueryAllActions returns all last known action invocation status of the given thing
func (svc *DigitwinStore) QueryAllActions(dThingID string) (
	v map[string]digitwin.ActionValue, err error) {

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err = fmt.Errorf("QueryAllActions: dThing with ID '%s' not found", dThingID)
		return v, err
	}
	return dtw.ActionValues, err
}

// ReadAllEvents returns all last known action invocation status of the given thing
func (svc *DigitwinStore) ReadAllEvents(dThingID string) (
	v map[string]digitwin.ThingValue, err error) {

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err = fmt.Errorf("ReadAllEvents: dThing with ID '%s' not found", dThingID)
		return v, err
	}
	return dtw.EventValues, err
}

// ReadAllProperties returns all last known property values of the given thing
func (svc *DigitwinStore) ReadAllProperties(dThingID string) (
	v map[string]digitwin.ThingValue, err error) {

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err = fmt.Errorf("ReadAllProperties: dThing with ID '%s' not found", dThingID)
		return v, err
	}
	return dtw.PropValues, err
}

// ReadEvent returns the last known event status of the given name
// this returns an empty value if no last known event is found
// If 'strict' is set then this fails with an error if the event is not in the TD.
func (svc *DigitwinStore) ReadEvent(
	dThingID string, name string) (v digitwin.ThingValue, err error) {

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err = fmt.Errorf("ReadEvent: dThing with ID '%s' not found", dThingID)
		return v, err
	}
	if svc.strict {
		// affordance must exist
		aff, found := dtw.DtwTD.Events[name]
		_ = aff
		if !found {
			return v, fmt.Errorf("ReadEvent: Event '%s' not found in digital twin '%s'", name, dThingID)
		}
	}
	// event value might not exist
	v, found = dtw.EventValues[name]
	return v, nil
}

// ReadProperty returns the last known property value of the given name,
// or an empty value if no value is known.
// This returns an error if the dThingID doesn't exist.
func (svc *DigitwinStore) ReadProperty(
	dThingID string, name string) (v digitwin.ThingValue, err error) {

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err = fmt.Errorf("ReadProperty: dThing with ID '%s' not found", dThingID)
		return v, err
	}
	if svc.strict {
		// affordance must exist
		aff, found := dtw.DtwTD.Properties[name]
		_ = aff
		if !found {
			return v, fmt.Errorf("ReadProperty: Property '%s' not found in digital twin '%s'", name, dThingID)
		}
	}
	// value might not exist is optional
	v, found = dtw.PropValues[name]
	return v, nil
}

// ReadDTW returns the Digitwin instance.
//func (svc *DigitwinStore) ReadDTW(
//	dThingID string) (resp digitwin.DigitalTwinInstance, err error) {
//
//	svc.cacheMux.RLock()
//	defer svc.cacheMux.RUnlock()
//	dtw, found := svc.dtwCache[dThingID]
//	if !found {
//		err = fmt.Errorf("dThing with ID '%s' not found", dThingID)
//		return resp, err
//	}
//	return *dtw, err
//}

// ReadDThing returns the Digitwin TD document in the store.
func (svc *DigitwinStore) ReadDThing(dThingID string) (dtd *tdd.TD, err error) {

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err = fmt.Errorf("dThing with ID '%s' not found", dThingID)
		return dtd, err
	}
	return dtw.DtwTD, err
}

// ReadTDs returns a list of digital twin TDs
//
// limit is the maximum number of records to return
// offset is the offset of the first record to return
func (svc *DigitwinStore) ReadTDs(offset int, limit int) (resp []*tdd.TD, err error) {

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	// Use the thingKeys index to ensure consistent iteration and to quickly
	// skip offset items (maps are not consistent between iterations)
	if offset >= len(svc.thingKeys) {
		// empty result
		return resp, nil
	}
	if offset+limit > len(svc.thingKeys) {
		limit = len(svc.thingKeys) - offset
	}
	tdKeys := svc.thingKeys[offset:limit]
	resp = make([]*tdd.TD, 0, len(tdKeys))

	// add the documents
	for _, k := range tdKeys {
		dtw := svc.dtwCache[k]
		resp = append(resp, dtw.DtwTD)
	}
	return resp, nil
}

// RemoveDTW deletes the digitwin instance of an agent with the given ThingID
func (svc *DigitwinStore) RemoveDTW(dThingID string, senderID string) error {
	// TBD: should we mark this as deleted instead? retain historical things?
	slog.Debug("RemoveTD",
		slog.String("dThingID", dThingID),
		slog.String("senderID", senderID))

	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	// remove from both cache and bucket
	err := svc.dtwBucket.Delete(dThingID)
	delete(svc.dtwCache, dThingID)
	delete(svc.changedThings, dThingID)
	// delete from the index array. A bit primitive but it is rarely used
	for i, key := range svc.thingKeys {
		if key == dThingID {
			svc.thingKeys[i] = svc.thingKeys[len(svc.thingKeys)-1]
			svc.thingKeys = svc.thingKeys[:len(svc.thingKeys)-1]
			break
		}
	}
	return err
}

// SaveChanges persists digital twins that have been modified since the
// last call to this function.
func (svc *DigitwinStore) SaveChanges() error {
	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	for dThingID := range svc.changedThings {
		dtw, found := svc.dtwCache[dThingID]
		if !found {
			slog.Error("SaveChanges. Digitwin to save not found. Skipped.",
				"dThingID", dThingID)
			continue
		}
		// write the new digital twin
		//dtwSer, err := json.Marshal(dtw)
		dtwSer, err := jsoniter.Marshal(dtw)
		//dtwSer, err := msgpack.Marshal(dtw)

		if err != nil {
			slog.Error("SaveChanges. Marshal failed. Skipped",
				"dThingID", dThingID, "err", err)
			continue
		}
		err = svc.dtwBucket.Set(dThingID, dtwSer)
		if err != nil {
			slog.Error("SaveChanges. Writing to bucket failed. Skipped",
				"dThingID", dThingID, "err", err)
			continue
		}
	}
	svc.changedThings = make(map[string]any)
	return nil
}

// UpdateTD update the provided TD and the derived digital twin TD in the
// stored digital twin record.
// If no digital twin exists yet, it is created.
func (svc *DigitwinStore) UpdateTD(
	agentID string, thingTD *tdd.TD, digitwinTD *tdd.TD) {

	slog.Debug("UpdateDTD",
		slog.String("agentID", agentID),
		slog.String("thingID", thingTD.ID))

	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()
	dThingID := digitwinTD.ID
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		dtw = &DigitalTwinInstance{
			AgentID:      agentID,
			ID:           dThingID,
			PropValues:   make(map[string]digitwin.ThingValue),
			EventValues:  make(map[string]digitwin.ThingValue),
			ActionValues: make(map[string]digitwin.ActionValue),
		}
		svc.dtwCache[dThingID] = dtw
		svc.thingKeys = append(svc.thingKeys, dThingID)
	}
	dtw.ThingTD = thingTD
	dtw.DtwTD = digitwinTD
	svc.changedThings[dThingID] = true
}

// UpdateActionStart updates the action with a new start and pending status
//
// dThingID is the digital twin thingID
// name is the name of the action whose progress is updated.
// messageID is the request messageID used to correlate the async reply
// senderID is the ID of the sender of the request
func (svc *DigitwinStore) UpdateActionStart(
	dThingID string, name string, input any, messageID string, senderID string) error {
	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err := fmt.Errorf("dThing with ID '%s' is not a known Thing", dThingID)
		return err
	}
	// action affordance must exist
	aff, found := dtw.DtwTD.Actions[name]
	_ = aff
	if !found {
		return fmt.Errorf("UpdateActionStart: Action '%s' not found in digital twin '%s'", name, dThingID)
	}
	actionValue, found := dtw.ActionValues[name]
	if !found {
		actionValue = digitwin.ActionValue{}
	}
	actionValue.Name = name
	actionValue.SenderID = senderID
	actionValue.Input = input
	actionValue.Progress = vocab.ProgressStatusPending
	actionValue.Updated = time.Now().Format(utils.RFC3339Milli)
	actionValue.MessageID = messageID
	dtw.ActionValues[name] = actionValue
	svc.changedThings[dThingID] = true

	return nil
}

// UpdateActionProgress updates the progress of the last invoked action or property write
//
//	agentID is the ID of the agent sending the update.
//	thingID is the ID of the original thing as managed by the agent.
//	name is the name of the action whose progress is updated.
//	status of the action
//	output of the action. Only used when status is completed
func (svc *DigitwinStore) UpdateActionProgress(
	agentID string, thingID string, name string, status string, output any) (
	actionValue digitwin.ActionValue, err error) {

	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	dThingID := tdd.MakeDigiTwinThingID(agentID, thingID)
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err := fmt.Errorf("dThing with ID '%s' not found", dThingID)
		return actionValue, err
	}
	// action affordance or property affordance must exist
	_, found = dtw.DtwTD.Actions[name]
	if found {
		// handle action progress
		// action value should exist. recover if it doesn't
		actionValue, found = dtw.ActionValues[name]
		if !found {
			actionValue = digitwin.ActionValue{}
		}
		actionValue.Progress = status
		actionValue.Updated = time.Now().Format(utils.RFC3339Milli)
		if status == vocab.ProgressStatusCompleted {
			actionValue.Output = output
		}
		dtw.ActionValues[name] = actionValue
		svc.changedThings[dThingID] = true

		return actionValue, nil
	} else {
		// property write progress is ignored as the thing should simply update
		//the property value after applying the write.
	}
	return actionValue, fmt.Errorf("UpdateActionProgress: Action '%s' not found in digital twin '%s'", name, dThingID)

}

// UpdateEventValue updates the last known thing event value.
//
// This does accept event values that are not defined in the TD.
// This is intentional.
//
// agentID is the ID of the agent sending the update.
// thingID is the ID of the original thing as managed by the agent.
// eventName is the name of the event whose value is updated.
// messageID is if the event is in response to an action or write property
//
// # Invoked by agents
//
// This returns the digital twin's ID where the event is stored.
func (svc *DigitwinStore) UpdateEventValue(
	agentID string, thingID string, eventName string, data any, messageID string) (string, error) {
	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	dThingID := tdd.MakeDigiTwinThingID(agentID, thingID)
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err := fmt.Errorf("dThing with ID '%s' not found", dThingID)
		slog.Info("UpdateEventValue Can't update state of an unknown Thing. Event ignored.", "dThingID", dThingID)
		return dThingID, err
	}
	eventValue := digitwin.ThingValue{
		Data:      data,
		Updated:   time.Now().Format(utils.RFC3339Milli),
		MessageID: messageID,
		Name:      eventName,
	}
	dtw.EventValues[eventName] = eventValue
	svc.changedThings[dThingID] = true

	return dThingID, nil
}

// UpdatePropertyValue updates the last known thing property value.
//
// agentID is the ID of the agent sending the update.
// thingID is the ID of the original thing as managed by the agent.
// propName is the name of the property whose value is updated or "" when value is a map
// newValue of the property or a map of property name-value pairs
// messageID provided by the agent, in response to an action or write
//
// This returns a flag indicating whether the property value has changed.
func (svc *DigitwinStore) UpdatePropertyValue(
	agentID string, thingID string, propName string, newValue any, messageID string) (
	hasChanged bool, err error) {

	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	dThingID := tdd.MakeDigiTwinThingID(agentID, thingID)
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err := fmt.Errorf("dThing with ID '%s' not found", dThingID)
		return false, err
	}
	if propName != "" {
		if svc.strict {
			aff := dtw.DtwTD.GetProperty(propName)
			if aff == nil {
				return false,
					fmt.Errorf("UpdatePropertyValue: unknown property '%s' for thing '%s'", propName, dThingID)
			}
		}
		propValue, found := dtw.PropValues[propName]
		if !found {
			propValue = digitwin.ThingValue{}
		}
		oldValue := propValue.Data
		hasChanged = oldValue != propValue

		propValue.Data = newValue
		propValue.MessageID = messageID
		propValue.Name = propName
		propValue.Updated = time.Now().Format(utils.RFC3339Milli)

		dtw.PropValues[propName] = propValue
	} else {
		// no property name, expect a map with name-value pairs
		propMap := make(map[string]any)
		err = utils.Decode(newValue, &propMap)
		if err != nil {
			err = fmt.Errorf("UpdatePropertyValue: of dthing '%s'. Expected a property map: %w", dThingID, err)
			return false, err
		}
		for propName, newValue = range propMap {
			if svc.strict {
				aff := dtw.DtwTD.GetProperty(propName)
				if aff == nil {
					return false,
						fmt.Errorf("UpdatePropertyValue: unknown property '%s' for thing '%s'", propName, dThingID)
				}
			}
			propValue, found := dtw.PropValues[propName]
			if !found {
				propValue = digitwin.ThingValue{}
			}
			oldValue := propValue.Data
			hasChanged = oldValue != propValue

			propValue.Data = newValue
			propValue.MessageID = messageID
			propValue.Name = propName
			propValue.Updated = time.Now().Format(utils.RFC3339Milli)
			dtw.PropValues[propName] = propValue
		}
	}
	svc.changedThings[dThingID] = hasChanged

	return hasChanged, nil
}

// WriteProperty stores a new  property value
// This sets the 'updated' timestamp if empty
//
//	dThingID is the digital twin ID
//	tv is the new thing value of the property
func (svc *DigitwinStore) WriteProperty(dThingID string, tv digitwin.ThingValue) error {
	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err := fmt.Errorf("dThing with ID '%s' not found", dThingID)
		return err
	}
	if tv.Updated == "" {
		tv.Updated = time.Now().Format(utils.RFC3339Milli)
	}
	dtw.PropValues[tv.Name] = tv
	svc.changedThings[dThingID] = true

	return nil
}

// OpenDigitwinStore initializes the digitwin store using the given storage bucket.
// This will load the digitwin directory into a memory cache.
// The storage bucket will be closed when the store is closed.
//
//	store is the bucket store to store the data. This is opened by 'Start' and closed by 'Stop'
func OpenDigitwinStore(bucketStore buckets.IBucketStore, strict bool) (*DigitwinStore, error) {
	bucket := bucketStore.GetBucket(DTWBucketName)

	svc := &DigitwinStore{
		dtwBucket:     bucket,
		dtwCache:      make(map[string]*DigitalTwinInstance),
		changedThings: make(map[string]any),
		thingKeys:     make([]string, 0),
		strict:        strict,
	}
	// fill the in-memory cache
	err := svc.LoadCacheFromStore()
	if err != nil {
		return svc, err
	}
	return svc, nil
}
