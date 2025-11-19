package store

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/hiveot/hivekit/go/buckets"
	"github.com/hiveot/hivekit/go/messaging"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot/td"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	jsoniter "github.com/json-iterator/go"
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
	_ = svc.SaveChanges(false)
	if svc.dtwBucket != nil {
		_ = svc.dtwBucket.Close()
		svc.dtwBucket = nil
	}
}

// GetDigitwinInfo returns the digital twin info containing both the digitwin and agent TD
// This returns nil if no such digital twin exists
func (store *DigitwinStore) GetDigitwinInfo(dThingID string) *DigitalTwinInstance {
	dti := store.dtwCache[dThingID]
	return dti
}

// LoadCacheFromStore saves the current changes and reloads the cache from
// store into memory.
func (store *DigitwinStore) LoadCacheFromStore() error {
	_ = store.SaveChanges(false)

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

// NewActionStart updates the action with a new start and pending status.
//
// This stores the action request for use with query actions.
//
// Note 'safe' actions are not stored as they don't affect the state of a Thing.
// The output is just a transformation of the input.
//
// This returns true if the request is stored or false if the request is safe.
func (svc *DigitwinStore) NewActionStart(req *messaging.RequestMessage) (
	actionStatus digitwin.ActionStatus, stored bool, err error) {
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
		return actionStatus, false, nil
	}
	// action affordance should exist
	aff, found := dtw.DigitwinTD.Actions[req.Name]
	_ = aff
	if !found {
		// FIXME:The request might not be an action but could also be a Thing level operation
		// Need to understand the use-cases this might occur before changing this
		// into an error or info. Most operations should be handled by the digital twin,
		// not by remote agents.
		err := fmt.Errorf("NewActionStart: Action '%s' not found in "+
			"digital twin '%s'. Action is not recorded", req.Name, req.ThingID)
		slog.Warn(err.Error())
		// this is currently only a warning
		return actionStatus, false, nil
	}
	if aff.Safe {
		// Safe actions do not affect the Thing state. The response is a function
		// if the input parameters so no use storing these.
		slog.Info("Not recording a safe action as it doesnt affect Thing state")
		return actionStatus, false, nil
	}
	// this is an action that affects the Thing state (not safe). Record it.
	actionStatus, found = dtw.ActionStatuses[req.Name]
	if !found {
		actionStatus = digitwin.ActionStatus{}
	}
	actionStatus.Name = req.Name
	actionStatus.ThingID = req.ThingID
	actionStatus.SenderID = req.SenderID
	actionStatus.Input = req.Input
	actionStatus.State = messaging.StatusPending
	actionStatus.TimeRequested = req.Created
	actionStatus.ActionID = req.CorrelationID
	dtw.ActionStatuses[req.Name] = actionStatus
	svc.changedThings[req.ThingID] = true

	return actionStatus, true, nil
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
		aff, found := dtw.DigitwinTD.Actions[name]
		_ = aff
		if !found {
			return v, fmt.Errorf("QueryAction: Action '%s' not found in digital twin '%s'", name, dThingID)
		}
	}
	v, found = dtw.ActionStatuses[name]
	_ = found
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
		err = fmt.Errorf("GetAllProperties: dThing with ID '%s' not found", dThingID)
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
		aff, found := dtw.DigitwinTD.Events[name]
		_ = aff
		if !found {
			return v, fmt.Errorf("ReadEvent: Event '%s' not found in digital twin '%s'", name, dThingID)
		}
	}
	// event value might not exist
	v, found = dtw.EventValues[name]
	_ = found
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
		aff, found := dtw.DigitwinTD.Properties[name]
		_ = aff
		if !found {
			return v, fmt.Errorf("ReadProperty: Property '%s' not found in digital twin '%s'", name, dThingID)
		}
	}
	// value might not exist is optional
	v, found = dtw.PropValues[name]
	_ = found
	return v, nil
}

// ReadDThing returns the Digitwin TD document in the store.
func (svc *DigitwinStore) ReadDThing(dThingID string) (dtd *td.TD, err error) {

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	dtw, found := svc.dtwCache[dThingID]
	if !found {
		err = fmt.Errorf("dThing with ID '%s' not found", dThingID)
		return dtd, err
	}
	return dtw.DigitwinTD, err
}

// ReadTDs returns a list of digital twin TDs
//
// limit is the maximum number of records to return
// offset is the offset of the first record to return
func (svc *DigitwinStore) ReadTDs(offset int64, limit int64) (resp []*td.TD, err error) {

	svc.cacheMux.RLock()
	defer svc.cacheMux.RUnlock()
	// Use the thingKeys index to ensure consistent iteration and to quickly
	// skip offset items (maps are not consistent between iterations)
	if offset >= int64(len(svc.thingKeys)) {
		// empty result
		return resp, nil
	}
	if offset+limit > int64(len(svc.thingKeys)) {
		limit = int64(len(svc.thingKeys)) - offset
	}
	tdKeys := svc.thingKeys[offset:limit]
	resp = make([]*td.TD, 0, len(tdKeys))

	// add the documents
	for _, k := range tdKeys {
		dtw := svc.dtwCache[k]
		resp = append(resp, dtw.DigitwinTD)
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
//
//	background save in the background
func (svc *DigitwinStore) SaveChanges(background bool) error {
	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	// Serialize the changed things for persisting to the bucket store in the background
	changedDtwJson := make(map[string][]byte)
	for dThingID := range svc.changedThings {
		var dtwSer []byte
		dtw, found := svc.dtwCache[dThingID]
		// if the thing is no longer there it has been deleted.
		if found {
			dtwSer, _ = jsoniter.Marshal(dtw)
		}
		changedDtwJson[dThingID] = dtwSer
	}

	// Don't block the digital twins. Update the store in the background.
	saveit := func() {
		for dThingID, dThingJSON := range changedDtwJson {
			var err error
			if dThingJSON != nil {
				err = svc.dtwBucket.Set(dThingID, dThingJSON)
			} else {
				err = svc.dtwBucket.Delete(dThingID)
			}
			if err != nil {
				slog.Error("SaveChanges. Writing to bucket failed. Skipped",
					"dThingID", dThingID, "err", err)
				continue
			}
		}
	}
	if background {
		go saveit()
	} else {
		saveit()
	}

	// continue with a clean list of changed things
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
	dtw.AgentTD = thingTD
	dtw.DigitwinTD = digitwinTD
	svc.changedThings[dThingID] = true
}

// UpdateActionWithNotification (by agent) updates the action with a notification.
//
// notif is a action notification with progress status
func (svc *DigitwinStore) UpdateActionWithNotification(notif *messaging.NotificationMessage) {
	var actionStatus digitwin.ActionStatus
	var rxStatus digitwin.ActionStatus

	err := utils.DecodeAsObject(notif.Value, &rxStatus)
	if err != nil || rxStatus.State == "" {
		slog.Warn("UpdateActionWithNotification: Notification does not contain an ActionStatus")
		return
	}

	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	dtw, found := svc.dtwCache[notif.ThingID]
	if !found {
		// not a known thing
		slog.Warn("UpdateActionWithNotification: not a known Thing",
			"thingID", notif.ThingID,
		)
		return
	}
	actionStatus, found = dtw.ActionStatuses[notif.Name]
	if !found {
		slog.Warn("UpdateActionWithNotification: no status for the action",
			"actionID", rxStatus.ActionID,
			"thingID", notif.ThingID,
			"name", notif.Name,
		)
		return
	}
	if actionStatus.State == messaging.StatusCompleted {
		slog.Warn("UpdateActionWithNotification: Action is already completed",
			"actionID", rxStatus.ActionID,
			"thingID", notif.ThingID,
			"name", notif.Name,
		)
		return
	}
	actionStatus.TimeUpdated = rxStatus.TimeUpdated
	actionStatus.State = rxStatus.State
	dtw.ActionStatuses[actionStatus.Name] = actionStatus
	svc.changedThings[actionStatus.ThingID] = true
}

// UpdateActionWithResponse (by agent) updates the action with a response.
//
// Note that a response means that the action is completed.
//
// resp is a response from invokeaction containing an ActionStatus as value
func (svc *DigitwinStore) UpdateActionWithResponse(
	resp *messaging.ResponseMessage) (actionStatus digitwin.ActionStatus, err error) {

	svc.cacheMux.Lock()
	defer svc.cacheMux.Unlock()

	dtw, found := svc.dtwCache[resp.ThingID]
	if !found {
		err := fmt.Errorf("dThing with ID '%s' not found", resp.ThingID)
		return actionStatus, err
	}
	// action affordance must exist
	_, found = dtw.DigitwinTD.Actions[resp.Name]
	if !found {
		return actionStatus, fmt.Errorf(
			"UpdateActionWithResponse: Action '%s' not found in digital twin '%s'",
			resp.Name, resp.ThingID)
	}

	// action response contains a messaging.ActionStatus object
	var respStatus messaging.ActionStatus
	err = utils.DecodeAsObject(resp.Value, &respStatus)

	if err != nil {
		// the response does not hold an ActionStatus object,
		// assume it is completed and contains the output directly.
		respStatus.Output = resp.Value
		respStatus.TimeUpdated = resp.Timestamp
		respStatus.State = messaging.StatusCompleted
		respStatus.Error = resp.Error
		// respStatus.ActionID=resp.CorrelationID

		// The protocol binding didn't decode the action response correctly
		// because in hiveot all responses should hold the action status
		slog.Error("UpdateActionWithResponse: Invalid ActionStatus in response. Recover by using the value as the output",
			"thingID", resp.ThingID,
			"actionName", resp.Name,
			"value", resp.ToString(20),
			"err", err.Error(),
		)
	}

	// this is a progress update; update only the status fields.
	// action value should exist. recover if it doesn't
	actionStatus, found = dtw.ActionStatuses[resp.Name]
	if !found {
		// an existing action status is expected. Recover by creating a new one.
		// convert messaging.ActionStatus to api.ActionStatus
		err = utils.Decode(respStatus, &actionStatus)
		if err != nil {
			slog.Error("UpdateActionWithResponse: Cannot decode ActionStatus from response",
				"thingID", resp.ThingID,
				"actionName", resp.Name,
				"err", err.Error(),
			)
			return actionStatus, err
		}
		actionStatus.Input = nil // input is not in the response
	}
	actionStatus.TimeUpdated = utils.FormatNowUTCMilli()
	if resp.Error != nil {
		// if the response itself holds an error
		actionStatus.Error = &digitwin.ErrorValue{
			Detail: resp.Error.Detail,
			Status: int64(resp.Error.Status),
			Title:  resp.Error.Title,
			Type:   resp.Error.Type,
		}
		actionStatus.State = messaging.StatusFailed
	} else if respStatus.Error != nil {
		// if the response actionStatus holds the error
		actionStatus.Error = &digitwin.ErrorValue{
			Detail: respStatus.Error.Detail,
			Status: int64(respStatus.Error.Status),
			Title:  respStatus.Error.Title,
			Type:   respStatus.Error.Type,
		}
		actionStatus.State = respStatus.State
	} else {
		// no error
		actionStatus.Output = respStatus.Output
		actionStatus.TimeUpdated = respStatus.TimeUpdated
		actionStatus.State = respStatus.State
	}
	dtw.ActionStatuses[resp.Name] = actionStatus
	svc.changedThings[resp.ThingID] = true

	return actionStatus, nil

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

	ev.AffordanceType = string(messaging.AffordanceTypeEvent)
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

	newValue.AffordanceType = string(messaging.AffordanceTypeProperty)
	dtw, found := svc.dtwCache[newValue.ThingID]
	if !found {
		err := fmt.Errorf("dThing with ID '%s' not found", newValue.ThingID)
		return false, err
	}
	if svc.strict {
		aff := dtw.DigitwinTD.GetProperty(newValue.Name)
		if aff == nil {
			return false,
				fmt.Errorf("UpdatePropertyValue: unknown property '%s' for thing '%s'",
					newValue.Name, newValue.ThingID)
		}
	}
	// TODO: value timestamp sanity check. Is it worth it?
	hasChanged = true
	dtw.PropValues[newValue.Name] = newValue
	svc.changedThings[newValue.ThingID] = true
	//propValue, found := dtw.PropValues[newValue.Name]
	//if !found {
	//	hasChanged = true
	//	dtw.PropValues[newValue.Name] = newValue
	//	svc.changedThings[newValue.ThingID] = true
	//} else if newValue.Timestamp < propValue.Timestamp {
	//	slog.Warn("Timestamp of new property value is before last value",
	//		"thingID", propValue.ThingID,
	//		"name", propValue.Name,
	//		"last timestamp", propValue.Timestamp,
	//		"new timestamp", newValue.Timestamp,
	//	)
	//	hasChanged = false
	//} else if newValue.Timestamp > now {
	//	slog.Warn("Timestamp of new value is in the future")
	//	hasChanged = false
	//} else {
	//	hasChanged = true
	//	dtw.PropValues[newValue.Name] = newValue
	//	svc.changedThings[newValue.ThingID] = true
	//}
	//_ = propValue

	return hasChanged, nil
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
		slog.Error("OpenDigitwinStore failed", "err", err.Error())
		return svc, err
	}
	return svc, nil
}
