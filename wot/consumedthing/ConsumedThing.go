package consumedthing

import (
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/services/history/historyclient"
	transports2 "github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
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
	cc transports2.IClientConnection

	td *td.TD
	// observer of property value changes by property name
	observers map[string]InteractionListener
	// subscribers to events by eventName
	subscribers map[string]InteractionListener

	// action status values
	actionValues map[string]*transports2.RequestStatus
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
	err := ct.cc.SendRequest(op, thingID, name, input, output)
	return err
}

// build a map of interaction outputs for the given values
func (ct *ConsumedThing) buildInteractionOutputMap(tvm map[string]*transports2.ThingMessage) map[string]*InteractionOutput {
	outMap := make(map[string]*InteractionOutput)
	for key, tv := range tvm {
		iout := NewInteractionOutputFromValue(tv, ct.td)
		outMap[key] = iout
	}
	return outMap
}

// Create an interactionOutput for the given thing message
func (ct *ConsumedThing) buildInteractionOutput(tv *transports2.ThingMessage) *InteractionOutput {
	iout := NewInteractionOutputFromValue(tv, ct.td)
	return iout
}

// GetPropValue returns the interaction output of the latest property value.
//
// This returns an empty InteractionOutput if not found
func (ct *ConsumedThing) GetPropValue(name string) (iout *InteractionOutput) {
	iout, found := ct.propValues[name]
	_ = found
	if iout == nil {
		// not a known prop value so create an empty io with a schema from the td
		iout = &InteractionOutput{
			ThingID: ct.td.ID,
			Name:    name,
		}
		iout.SetSchemaFromTD(ct.td)
		slog.Debug("Value not (yet) published for property ", "name", name, "thingID", ct.td.ID)
	}
	return iout
}

// GetEventValue returns the interaction output of the latest value of an event
//
// This returns an empty InteractionOutput if not found
func (ct *ConsumedThing) GetEventValue(name string) (iout *InteractionOutput) {
	iout, found := ct.eventValues[name]
	_ = found
	if iout == nil {
		// not a known event value so create an empty io with a schema from the td
		iout = &InteractionOutput{
			ThingID: ct.td.ID,
			Name:    name,
		}
		iout.SetSchemaFromTD(ct.td)
		slog.Info("Value not (yet) found for event ", "name", name, "thingID", ct.td.ID)
	}
	return iout
}

// GetThingDescription return the TD document that is represented here.
func (ct *ConsumedThing) GetThingDescription() *td.TD {
	return ct.td
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
	err := ct._rpc(wot.OpInvokeAction, name, params.value, &output)

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
// This updates the action progress value and invokes the action callback, if any.
//
//	tm is the event message received from the hub. This isn't standard WoT so
//	the objective is to remove the need for it.
func (ct *ConsumedThing) OnDeliveryUpdate(msg *transports2.ThingMessage) {
	action, found := ct.actionValues[msg.Name]
	_ = action
	if !found {
		slog.Error("Action update without action?",
			"thingID", msg.ThingID,
			"action", msg.Name)
		return
	}
	stat := transports2.RequestStatus{}
	err := tputils.DecodeAsObject(msg.Data, &stat)
	if stat.Error != "" {
		slog.Error("Delivery update invalid payload",
			"thingID", msg.ThingID,
			"action", msg.Name,
			"err", err.Error())
	}
	ct.actionValues[msg.Name] = &stat
}

// OnEvent handles receiving of an event.
// To be called by the manager of this ConsumerThing, the one that receives
// all subscribed events from the hub client.
// This updates the latest event value and invokes the registered event subscriber, if any.
//
//	tm is the event message received from the hub. This isn't standard WoT so
//	the objective is to remove the need for it.
func (ct *ConsumedThing) OnEvent(tv *transports2.ThingMessage) {
	io := ct.buildInteractionOutput(tv)
	ct.eventValues[tv.Name] = io
	subscr, found := ct.subscribers[tv.Name]
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
func (ct *ConsumedThing) OnPropertyUpdate(msg *transports2.ThingMessage) {
	io := ct.buildInteractionOutput(msg)
	ct.propValues[msg.Name] = io
	observer, found := ct.observers[msg.Name]
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
	//// if there is no known value, read it now
	//if iout == nil {
	//	ct.mux.RLock()
	//	aff, _ := ct.td.Events[name]
	//	ct.mux.RUnlock()
	//	if aff == nil {
	//		return nil
	//	}
	//	td := ct.GetThingDescription()
	//	var raw any
	//	err := ct._rpc(wot.HTOpReadEvent, name, nil, &raw)
	//	iout := NewInteractionOutput(ct.td.ID, name, schema, raw, "")
	//	iout.SetSchemaFromTD(ct.td)
	//
	//	//tv, err := ct.cc.ReadEvent(ct.td.ID, name)
	//	form := td.GetForm(wot.HTOpReadEvent, name, ct.cc.GetProtocolType())
	//	var eventValue any
	//	err := ct.cc.SendRequest(form, ct.td.ID, name, nil, &eventValue)
	//
	//	if err == nil {
	//		iout = NewInteractionOutputFromValue(&tv, td)
	//	} else {
	//		iout = NewInteractionOutput(td.ID, name, aff.Data, nil, "")
	//	}
	//	ct.mux.Lock()
	//	ct.eventValues[name] = iout
	//	ct.mux.Unlock()
	//}
	return iout
}

// ReadHistory returns the history for the given name
// If the number of values exceed the maximum then this returns itemsRemaining
// as true. An additional call can be made using the last returned timestamp to
// get the remaining values.
func (ct *ConsumedThing) ReadHistory(
	name string, timestamp time.Time, duration time.Duration) (
	values []*transports2.ThingMessage, itemsRemaining bool, err error) {

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
	//if iout == nil {
	//	ct.mux.RLock()
	//	aff, _ := ct.td.Properties[name]
	//	ct.mux.RUnlock()
	//	if aff == nil {
	//		return nil
	//	}
	//	td := ct.GetThingDescription()
	//	tv, err := digitwin.ValuesReadProperty(ct.cc, ct.td.ID, name)
	//	if err == nil {
	//		iout = NewInteractionOutputFromValue(&tv, td)
	//	} else {
	//		iout = NewInteractionOutput(td.ID, name, &aff.DataSchema, nil, "")
	//	}
	//	ct.mux.Lock()
	//	ct.eventValues[name] = iout
	//	ct.mux.Unlock()
	//}
	return iout
}

// ReadAllEvents reads all Thing event values.
func (ct *ConsumedThing) ReadAllEvents() map[string]*InteractionOutput {
	var err error
	var evList []digitwin.ThingValue

	err = ct._rpc(wot.HTOpReadAllEvents, "", nil, &evList)
	if err != nil {
		return nil
	}
	for _, v := range evList {
		io := NewInteractionOutput(ct.td, AffordanceTypeEvent, v.Name, v.Data, v.Updated)
		ct.eventValues[v.Name] = io
	}
	return ct.eventValues
}

// ReadAllProperties reads all Thing property values and returns them in a
// map of InteractionOutputs.
func (ct *ConsumedThing) ReadAllProperties() map[string]*InteractionOutput {
	propList, err := digitwin.ValuesReadAllProperties(ct.cc, ct.td.ID)
	if err != nil {
		return nil
	}
	for _, v := range propList {
		io := NewInteractionOutput(ct.td, AffordanceTypeProperty, v.Name, v.Data, v.Updated)
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
func (ct *ConsumedThing) WriteProperty(name string, value InteractionInput) (err error) {

	//just a simple wrapper around the transport client
	thingID := ct.td.ID
	raw := value.value
	err = ct.cc.SendNotification(wot.OpWriteProperty, thingID, name, raw)

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
func NewConsumedThing(td *td.TD, hc transports2.IClientConnection) *ConsumedThing {
	c := ConsumedThing{
		td:           td,
		cc:           hc,
		observers:    make(map[string]InteractionListener),
		subscribers:  make(map[string]InteractionListener),
		actionValues: make(map[string]*transports2.RequestStatus),
		eventValues:  make(map[string]*InteractionOutput),
		propValues:   make(map[string]*InteractionOutput),
	}
	return &c
}
