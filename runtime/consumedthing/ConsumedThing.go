package consumedthing

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/services/history/historyclient"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/consumer"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"sync"
	"time"
)

// InteractionListener is the handler that receives updates to interaction
// requests, eg write property, invoke action or subscribe to events.
type InteractionListener func(*InteractionOutput)

// ConsumedThing implements the ConsumedThing interface for accessing a Thing's
// schema and values roughly in line with the WoT scripting API.
//
// However, since the scripting API is based on Javascript some differences are
// made to accommodate the different environment.
//
// This keeps a copy of the Thing's property and event values and updates on changes.
type ConsumedThing struct {
	// The consumer instance this uses for invoking actions
	co *consumer.Consumer

	// ID of this Thing for use by consumers
	ThingID string

	// associated thing description instance
	tdi *td.TD
	// observer of property value changes by property name
	observers map[string]InteractionListener
	// subscribers to events by eventName
	subscribers map[string]InteractionListener

	// action status output values
	actionOutputs map[string]*InteractionOutput
	// prop values
	propValues map[string]*InteractionOutput
	// event values
	eventValues map[string]*InteractionOutput

	mux sync.RWMutex
}

// build a map of interaction outputs for the given values
//func (ct *ConsumedThing) buildInteractionOutputMap(
//	tmm map[string]*transports.ResponseMessage) map[string]*InteractionOutput {
//
//	outMap := make(map[string]*InteractionOutput)
//	for key, msg := range tmm {
//		iout := NewInteractionOutputFromResponse(msg, ct.tdi)
//		outMap[key] = iout
//	}
//	return outMap
//}

// Create an interactionOutput for the given thing message
//func (ct *ConsumedThing) buildInteractionOutput(msg *transports.ResponseMessage) *InteractionOutput {
//	iout := NewInteractionOutputFromResponse(msg, ct.tdi)
//	return iout
//}

// GetActionInput returns the action input value of the given action, if available
func (ct *ConsumedThing) GetActionInput(as transports.ActionStatus) *InteractionInput {
	iin := NewInteractionInput(ct.tdi, as.Name, as.Input)
	return iin
}

// GetActionOutput returns the interaction output of the given action status
// See also GetValue that always return an iout (for rendering purpose)
//
// This returns nil if name is not a known action
func (ct *ConsumedThing) GetActionOutput(as transports.ActionStatus) (iout *InteractionOutput) {
	iout = NewInteractionOutput(ct.GetTD(), transports.AffordanceTypeAction, as.Name, as.Output, as.Updated)
	return iout
}

// GetActionStatus returns the ActionStatus object of the latest action value.
//
// This returns nil if not found
//func (ct *ConsumedThing) GetActionStatus(name string) (as *transports.ActionStatus) {
//	ct.mux.RLock()
//	as, _ = ct.actionStatus[name]
//	ct.mux.RUnlock()
//	return as
//}

// GetAtTypeTitle return the Thing @type field as a human readable text
// If @type contains an array then the title of the first value is returned.
func (ct *ConsumedThing) GetAtTypeTitle() string {
	atTypeValue := ""
	switch t := ct.tdi.AtType.(type) {
	case string:
		atTypeValue = t
	case []string:
		if len(t) > 0 {
			atTypeValue = t[0]
		}
	}
	// FIXME: read from file to support different vocabularies?
	atTypeVocab, found := vocab.ThingClassesMap[atTypeValue]
	if !found {
		return atTypeValue
	}
	return atTypeVocab.Title
}

// GetEventOutput returns the interaction output of the latest value of an event
// See also GetValue that always return an iout (for rendering purpose)
//
// This returns nil if not found
func (ct *ConsumedThing) GetEventOutput(name string) (iout *InteractionOutput) {
	ct.mux.RLock()
	iout, _ = ct.eventValues[name]
	ct.mux.RUnlock()
	return iout
}

// GetPropInput returns the property input value for writing
func (ct *ConsumedThing) GetPropInput(name string) *InteractionInput {
	ct.mux.RLock()
	iout, _ := ct.propValues[name]
	ct.mux.RUnlock()
	if iout == nil {
		return nil
	}
	iin := NewInteractionInput(ct.tdi, name, iout.Value.Raw)
	return iin
}

// GetPropOutput returns the interaction output of the latest property value.
// See also GetValue that always return an iout (for rendering purpose)
//
// This returns nil if not found
func (ct *ConsumedThing) GetPropOutput(name string) (iout *InteractionOutput) {
	ct.mux.RLock()
	iout, _ = ct.propValues[name]
	ct.mux.RUnlock()
	return iout
}

// GetTD return the TD document that is represented here.
func (ct *ConsumedThing) GetTD() *td.TD {
	return ct.tdi
}

// GetTitle return the TD document title
// If a title property is available return its value instead of the TD title.
// This lets a Thing update its TD title without re-issuing a new TD.
func (ct *ConsumedThing) GetTitle() string {
	title := ct.tdi.Title
	ct.mux.RLock()
	iout, found := ct.propValues[vocab.PropDeviceTitle]
	ct.mux.RUnlock()
	if found {
		title = iout.Value.Text()
	}
	return title
}

// GetValue returns the interaction output of the latest event or property value.
//
// If name is an event it is returned first, otherwise it falls back to property.
//
// This returns an empty InteractionOutput if not found
func (ct *ConsumedThing) GetValue(name string) *InteractionOutput {
	ct.mux.RLock()
	iout, found := ct.eventValues[name]
	if !found {
		iout, found = ct.propValues[name]
	}
	if !found {
		iout, found = ct.actionOutputs[name]
	}
	ct.mux.RUnlock()
	_ = found
	if iout == nil {
		// not a known prop or event value so create an empty io with a schema from the td
		iout = &InteractionOutput{
			ThingID: ct.tdi.ID,
			Name:    name,
		}
	}
	return iout
}

// GetAllProperties returns all Thing property values and returns them in a
// map of InteractionOutputs.
func (ct *ConsumedThing) GetAllProperties() map[string]*InteractionOutput {
	return ct.propValues
}

// GetAllActions returns all Thing action status values and returns them in a
// map of InteractionOutputs.
func (ct *ConsumedThing) GetAllActions() map[string]*InteractionOutput {
	return ct.actionOutputs
}

// InvokeAction requests an action on the Thing
// This updates the action status with the response. For async actions the owner
// of the Consumer must invoke OnResponse to update the action status.
func (ct *ConsumedThing) InvokeAction(name string, iin InteractionInput) (*InteractionOutput, error) {

	req := transports.NewRequestMessage(
		wot.OpInvokeAction, ct.ThingID, name, iin.Value.Raw, "")
	resp, err := ct.co.SendRequest(req, true)
	if err != nil {
		return nil, err
	}
	// update the
	iout := NewInteractionOutput(ct.tdi, transports.AffordanceTypeAction, name, resp.Output, resp.Updated)
	ct.mux.Lock()
	ct.actionOutputs[name] = iout
	ct.mux.Unlock()
	return iout, nil
}

// ObserveProperty registers a callback for updates to property values.
// Only a single subscription per property is allowed. This returns an error
// if an existing observer is already registered.
func (ct *ConsumedThing) ObserveProperty(name string, listener InteractionListener) error {
	if _, found := ct.observers[name]; found {
		return fmt.Errorf("A property observer is already registered")
	}
	ct.observers[name] = listener
	return nil
}

// OnResponse handles receiving a Thing event.
// To be called by the manager of this ConsumerThing, the one that receives
// all subscribed events from the hub client.
// This updates the latest event value and invokes the registered event subscriber, if any.
//
//	msg is the notification message received.
func (ct *ConsumedThing) OnResponse(msg *transports.ResponseMessage) {

	if msg.Operation == wot.OpSubscribeEvent &&
		msg.ThingID == digitwin.ThingDirectoryDThingID &&
		msg.Name == digitwin.ThingDirectoryEventThingUpdated {
		// decode the TD
		tdi := &td.TD{}
		err := jsoniter.UnmarshalFromString(msg.ToString(0), &tdi)
		if err != nil {
			slog.Error("invalid payload for TD event. Ignored",
				"thingID", msg.ThingID)
			return
		}
		// update consumed thing, if existing
		ct.mux.Lock()
		ct.tdi = tdi
		ct.mux.Unlock()
	} else if msg.Operation == wot.OpObserveProperty {
		// update value
		iout := NewInteractionOutputFromResponse(ct.tdi, transports.AffordanceTypeProperty, msg)
		ct.mux.Lock()
		ct.propValues[msg.Name] = iout
		ct.mux.Unlock()
		subscr, _ := ct.subscribers[msg.Name]
		if subscr != nil {
			subscr(iout)
		}
	} else if msg.Operation == wot.OpSubscribeEvent {
		iout := NewInteractionOutputFromResponse(ct.tdi, transports.AffordanceTypeEvent, msg)
		// this is a regular value event
		ct.mux.Lock()
		ct.eventValues[msg.Name] = iout
		ct.mux.Unlock()
		subscr, _ := ct.subscribers[msg.Name]
		if subscr != nil {
			subscr(iout)
		}
	} else if msg.Operation == wot.OpInvokeAction {
		iout := NewInteractionOutputFromResponse(ct.tdi, transports.AffordanceTypeAction, msg)
		// this is a regular action progress event
		ct.mux.Lock()
		ct.actionOutputs[msg.Name] = iout
		ct.mux.Unlock()
		subscr, _ := ct.subscribers[msg.Name]
		if subscr != nil {
			subscr(iout)
		}
	}
}

// QueryAction queries the action status record from the hub
//
// # The cached interaction output of this value can be obtained with GetActionOutput
//
// This returns an empty ActionStatus if not found
func (ct *ConsumedThing) QueryAction(name string) transports.ActionStatus {
	as, _ := ct.co.QueryAction(ct.ThingID, name)
	return as
}

// ReadEvent refreshes the last event value by reading it from the hub
// TODO: subscribing to events should send the last one
func (ct *ConsumedThing) ReadEvent(name string) *InteractionOutput {

	tv, err := digitwin.ThingValuesReadEvent(ct.co, name, ct.ThingID)
	if err != nil {
		return nil
	}
	iout := NewInteractionOutput(ct.tdi, transports.AffordanceTypeEvent, name, tv.Output, tv.Updated)
	//iout.setSchemaFromTD(ct.tdi)
	ct.mux.Lock()
	ct.eventValues[name] = iout
	ct.mux.Unlock()
	return iout
}

// ReadHistory returns the history of a property or event.
//
// This requires the history service.
//
// If the number of values exceed the maximum then this returns itemsRemaining
// as true. An additional call can be made using the last returned timestamp to
// get the remaining values.
func (ct *ConsumedThing) ReadHistory(
	name string, timestamp time.Time, duration time.Duration) (
	values []*transports.ThingValue, itemsRemaining bool, err error) {

	// FIXME: ReadHistory is not (yet) part of the WoT specification. Ege mentioned it would
	// be added soon so this will change to follow the WoT specification.
	// Until then this is tied to the Hub's history service.

	hist := historyclient.NewReadHistoryClient(ct.co)
	// todo: is there a need to read in batches? not for a single day.
	values, itemsRemaining, err = hist.ReadHistory(
		ct.tdi.ID, name, timestamp, duration, 500)

	return values, itemsRemaining, err
}

// ReadProperty reads the Thing property value from the Thing and updates its cache.
// Call GetPropertyValue to get the cached value.
func (ct *ConsumedThing) ReadProperty(name string) *InteractionOutput {

	resp, err := ct.co.ReadProperty(ct.ThingID, name)
	if err != nil {
		return nil
	}
	iout := NewInteractionOutput(ct.tdi, transports.AffordanceTypeProperty, name, resp.Output, resp.Updated)
	ct.mux.Lock()
	ct.propValues[name] = iout
	ct.mux.Unlock()
	return iout
}

// Refresh reloads all property and event values from the Hub and updates the
// cache.
func (ct *ConsumedThing) Refresh() error {
	var iout *InteractionOutput
	// refresh events
	valueMap, err := digitwin.ThingValuesReadAllEvents(ct.co, ct.ThingID)
	if err != nil {
		return err
	}
	for name, _ := range ct.tdi.Events {
		tv, found := valueMap[name]
		if found {
			iout = NewInteractionOutputFromValue(ct.tdi, transports.AffordanceTypeEvent, tv)
		} else {
			iout = NewInteractionOutput(ct.tdi, transports.AffordanceTypeEvent, name, nil, "")
		}
		ct.propValues[name] = iout
	}
	// refresh properties
	valueMap, err = digitwin.ThingValuesReadAllProperties(ct.co, ct.ThingID)
	//valueMap, err = ct.co.ReadAllProperties(ct.ThingID)
	if err != nil {
		return err
	}
	for name, _ := range ct.tdi.Properties {
		tv, found := valueMap[name]
		if found {
			iout = NewInteractionOutputFromValue(ct.tdi, transports.AffordanceTypeProperty, tv)
		} else {
			iout = NewInteractionOutput(
				ct.tdi, transports.AffordanceTypeProperty, name, nil, "")
		}
		ct.propValues[name] = iout
	}
	// refresh action status
	actionStatusMap, err := ct.co.QueryAllActions(ct.ThingID)
	if err != nil {
		return err
	}
	for name, as := range actionStatusMap {
		// if the TD doesn't have this property then ignore it
		actionAff := ct.tdi.GetAction(name)
		if actionAff != nil {
			iout := NewInteractionOutputFromActionStatus(ct.tdi, as)
			ct.actionOutputs[name] = iout
		}
	}
	return nil
}

// SubscribeEvent sets the handler to invoke when event with the name is received
// This returns an error if an existing subscriber already exists
func (ct *ConsumedThing) SubscribeEvent(name string, listener InteractionListener) error {
	if _, found := ct.subscribers[name]; found {
		return fmt.Errorf("An event subscriber is already registered")
	}
	ct.subscribers[name] = listener
	return nil
}

// WriteProperty requests a change to a property value.
//
// Note that writing a property can take some time, especially if the device is asleep.
// run as a go-routing if measures are taken to receive async updates.
//
// Note that WoT does not specify a mechanism to confirm the success or failure of
// delivery and applying the value so this might timeout on incompatible Things.
//
// This returns an error if the request wasnt be completed.
//
// FIXME: provide a way to handle timeouts more gracefully. Maybe this should
// use async?
func (ct *ConsumedThing) WriteProperty(name string, ii InteractionInput) (err error) {
	err = ct.co.WriteProperty(ct.ThingID, name, ii.Value.Raw, true)
	return err
}

// NewConsumedThing creates a new instance of a Thing
// Call Stop() when done
func NewConsumedThing(tdi *td.TD, co *consumer.Consumer) *ConsumedThing {
	c := ConsumedThing{
		ThingID:       tdi.ID,
		tdi:           tdi,
		co:            co,
		observers:     make(map[string]InteractionListener),
		subscribers:   make(map[string]InteractionListener),
		actionOutputs: make(map[string]*InteractionOutput),
		eventValues:   make(map[string]*InteractionOutput),
		propValues:    make(map[string]*InteractionOutput),
	}
	return &c
}
