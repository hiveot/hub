package consumedthing

import (
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/services/history/historyclient"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"sync"
	"time"
)

// InteractionListener is the handler that receives updates to interaction
// requests, eg write property, invoke action or subscribe to events.
type InteractionListener func(*InteractionOutput)

const AffordanceTypeEvent = "event"
const AffordanceTypeProperty = "property"
const AffordanceTypeAction = "action"

// ConsumedThing implements the ConsumedThing interface for accessing a Thing's
// schema and values roughly in line with the WoT scripting API.
//
// However, since the scripting API is based on Javascript some differences are
// made to accommodate the different environment.
//
// This keeps a copy of the Thing's property and event values and updates on changes.
type ConsumedThing struct {
	cc transports.IConsumerConnection

	td *td.TD
	// observer of property value changes by property name
	observers map[string]InteractionListener
	// subscribers to events by eventName
	subscribers map[string]InteractionListener

	// action status values
	//actionValues map[string]*transports.ActionStatus
	// prop values
	propValues map[string]*InteractionOutput
	// event values
	eventValues map[string]*InteractionOutput

	mux sync.RWMutex
}

// Perform an operation on this Thing and affordance.
// This looks up the form in the TD for this transport's protocol, marshals the input,
// sends the request and waits for a completed or failed response.
func (ct *ConsumedThing) _rpc(op string, name string, input interface{}, output interface{}) error {
	//just a simple wrapper around the transport client
	thingID := ct.td.ID
	err := ct.cc.Rpc(op, thingID, name, input, output)
	return err
}

// build a map of interaction outputs for the given values
func (ct *ConsumedThing) buildInteractionOutputMap(
	tmm map[string]*transports.NotificationMessage) map[string]*InteractionOutput {

	outMap := make(map[string]*InteractionOutput)
	for key, msg := range tmm {
		iout := NewInteractionOutputFromMessage(msg, ct.td)
		outMap[key] = iout
	}
	return outMap
}

// Create an interactionOutput for the given thing message
func (ct *ConsumedThing) buildInteractionOutput(tm *transports.NotificationMessage) *InteractionOutput {
	iout := NewInteractionOutputFromMessage(tm, ct.td)
	return iout
}

// GetActionStatus returns the action status record of the last action
//
// This returns an empty ActionStatus if not found
func (ct *ConsumedThing) GetActionStatus(name string) *digitwin.ActionStatus {
	var stat digitwin.ActionStatus
	err := ct._rpc(wot.OpQueryAction, name, nil, &stat)

	//ct.InvokeAction(digitwin.ValuesActionQueryAction, args)
	//actionVal, err := digitwin.ValuesQueryAction(sess.GetHubClient(), name, thingID)
	//err = hc.Rpc("invokeaction", ValuesDThingID, ValuesQueryActionMethod, &args, &actionvalue)
	_ = err
	return &stat
}

// GetEventValue returns the interaction output of the latest value of an event
// See also GetValue that always return an iout (for rendering purpose)
//
// This returns nil if not found
func (ct *ConsumedThing) GetEventValue(name string) (iout *InteractionOutput) {
	ct.mux.RLock()
	iout, _ = ct.eventValues[name]
	ct.mux.RUnlock()
	return iout
}

// GetPropValue returns the interaction output of the latest property value.
// See also GetValue that always return an iout (for rendering purpose)
//
// This returns nil if not found
func (ct *ConsumedThing) GetPropValue(name string) (iout *InteractionOutput) {
	ct.mux.RLock()
	iout, _ = ct.propValues[name]
	ct.mux.RUnlock()
	return iout
}

// GetThingDescription return the TD document that is represented here.
func (ct *ConsumedThing) GetThingDescription() *td.TD {
	return ct.td
}

// GetTitle return the TD document title
// If a title property is available return its value instead of the TD title.
// This lets a Thing update its TD title without re-issuing a new TD.
func (ct *ConsumedThing) GetTitle() string {
	title := ct.td.Title
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
	ct.mux.RUnlock()
	_ = found
	if iout == nil {
		// not a known prop or event value so create an empty io with a schema from the td
		iout = &InteractionOutput{
			ThingID: ct.td.ID,
			Name:    name,
		}
		iout.setSchemaFromTD(ct.td)
	}
	return iout
}

// InvokeAction requests an action on the Thing
func (ct *ConsumedThing) InvokeAction(name string, params InteractionInput) *InteractionOutput {
	var output any
	aff := ct.td.GetAction(name)
	if aff == nil {
		return nil
	}
	// find the form that describes the protocol for invoking an action
	//stat := ct.cc.PublishFromForm(href, actionForm, params.value, "")
	err := ct._rpc(wot.OpInvokeAction, name, params.Value.Raw, &output)

	o := NewInteractionOutput(ct.td, AffordanceTypeAction, name, output, "")
	o.Err = err
	return o
}

// ObserveProperty registers a handler to changes in property value.
// Only a single subscription per property is allowed. This returns an error
// if an existing observer is already registered.
func (ct *ConsumedThing) ObserveProperty(name string, listener InteractionListener) error {
	if _, found := ct.observers[name]; found {
		return fmt.Errorf("A property observer is already registered")
	}
	ct.observers[name] = listener
	return nil
}

// OnDeliveryUpdate handles receiving of an action progress event.
// To be called by the manager of this ConsumerThing, the one that receives
// all subscribed messages from the hub client.
// FIXME: delivery updates are no longer events/notifications
// This updates the action progress value and invokes the action callback, if any.
//
//	tm is the event message received from the hub. This isn't standard WoT so
//	the objective is to remove the need for it.
//func (ct *ConsumedThing) OnDeliveryUpdate(msg transports.ResponseMessage) {
//	action, found := ct.actionValues[msg.Name]
//	_ = action
//	if !found {
//		slog.Error("Action update without action?",
//			"thingID", msg.ThingID,
//			"action", msg.Name)
//		return
//	}
//	stat := transports.ActionStatus{}
//	err := tputils.DecodeAsObject(msg.Output, &stat)
//	if stat.Error != "" {
//		slog.Error("Delivery update invalid payload",
//			"thingID", msg.ThingID,
//			"action", msg.Name,
//			"err", err.Error())
//	}
//	ct.actionValues[msg.Name] = &stat
//}

// OnEvent handles receiving of an event.
// To be called by the manager of this ConsumerThing, the one that receives
// all subscribed events from the hub client.
// This updates the latest event value and invokes the registered event subscriber, if any.
//
//	tm is the event message received from the hub. This isn't standard WoT so
//	the objective is to remove the need for it.
func (ct *ConsumedThing) OnEvent(msg transports.NotificationMessage) {
	io := ct.buildInteractionOutput(&msg)
	ct.mux.Lock()
	ct.eventValues[msg.Name] = io
	subscr, found := ct.subscribers[msg.Name]
	ct.mux.Unlock()

	if found {
		subscr(io)
	}
}

// OnPropertyUpdate handles receiving of a property value update.
// To be called by the manager of this ConsumerThing, the one that receives
// all subscribed events from the client.
// This updates the latest property value and invokes the registered property observer, if any.
//
//	msg is the property message received from the hub. This isn't standard WoT so
//	the objective is to remove the need for it.
func (ct *ConsumedThing) OnPropertyUpdate(msg transports.NotificationMessage) {
	io := ct.buildInteractionOutput(&msg)
	ct.mux.Lock()
	ct.propValues[msg.Name] = io
	observer, found := ct.observers[msg.Name]
	ct.mux.Unlock()
	if found {
		observer(io)
	}
}

// OnTDUpdate handles receiving of an update to the TD document.
// This affects newly created interaction outputs which will be created with
// the updated affordance schema.
//
//	msg is the property message received from the hub. This isn't standard WoT so
//	the objective is to remove the need for it.
func (ct *ConsumedThing) OnTDUpdate(newTD *td.TD) {
	ct.mux.Lock()
	defer ct.mux.Unlock()
	ct.td = newTD
}

// ReadEvent returns the last known Thing event value or nil if name is not an event
// Call ReadAllEvents to refresh the values.
//
// If no value is yet known then create an affordance and read a value.
func (ct *ConsumedThing) ReadEvent(name string) *InteractionOutput {
	ct.mux.RLock()
	iout, _ := ct.eventValues[name]
	ct.mux.RUnlock()
	// if there is no known value, read it now
	if iout == nil {
		ct.mux.RLock()
		aff, _ := ct.td.Events[name]
		ct.mux.RUnlock()
		if aff == nil {
			return nil // not a known event
		}
		tdi := ct.GetThingDescription()
		var raw any
		err := ct._rpc(wot.HTOpReadEvent, name, nil, &raw)
		_ = err
		iout = NewInteractionOutput(tdi, AffordanceTypeEvent, name, raw, "")
		iout.setSchemaFromTD(ct.td)

		//tv, err := ct.cc.ReadEvent(ct.td.ID, name)
		//form := td.GetForm(wot.HTOpReadEvent, name, ct.cc.GetProtocolType())
		//	var eventValue any
		//	err := ct.cc.SendRequest(form, ct.td.ID, name, nil, &eventValue)
		//
		//if err == nil {
		//	iout = NewInteractionOutputFromMessage(&tv, td)
		//} else {
		//	iout = NewInteractionOutput(td.ID, name, aff.Data, nil, "")
		//}
		ct.mux.Lock()
		ct.eventValues[name] = iout
		ct.mux.Unlock()
	}
	return iout
}

// ReadHistory returns the history for the given name
// If the number of values exceed the maximum then this returns itemsRemaining
// as true. An additional call can be made using the last returned timestamp to
// get the remaining values.
func (ct *ConsumedThing) ReadHistory(
	name string, timestamp time.Time, duration time.Duration) (
	values []*transports.NotificationMessage, itemsRemaining bool, err error) {

	// FIXME: ReadHistory is not (yet) part of the WoT specification. Ege mentioned it would
	// be added soon so this will change to follow the WoT specification.
	// Until then this is tied to the Hub's history service.

	hist := historyclient.NewReadHistoryClient(ct.cc)
	// todo: is there a need to read in batches? not for a single day.
	values, itemsRemaining, err = hist.ReadHistory(
		ct.td.ID, name, timestamp, duration, 500)

	return values, itemsRemaining, err
}

// ReadProperty returns the last known Thing property value or nil if name is not a property
// Call ReadAllProperties to refresh the property values.
func (ct *ConsumedThing) ReadProperty(name string) *InteractionOutput {
	ct.mux.RLock()
	iout, _ := ct.propValues[name]
	ct.mux.RUnlock()
	return iout
}

// ReadAllEvents reads all Thing event values.
func (ct *ConsumedThing) ReadAllEvents() map[string]*InteractionOutput {
	var err error
	var evList []transports.NotificationMessage
	ct.mux.Lock()
	defer ct.mux.Unlock()
	// FIXME: this sometimes returns actions as events
	//  zwavejs D0547D32.2 37-targetValue-0 (binary switch)
	// options for graceful recovery:
	//   1. show 'unknown event' (current)
	//   2. use action/property output schema if affordance is an action name
	//   3.
	err = ct._rpc(wot.HTOpReadAllEvents, "", nil, &evList)
	if err != nil {
		return nil
	}
	for _, tm := range evList {
		io := NewInteractionOutput(
			ct.td, AffordanceTypeEvent, tm.Name, tm.Data, tm.Created)
		ct.eventValues[tm.Name] = io
	}
	return ct.eventValues
}

// ReadAllProperties reads all Thing property values and returns them in a
// map of InteractionOutputs.
func (ct *ConsumedThing) ReadAllProperties() map[string]*InteractionOutput {
	var propList []transports.NotificationMessage

	//propList, err := digitwin.ValuesReadAllProperties(ct.cc, ct.td.ID)
	err := ct._rpc(wot.OpReadAllProperties, "", nil, &propList)
	if err != nil {
		return nil
	}
	ct.mux.Lock()
	defer ct.mux.Unlock()

	for _, v := range propList {
		io := NewInteractionOutput(
			ct.td, AffordanceTypeProperty, v.Name, v.Data, v.Created)
		ct.propValues[v.Name] = io
	}
	return ct.propValues
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
// Writing a property can take some time, especially if the device is asleep. This
// operation is therefore asynchronously and returns when delivery to the Thing is
// successful.
//
// In case of a hiveot this means delivery to the digital twin, not delivery to
// the actual device.
//
// Note that WoT does not specify a mechanism to confirm the success or failure of
// delivery and applying the value.
//
// This returns a correlation ID and an error if the request cannot be delivered to the server.
func (ct *ConsumedThing) WriteProperty(name string, ii InteractionInput) (err error) {

	//just a simple wrapper around the transport client
	thingID := ct.td.ID
	raw := ii.Value.Raw
	err = ct.cc.WriteProperty(thingID, name, raw, true)

	return err
}

// WriteMultipleProperties requests a change to multiple property values.
// This takes place asynchronously.
// cb is invoked with the InteractionOutput containing the delivery progress.
func (ct *ConsumedThing) WriteMultipleProperties(
	values map[string]InteractionInput, cb InteractionListener) {

}

// NewConsumedThing creates a new instance of a Thing
// Call Stop() when done
func NewConsumedThing(td *td.TD, hc transports.IConsumerConnection) *ConsumedThing {
	c := ConsumedThing{
		td:          td,
		cc:          hc,
		observers:   make(map[string]InteractionListener),
		subscribers: make(map[string]InteractionListener),
		//actionValues: make(map[string]*transports.ActionStatus),
		eventValues: make(map[string]*InteractionOutput),
		propValues:  make(map[string]*InteractionOutput),
	}
	return &c
}
