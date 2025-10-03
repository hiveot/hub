package consumedthing

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/messaging"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"sync"
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
	co *messaging.Consumer

	// ID of this Thing for use by consumers
	ThingID string
	// The Thing title or property 'title'
	Title string
	// The Thing description or property 'description'
	Description string

	// tdi is the immutable associated TD
	tdi *td.TD
	// observer of property value changes by property name
	observers map[string]InteractionListener
	// subscribers to events by eventName
	subscribers map[string]InteractionListener

	// action status input values
	actionInputs map[string]*InteractionInput
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

// GetActionAff returns the action affordance or nil if not found
func (ct *ConsumedThing) GetActionAff(name string) *td.ActionAffordance {
	aff, found := ct.tdi.Actions[name]
	_ = found
	return aff
}

// GetActionInputFromStatus returns the action input value of the given action, if available
func (ct *ConsumedThing) GetActionInputFromStatus(as messaging.ActionStatus) *InteractionInput {
	iin := NewInteractionInput(ct.tdi, as.Name, as.Input)
	return iin
}

// GetActionOutputFromStatus returns the interaction output of the given
// action status.
//
// If as.name is not a known action then fallback to property and event schema.
//
// See also GetValue that always return an iout (for rendering purpose)
//
// This returns nil if as.name is not a known action
func (ct *ConsumedThing) GetActionOutputFromStatus(as messaging.ActionStatus) (iout *InteractionOutput) {

	iout = NewInteractionOutput(ct, messaging.AffordanceTypeAction, as.Name, as.Output, as.TimeUpdated)

	// graceful fallback.
	// If no output schema use property or event with the same name
	if iout.Schema.Type == "" {
		fallbackOutput := ct.GetPropertyOutput(as.Name)
		if fallbackOutput == nil {
			fallbackOutput = ct.GetEventOutput(as.Name)
		}
		if fallbackOutput != nil {
			iout.Schema = fallbackOutput.Schema
			iout.Value = fallbackOutput.Value
		}
	}
	return iout
}

// GetActionOutput returns the cached interaction output of the given action affordance
//
// This returns nil if no cached value is available
func (ct *ConsumedThing) GetActionOutput(name string) (iout *InteractionOutput) {

	ct.mux.RLock()
	iout, _ = ct.actionOutputs[name]
	ct.mux.RUnlock()
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

// GetConsumer returns the consumer instance this ConsumedT hing uses to communicate with
// the server.
func (ct *ConsumedThing) GetConsumer() (co *messaging.Consumer) {
	return ct.co
}

// GetEventAff returns the event affordance or nil if not found
func (ct *ConsumedThing) GetEventAff(name string) *td.EventAffordance {
	aff, found := ct.tdi.Events[name]
	_ = found
	return aff
}

// GetEventOutput returns the cached interaction output of the latest value of an event
// See also GetValue that always return an iout (for rendering purpose)
//
// This returns nil if not found
func (ct *ConsumedThing) GetEventOutput(name string) (iout *InteractionOutput) {
	ct.mux.RLock()
	iout, _ = ct.eventValues[name]
	ct.mux.RUnlock()
	return iout
}

// GetPropertyAff returns the cached property affordance or nil if not found
func (ct *ConsumedThing) GetPropertyAff(name string) *td.PropertyAffordance {
	aff, found := ct.tdi.Properties[name]
	_ = found
	return aff
}

// GetPropertyInput returns the interaction input for a property
// This populates the input with the last known cached value.
func (ct *ConsumedThing) GetPropertyInput(name string) *InteractionInput {
	ct.mux.RLock()
	iout, _ := ct.propValues[name]
	ct.mux.RUnlock()
	if iout == nil {
		return nil
	}
	iin := NewInteractionInput(ct.tdi, name, iout.Value.Raw)
	return iin
}

// GetPropertyOutput returns the interaction output of the latest property value.
// See also GetValue that always return an iout (for rendering purpose)
//
// # This returns nil if not found
//
// FIXME: this returns an old cached value.
// Instead it should show an updated iout
func (ct *ConsumedThing) GetPropertyOutput(name string) (iout *InteractionOutput) {
	ct.mux.RLock()
	iout, _ = ct.propValues[name]
	ct.mux.RUnlock()
	return iout
}

// GetValue returns the interaction output of the latest received event or property value.
// To refresh the value use ReadValue() instead.
// Note: This returns an NoValue() InteractionOutput if name is not known
func (ct *ConsumedThing) GetValue(affType messaging.AffordanceType, name string) (iout *InteractionOutput) {
	var found bool

	//should name be matched against the affordances TD instead of cached values?
	if affType == "action" {
		iout = ct.GetActionOutput(name)
	} else if affType == "event" {
		iout = ct.GetEventOutput(name)
	} else {
		// must be a property
		iout = ct.GetPropertyOutput(name)
	}

	_ = found
	if iout == nil {
		// not a known value so create an empty io with NoSchmea
		iout = &InteractionOutput{
			ThingID: ct.tdi.ID,
			Name:    name,
			Value:   DataSchemaValue{Raw: "ConsumedThing is missing data"},
			Schema:  td.NoSchema(),
		}
	}
	return iout
}

// GetAllActionOutputs returns all Thing action status values and returns them in a
// map of InteractionOutputs.
func (ct *ConsumedThing) GetAllActionOutputs() map[string]*InteractionOutput {
	return ct.actionOutputs
}

// GetAllEvents returns all Thing event  values and returns them in a
// map of InteractionOutputs.
func (ct *ConsumedThing) GetAllEvents() map[string]*InteractionOutput {
	return ct.eventValues
}

// GetAllProperties returns all Thing property values and returns them in a
// map of InteractionOutputs.
func (ct *ConsumedThing) GetAllProperties() map[string]*InteractionOutput {
	return ct.propValues
}

// InvokeAction requests an action on the Thing
// This updates the action status with the response. For async actions the owner
// of the Consumer must invoke OnResponse to update the action status.
func (ct *ConsumedThing) InvokeAction(name string, iin InteractionInput) (*InteractionOutput, error) {

	req := messaging.NewRequestMessage(
		wot.OpInvokeAction, ct.ThingID, name, iin.Value.Raw, "")
	resp, err := ct.co.SendRequest(req, true)
	if err != nil {
		return nil, err
	}
	// update the
	iout := NewInteractionOutput(ct,
		messaging.AffordanceTypeAction, name, resp.Value, resp.Timestamp)
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

// OnNotification handles receiving a Thing event or property update.
// To be called by the manager of this ConsumedThing, the one that receives
// all subscribed events from the hub client.
// This updates the latest event value and invokes the registered event subscriber, if any.
//
//	msg is the notification message received.
func (ct *ConsumedThing) OnNotification(notif *messaging.NotificationMessage) {

	if notif.Operation == wot.OpSubscribeEvent &&
		notif.ThingID == digitwin.ThingDirectoryDThingID &&
		notif.Name == digitwin.ThingDirectoryEventThingUpdated {
		// decode the TD
		tdi := &td.TD{}
		err := jsoniter.UnmarshalFromString(notif.ToString(0), &tdi)
		if err != nil {
			slog.Error("invalid payload for TD event. Ignored",
				"thingID", notif.ThingID)
			return
		}
		// update consumed thing, if existing
		ct.mux.Lock()
		ct.tdi = tdi
		ct.mux.Unlock()
	} else if notif.Operation == wot.OpObserveProperty {
		// update value
		iout := NewInteractionOutputFromNotification(
			ct, messaging.AffordanceTypeProperty, notif)
		ct.mux.Lock()
		ct.propValues[notif.Name] = iout
		// the consumed thing title and description are updated with corresponding properties
		if notif.Name == wot.WoTTitle {
			ct.Title = iout.Value.Text()
		} else if notif.Name == wot.WoTDescription {
			ct.Description = iout.Value.Text()
		}
		ct.mux.Unlock()

		subscr, _ := ct.subscribers[notif.Name]
		if subscr != nil {
			subscr(iout)
		}
	} else if notif.Operation == wot.OpSubscribeEvent {
		iout := NewInteractionOutputFromNotification(ct, messaging.AffordanceTypeEvent, notif)
		// this is a regular value event
		ct.mux.Lock()
		ct.eventValues[notif.Name] = iout
		ct.mux.Unlock()
		subscr, _ := ct.subscribers[notif.Name]
		if subscr != nil {
			subscr(iout)
		}
	} else if notif.Operation == wot.OpInvokeAction {
		iout := NewInteractionOutputFromNotification(ct, messaging.AffordanceTypeAction, notif)
		// this is a regular action progress event
		ct.mux.Lock()
		ct.actionOutputs[notif.Name] = iout
		ct.mux.Unlock()
		subscr, _ := ct.subscribers[notif.Name]
		if subscr != nil {
			subscr(iout)
		}
	}
}

// QueryAction queries the action status record from the hub and update the cache.
// If no action exists then actionstatus will be mostly empty
//
// This returns an empty ActionStatus if not found
func (ct *ConsumedThing) QueryAction(name string) messaging.ActionStatus {
	as, _ := ct.co.QueryAction(ct.ThingID, name)
	iin := ct.GetActionInputFromStatus(as)
	iout := ct.GetActionOutputFromStatus(as)
	ct.actionOutputs[name] = iout
	ct.actionInputs[name] = iin
	return as
}

// ReadEvent refreshes the last event value by reading it from the hub
// TODO: subscribing to events should send the last one
func (ct *ConsumedThing) ReadEvent(name string) *InteractionOutput {

	tv, err := digitwin.ThingValuesReadEvent(ct.co, name, ct.ThingID)
	if err != nil {
		return nil
	}
	iout := NewInteractionOutput(ct, messaging.AffordanceTypeEvent, name, tv.Data, tv.Timestamp)
	//iout.setSchemaFromTD(ct.tdi)
	ct.mux.Lock()
	ct.eventValues[name] = iout
	ct.mux.Unlock()
	return iout
}

// ReadProperty reads the Thing property value from the Thing and updates its cache.
// Call GetPropertyValue to get the cached value.
func (ct *ConsumedThing) ReadProperty(name string) *InteractionOutput {

	resp, err := ct.co.ReadProperty(ct.ThingID, name)
	if err != nil {
		return nil
	}
	iout := NewInteractionOutput(ct, messaging.AffordanceTypeProperty, name, resp.Data, resp.Timestamp)
	ct.mux.Lock()
	ct.propValues[name] = iout
	ct.mux.Unlock()
	return iout
}

// Refresh reloads all property and event values from the Hub and updates the
// cache.
// This also updates the Thing title and description if they have corresponding
// properties.
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
			iout = NewInteractionOutputFromValue(ct, messaging.AffordanceTypeEvent, tv)
		} else {
			iout = NewInteractionOutput(ct, messaging.AffordanceTypeEvent, name, nil, "")
		}
		ct.eventValues[name] = iout
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
			iout = NewInteractionOutputFromValue(ct, messaging.AffordanceTypeProperty, tv)
			// the consumed thing title and description can be modified with corresponding properties
			if name == wot.WoTTitle {
				ct.Title = iout.Value.Text()
			} else if name == wot.WoTDescription {
				ct.Description = iout.Value.Text()
			}
		} else {
			iout = NewInteractionOutput(
				ct, messaging.AffordanceTypeProperty, name, nil, "")
		}
		ct.propValues[name] = iout
	}
	// refresh action status
	actionStatusMap, err := ct.co.QueryAllActions(ct.ThingID)
	if err != nil {
		return err
	}
	for name, _ := range ct.tdi.Actions {
		as, found := actionStatusMap[name]
		if found {
			// if the TD doesn't have this action then ignore it
			actionAff := ct.tdi.GetAction(name)
			if actionAff != nil {
				iout := NewInteractionOutputFromActionStatus(ct, as)
				ct.actionOutputs[name] = iout
			}
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

// TD return the TD document that is represented here.
func (ct *ConsumedThing) TD() *td.TD {
	return ct.tdi
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
func NewConsumedThing(tdi *td.TD, co *messaging.Consumer) *ConsumedThing {
	ct := ConsumedThing{
		ThingID: tdi.ID,
		// title and description are updated with corresponding properties if they exist
		Title:         tdi.Title,
		Description:   tdi.Description,
		tdi:           tdi,
		co:            co,
		observers:     make(map[string]InteractionListener),
		subscribers:   make(map[string]InteractionListener),
		actionInputs:  make(map[string]*InteractionInput),
		actionOutputs: make(map[string]*InteractionOutput),
		eventValues:   make(map[string]*InteractionOutput),
		propValues:    make(map[string]*InteractionOutput),
	}
	return &ct
}
