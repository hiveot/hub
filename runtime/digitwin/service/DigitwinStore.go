package service

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime/digitwin"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"sync"
	"time"
)

type DigitwinStore struct {
	// The digital twin storage bucket
	dtwBucket buckets.IBucket

	// in-memory cache of the digital twin Things by dThingID
	dtwCache map[string]*digitwin.DigitalTwinInstance

	// map of changed digital thing IDs
	changedThings map[string]any

	// list of DTW thingIDs used as a consistent iterator for reading batches
	thingKeys []string

	// mutex for read/writing the cache
	cacheMux sync.RWMutex // mutex for the following two fields
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
	store.dtwCache = make(map[string]*digitwin.DigitalTwinInstance)
	store.changedThings = make(map[string]any)
	for {
		// read in batches of 300 TD documents
		tdmap, itemsRemaining := cursor.NextN(300)
		for dThingID, dtwSer := range tdmap {
			dtwInstance := &digitwin.DigitalTwinInstance{}
			err = json.Unmarshal(dtwSer, &dtwInstance)
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

// ReadAction returns the last known action invocation status of the given name
func (svc *DigitwinStore) ReadAction(
	dThingID string, name string) (v digitwin.DigitalTwinActionValue, err error) {

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err = fmt.Errorf("ReadAction: dThing with ID '%s' not found", dThingID)
		return v, err
	}
	v, found = dtw.ActionValues[name]
	if !found {
		return v, fmt.Errorf("ReadAction: Action '%s' not found in digital twin '%s'", name, dThingID)
	}
	return v, nil
}

// ReadAllActions returns all last known action invocation status of the given thing
func (svc *DigitwinStore) ReadAllActions(dThingID string) (
	v map[string]digitwin.DigitalTwinActionValue, err error) {

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err = fmt.Errorf("ReadAllActions: dThing with ID '%s' not found", dThingID)
		return v, err
	}
	return dtw.ActionValues, err
}

// ReadAllEvents returns all last known action invocation status of the given thing
func (svc *DigitwinStore) ReadAllEvents(dThingID string) (
	v map[string]digitwin.DigitalTwinEventValue, err error) {

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
	v map[string]digitwin.DigitalTwinPropertyValue, err error) {

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
func (svc *DigitwinStore) ReadEvent(
	dThingID string, name string) (v digitwin.DigitalTwinEventValue, err error) {

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err = fmt.Errorf("ReadEvent: dThing with ID '%s' not found", dThingID)
		return v, err
	}
	// affordance must exist
	aff, found := dtw.DtwTD.Events[name]
	_ = aff
	if !found {
		return v, fmt.Errorf("ReadEvent: Event '%s' not found in digital twin '%s'", name, dThingID)
	}
	// event value might not exist
	v, found = dtw.EventValues[name]
	return v, nil
}

// ReadProperty returns the last known property value of the given name
func (svc *DigitwinStore) ReadProperty(
	dThingID string, name string) (v digitwin.DigitalTwinPropertyValue, err error) {

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err = fmt.Errorf("ReadProperty: dThing with ID '%s' not found", dThingID)
		return v, err
	}
	// affordance must exist
	aff, found := dtw.DtwTD.Properties[name]
	_ = aff
	if !found {
		return v, fmt.Errorf("ReadProperty: Property '%s' not found in digital twin '%s'", name, dThingID)
	}

	// value might not exist is optional
	v, found = dtw.PropValues[name]
	return v, nil
}

// ReadDTW returns the Digitwin instance.
//func (svc *DigitwinStore) ReadDTW(
//	dThingID string) (resp digitwin2.DigitalTwinInstance, err error) {
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

// ReadDThing returns the Digitwin TD document.
func (svc *DigitwinStore) ReadDThing(dThingID string) (dtd tdd.TD, err error) {

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err = fmt.Errorf("dThing with ID '%s' not found", dThingID)
		return dtd, err
	}
	return dtw.DtwTD, err
}

// ReadDThingList returns a list of digital twin TDs
//
// limit is the maximum number of records to return
// offset is the offset of the first record to return
func (svc *DigitwinStore) ReadDThingList(
	offset int, limit int) (resp []tdd.TD, err error) {

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
	resp = make([]tdd.TD, 0, len(tdKeys))

	// add the documents
	for _, k := range tdKeys {
		dtw := svc.dtwCache[k]
		resp = append(resp, dtw.DtwTD)
	}
	return resp, nil
}

// RemoveDTW deletes the digitwin instance of an agent with the given ThingID
func (svc *DigitwinStore) RemoveDTW(dThingID string) error {
	// TBD: should we mark this as deleted instead? retain historical things?
	slog.Info("RemoveTD", slog.String("thingID", dThingID))

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
		dtwSer, err := json.Marshal(dtw)
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

// UpdateThing update the provided TD and the derived digital twin TD
// If no digital twin exists yet, it is created.
func (svc *DigitwinStore) UpdateThing(
	agentID string, thingTD tdd.TD, digitwinTD tdd.TD) {

	slog.Info("UpdateThing",
		slog.String("agentID", agentID),
		slog.String("thingID", thingTD.ID))

	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	dThingID := tdd.MakeDigiTwinThingID(agentID, thingTD.ID)
	digitwinTD.ID = dThingID // ensure they all use corresponding IDs
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		dtw = &digitwin.DigitalTwinInstance{
			AgentID:      agentID,
			ID:           dThingID,
			PropValues:   make(map[string]digitwin.DigitalTwinPropertyValue),
			EventValues:  make(map[string]digitwin.DigitalTwinEventValue),
			ActionValues: make(map[string]digitwin.DigitalTwinActionValue),
		}
		svc.dtwCache[dThingID] = dtw
		svc.thingKeys = append(svc.thingKeys, dThingID)
	}
	dtw.ThingTD = thingTD
	dtw.DtwTD = digitwinTD
	svc.changedThings[dThingID] = true
}

// UpdateActionStart updates the action with a new start
//
// consumerID is the ID of the consumer requesting the action.
// thingID is the ID of the original thing as managed by the agent.
// actionName is the name of the action whose progress is updated.
func (svc *DigitwinStore) UpdateActionStart(
	consumerID string, dThingID string, name string, input any, status string) error {
	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err := fmt.Errorf("dThing with ID '%s' not found", dThingID)
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
		actionValue = digitwin.DigitalTwinActionValue{}
	}
	actionValue.SenderID = consumerID
	actionValue.Input = input
	actionValue.Status = status
	actionValue.Updated = time.Now().Format(utils.RFC3339Milli)
	dtw.ActionValues[name] = actionValue
	svc.changedThings[dThingID] = true

	return nil
}

// UpdateActionProgress updates the progress of the last invoked action
//
// agentID is the ID of the agent sending the update.
// thingID is the ID of the original thing as managed by the agent.
// actionName is the name of the action whose progress is updated.
// status of the action
// output of the action. Only used when status is completed
func (svc *DigitwinStore) UpdateActionProgress(
	agentID string, thingID string, name string, status string, output any) error {
	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	dThingID := tdd.MakeDigiTwinThingID(agentID, thingID)
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err := fmt.Errorf("dThing with ID '%s' not found", dThingID)
		return err
	}
	// action affordance must exist
	aff, found := dtw.DtwTD.Actions[name]
	_ = aff
	if !found {
		return fmt.Errorf("UpdateActionProgress: Action '%s' not found in digital twin '%s'", name, dThingID)
	}
	// action value should exist. recover if it doesn't
	actionValue, found := dtw.ActionValues[name]
	if !found {
		actionValue = digitwin.DigitalTwinActionValue{}
	}
	actionValue.Status = status
	actionValue.Updated = time.Now().Format(utils.RFC3339Milli)
	if status == digitwin.StatusCompleted {
		actionValue.Output = output
	}
	dtw.ActionValues[name] = actionValue
	svc.changedThings[dThingID] = true

	return nil
}

// UpdateEventValue updates the last known thing event value.
//
// This does accept event values that are not in the TD. This is intentional.
//
// agentID is the ID of the agent sending the update.
// thingID is the ID of the original thing as managed by the agent.
// eventName is the name of the event whose value is updated.
//
// Invoked by agents
func (svc *DigitwinStore) UpdateEventValue(
	agentID string, thingID string, eventName string, data any) error {
	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	dThingID := tdd.MakeDigiTwinThingID(agentID, thingID)
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err := fmt.Errorf("dThing with ID '%s' not found", dThingID)
		return err
	}
	eventValue := digitwin.DigitalTwinEventValue{
		Data:    data,
		Updated: time.Now().Format(utils.RFC3339Milli),
	}
	dtw.EventValues[eventName] = eventValue
	svc.changedThings[dThingID] = true

	return nil
}

// UpdatePropertyValue updates the last known thing property value.
//
// agentID is the ID of the agent sending the update.
// thingID is the ID of the original thing as managed by the agent.
// propName is the name of the property whose value is updated.
// Invoked by agents
func (svc *DigitwinStore) UpdatePropertyValue(
	agentID string, thingID string, propName string, data any) error {
	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	dThingID := tdd.MakeDigiTwinThingID(agentID, thingID)
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err := fmt.Errorf("dThing with ID '%s' not found", dThingID)
		return err
	}
	propValue, found := dtw.PropValues[propName]
	if !found {
		propValue = digitwin.DigitalTwinPropertyValue{}
	}
	propValue.Data = data
	propValue.Updated = time.Now().Format(utils.RFC3339Milli)

	dtw.PropValues[propName] = propValue
	svc.changedThings[dThingID] = true

	return nil
}

// WriteProperty updates a property value status with a write request
//
//	senderID ID of the consumer requesting the write
//	dThingID is the digital twin ID
//	propName is the name of the property to write
//	data is the data to write as per TD
//	status is the write delivery status: pending, delivered,...
func (svc *DigitwinStore) WriteProperty(
	consumerID string, dThingID string, propName string, data any, writeStatus string) error {
	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err := fmt.Errorf("dThing with ID '%s' not found", dThingID)
		return err
	}
	propValue, found := dtw.PropValues[propName]
	if !found {
		propValue = digitwin.DigitalTwinPropertyValue{}
	}
	propValue.WriteData = data
	propValue.WriteSenderID = consumerID
	propValue.WriteUpdated = time.Now().Format(utils.RFC3339Milli)
	propValue.WriteStatus = writeStatus

	dtw.PropValues[propName] = propValue
	svc.changedThings[dThingID] = true

	return nil
}

// OpenDigitwinStore initializes the digitwin store using the given storage bucket.
// This will load the digitwin directory into a memory cache.
// The storage bucket will be closed when the store is closed.
//
//	store is the bucket store to store the data. This is opened by 'Start' and closed by 'Stop'
func OpenDigitwinStore(bucketStore buckets.IBucketStore) (*DigitwinStore, error) {
	bucket := bucketStore.GetBucket(DTWBucketName)

	svc := &DigitwinStore{
		dtwBucket:     bucket,
		dtwCache:      make(map[string]*digitwin.DigitalTwinInstance),
		changedThings: make(map[string]any),
		thingKeys:     make([]string, 0),
	}
	// fill the in-memory cache
	err := svc.LoadCacheFromStore()
	if err != nil {
		return svc, err
	}
	return svc, nil
}
