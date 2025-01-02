package store

import (
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"sync"
	"time"
)

// DTWBucketName contains the name of the digital twin instances storage bucket
const DTWBucketName = "dtw"

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
	dThingID string, name string) (v digitwin.ActionStatus, err error) {

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
	v, found = dtw.ActionStatuses[name]
	return v, nil
}

// QueryAllActions returns all last known action invocation status of the given thing
func (svc *DigitwinStore) QueryAllActions(dThingID string) (
	v map[string]digitwin.ActionStatus, err error) {

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err = fmt.Errorf("QueryAllActions: dThing with ID '%s' not found", dThingID)
		return v, err
	}
	// shallow copy
	actMap := make(map[string]digitwin.ActionStatus)
	for k, v := range dtw.ActionStatuses {
		actMap[k] = v
	}
	return actMap, err
}

// ReadAllEvents returns all last received events of the given thing
func (svc *DigitwinStore) ReadAllEvents(dThingID string) (
	v map[string]digitwin.ThingValue, err error) {

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err = fmt.Errorf("ReadAllEvents: dThing with ID '%s' not found", dThingID)
		return v, err
	}

	// shallow copy
	evMap := make(map[string]digitwin.ThingValue)
	for k, v := range dtw.EventValues {
		evMap[k] = v
	}
	return evMap, err
}

// ReadAllProperties returns a shallow copy of all last known property values of the given thing
func (svc *DigitwinStore) ReadAllProperties(dThingID string) (
	v map[string]digitwin.ThingValue, err error) {

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	dtw, found := svc.dtwCache[dThingID]

	if !found {
		err = fmt.Errorf("ReadAllProperties: dThing with ID '%s' not found", dThingID)
		return v, err
	}
	// shallow copy
	propMap := make(map[string]digitwin.ThingValue)
	for k, v := range dtw.PropValues {
		propMap[k] = v
	}
	return propMap, err
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
func (svc *DigitwinStore) ReadDThing(dThingID string) (dtd *td.TD, err error) {

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
func (svc *DigitwinStore) ReadTDs(offset int, limit int) (resp []*td.TD, err error) {

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
	resp = make([]*td.TD, 0, len(tdKeys))

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
//
// The given TD is  provided by the agent and has the agent's ThingID.
// If no digital twin exists yet, it is created.
//
// This returns false if the given TD is the same as the one on record.
func (svc *DigitwinStore) UpdateTD(
	agentID string, thingTD *td.TD, digitwinTD *td.TD) {

	slog.Debug("UpdateDTD",
		slog.String("agentID", agentID),
		slog.String("thingID", thingTD.ID))

	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()
	dThingID := digitwinTD.ID
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		dtw = &DigitalTwinInstance{
			AgentID:        agentID,
			ID:             dThingID,
			PropValues:     make(map[string]digitwin.ThingValue),
			EventValues:    make(map[string]digitwin.ThingValue),
			ActionStatuses: make(map[string]digitwin.ActionStatus),
		}
		svc.dtwCache[dThingID] = dtw
		svc.thingKeys = append(svc.thingKeys, dThingID)
	}
	dtw.ThingTD = thingTD
	dtw.DtwTD = digitwinTD
	svc.changedThings[dThingID] = true
}

// NewActionStart updates the action with a new start and pending status.
//
// This stores the action request for use with query actions.
//
// Note 'safe' actions are not stored as they don't affect the state of a Thing.
// The output is just a transformation of the input.
//
// This returns true if the request is stored or false if the request is safe.
func (svc *DigitwinStore) NewActionStart(req transports.RequestMessage) (stored bool, err error) {
	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	// A digital twin must exist. They are created when a TD is received.
	dtw, found := svc.dtwCache[req.ThingID]
	if !found {
		// for now just warn for unknown things, in case the TD is not yet known.
		// This might become an error in the future once the use-cases that can
		// trigger this are better understood.
		err := fmt.Errorf(
			"NewActionStart: Action requested on an unknown Thing with ID '%s'. "+
				"Action is not recorded", req.ThingID)
		slog.Warn(err.Error())
		return false, nil
	}
	// action affordance should exist
	aff, found := dtw.DtwTD.Actions[req.Name]
	_ = aff
	if !found {
		// The request might not be an action but could also be a Thing level operation
		// Need to understand the use-cases this might occur before changing this
		// into an error or info. Most operations should be handled by the digital twin,
		// not by remote agents.
		err := fmt.Errorf("NewActionStart: Action '%s' not found in "+
			"digital twin '%s'. Action is not recorded", req.Name, req.ThingID)
		slog.Warn(err.Error())
		// this is currently only a warning
		return false, nil
	}
	if aff.Safe {
		// Safe actions do not affect the Thing state. The response is a function
		// if the input parameters so no use storing these.
		slog.Info("Not recording a safe action as it doesnt affect Thing state")
		return false, nil
	}
	// this is an action that affects the Thing state (not safe). Record it.
	actionStatus, found := dtw.ActionStatuses[req.Name]
	if !found {
		actionStatus = digitwin.ActionStatus{}
	}
	actionStatus.Name = req.Name
	actionStatus.SenderID = req.SenderID
	actionStatus.Input = req.Input
	actionStatus.Status = transports.StatusPending
	actionStatus.TimeRequested = req.Created
	actionStatus.CorrelationID = req.CorrelationID
	dtw.ActionStatuses[req.Name] = actionStatus
	svc.changedThings[req.ThingID] = true

	return true, nil
}

// UpdateActionStatus (by agent) updates the progress of an action with a
// progress response.
//
// resp is a progress response with a ThingID of the digital twin
func (svc *DigitwinStore) UpdateActionStatus(agentID string, resp transports.ResponseMessage) (
	actionValue digitwin.ActionStatus, err error) {

	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	dtw, found := svc.dtwCache[resp.ThingID]
	if !found {
		err := fmt.Errorf("dThing with ID '%s' not found", resp.ThingID)
		return actionValue, err
	}
	// action affordance or property affordance must exist
	_, found = dtw.DtwTD.Actions[resp.Name]
	if found {
		// this is a progress update; update only the status fields.
		// action value should exist. recover if it doesn't
		actionValue, found = dtw.ActionStatuses[resp.Name]
		if !found {
			actionValue = digitwin.ActionStatus{}
		}
		actionValue.Status = resp.Status
		actionValue.TimeUpdated = time.Now().Format(wot.RFC3339Milli)
		if resp.Status == transports.StatusCompleted {
			actionValue.Output = resp.Output
			actionValue.TimeEnded = resp.Updated
		} else if resp.Error != "" {
			actionValue.Error = resp.Error
		}
		dtw.ActionStatuses[resp.Name] = actionValue
		svc.changedThings[resp.ThingID] = true

		return actionValue, nil
	} else {
		// property write progress is ignored as the thing should simply update
		//the property value after applying the write.
	}
	return actionValue, fmt.Errorf(
		"UpdateActionStatus: Action '%s' not found in digital twin '%s'",
		resp.Name, resp.ThingID)

}

// UpdateEventValue updates the last known thing event value.
//
// This does accept event values that are not defined in the TD.
// This is intentional.
//
// ev is the received notification of the event update
func (svc *DigitwinStore) UpdateEventValue(ev digitwin.ThingValue) error {
	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	dtw, found := svc.dtwCache[ev.ThingID]
	if !found {
		err := fmt.Errorf("dThing with ID '%s' not found", ev.ThingID)
		slog.Warn("UpdateEventValue Unknown Thing. Event ignored.", "dThingID", ev.ThingID)
		return err
	}
	dtw.EventValues[ev.Name] = ev
	svc.changedThings[ev.ThingID] = true

	return nil
}

// UpdatePropertyValue updates the last known thing property value.
//
// newValue of the property
//
// This returns a flag indicating whether the property value has changed.
func (svc *DigitwinStore) UpdatePropertyValue(newValue digitwin.ThingValue) (
	hasChanged bool, err error) {

	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	dtw, found := svc.dtwCache[newValue.ThingID]
	if !found {
		err := fmt.Errorf("dThing with ID '%s' not found", newValue.ThingID)
		return false, err
	}
	if svc.strict {
		aff := dtw.DtwTD.GetProperty(newValue.Name)
		if aff == nil {
			return false,
				fmt.Errorf("UpdatePropertyValue: unknown property '%s' for thing '%s'",
					newValue.Name, newValue.ThingID)
		}
	}
	propValue, found := dtw.PropValues[newValue.Name]
	if !found {
		hasChanged = true
	} else {
		hasChanged = propValue.Data != newValue.Data
	}
	dtw.PropValues[newValue.Name] = newValue
	svc.changedThings[newValue.ThingID] = hasChanged

	return hasChanged, nil
}

// UpdateProperties updates the last known thing property values with a new
// property value notification.
//
// This will bulk update all properties in the map. They are stored separately.
//
// agentID is the ID of the agent sending the update.
// dThingID is the ID of the digital twin.
// propMap map of property name-value pairs
// correlationID provided by the agent, in response to an action or write
//
// This returns a map with changed property values.
func (svc *DigitwinStore) UpdateProperties(dThingID string, created string, propMap map[string]any) (
	changes map[string]any, err error) {

	changes = make(map[string]any)
	for k, v := range propMap {
		newValue := digitwin.ThingValue{
			Created: created,
			Data:    v,
			Name:    k,
			ThingID: dThingID,
		}

		//	wot.HTOpUpdateProperty, dThingID, k, v)
		//}
		changed, _ := svc.UpdatePropertyValue(newValue)
		if changed {
			changes[k] = newValue
		}
	}
	return changes, nil
}

// WriteProperty stores a new  property value
// This sets the 'updated' timestamp if empty
//
//	dThingID is the digital twin ID
//	tv is the new thing value of the property
//func (svc *DigitwinStore) WriteProperty(dThingID string, tv digitwin.ThingValue) error {
//	svc.cacheMux.Lock()
//	defer svc.cacheMux.Unlock()
//
//	dtw, found := svc.dtwCache[dThingID]
//	if !found {
//		err := fmt.Errorf("dThing with ID '%s' not found", dThingID)
//		return err
//	}
//	if tv.Updated == "" {
//		tv.Updated = time.Now().Format(wot.RFC3339Milli)
//	}
//	dtw.PropValues[tv.Name] = tv
//	svc.changedThings[dThingID] = true
//
//	return nil
//}

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
