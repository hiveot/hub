package consumedthing

import (
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/services/history/historyclient"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"sync"
	"time"
)

// PropertyReadMap maps property keys to their InteractionOutput object that
// represents the property value.
// Intended for reading properties.
//type PropertyReadMap map[string]*InteractionOutput

// PropertyWriteMap maps property keys to their InteractionInput object for
// writing property values.
//type PropertyWriteMap map[string]*InteractionInput

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
	hc hubclient.IHubClient

	td *tdd.TD
	// observer of property value changes by property name
	observers map[string]InteractionListener
	// subscribers to events by eventName
	subscribers map[string]InteractionListener

	// action status values
	actionValues map[string]hubclient.DeliveryStatus
	// prop values
	propValues hubclient.ThingMessageMap
	// event values
	eventValues hubclient.ThingMessageMap

	mux sync.RWMutex
}

// build a map of interaction outputs for the given messages
func (ct *ConsumedThing) buildInteractionOutputMap(tmm hubclient.ThingMessageMap) map[string]*InteractionOutput {
	outMap := make(map[string]*InteractionOutput)
	for key, tm := range tmm {
		iout := NewInteractionOutputFromTM(tm, ct.td)
		outMap[key] = iout
	}
	return outMap
}

// Create an interactionOutput for the given thing message
func (ct *ConsumedThing) buildInteractionOutput(tm *hubclient.ThingMessage) *InteractionOutput {
	iout := NewInteractionOutputFromTM(tm, ct.td)
	return iout
}

// GetValue returns the interaction output of the latest value of an event, property or
// action output.
//
// This returns an empty interactionoutput if not found
func (ct *ConsumedThing) GetValue(name string) (*InteractionOutput, bool) {
	tm, found := ct.eventValues[name]
	if !found {
		tm, found = ct.propValues[name]
	}
	if tm == nil {
		tm = &hubclient.ThingMessage{
			ThingID: ct.td.ID,
			Name:    name,
		} // dummy
		slog.Warn("Value not (yet) found for name ", "name", name, "thingID", ct.td.ID)
	}
	// add the dataschema for the value
	iout := ct.buildInteractionOutput(tm)
	return iout, found
}

// GetThingDescription return the TD document that is represented here.
func (ct *ConsumedThing) GetThingDescription() *tdd.TD {
	return ct.td
}

// InvokeAction requests an action on the thing
func (ct *ConsumedThing) InvokeAction(name string, params InteractionInput) *InteractionOutput {
	aff := ct.td.GetAction(name)
	if aff == nil {
		return nil
	}
	// find the form that describes the protocol for invoking an action
	actionForm := ct.td.GetForm(vocab.WotOpInvokeAction, name, ct.hc.GetProtocolType())

	tm := hubclient.NewThingMessage(
		vocab.MessageTypeAction, ct.td.ID, name, params, ct.hc.ClientID())
	//
	////stat := ct.hc.InvokeAction(ct.td.ID, name, params)
	urlParams := map[string]string{
		"thingID": ct.td.ID,
		"name":    name,
	}
	href, err := ct.td.GetFormHRef(actionForm, urlParams)
	if err != nil {
		slog.Warn("InvokeAction", "err", err.Error())
	}
	stat := ct.hc.SendOperation(href, actionForm, params)

	o := NewInteractionOutputFromTM(tm, ct.td)
	o.Progress = stat
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
func (ct *ConsumedThing) OnDeliveryUpdate(msg *hubclient.ThingMessage) {
	action, found := ct.actionValues[msg.Name]
	_ = action
	if !found {
		slog.Error("Action update without action?",
			"thingID", msg.ThingID,
			"action", msg.Name)
		return
	}
	stat := hubclient.DeliveryStatus{}
	err := utils.DecodeAsObject(msg.Data, &stat)
	if stat.Error != "" {
		slog.Error("Delivery update invalid payload",
			"thingID", msg.ThingID,
			"action", msg.Name,
			"err", err.Error())
	}
	ct.actionValues[msg.Name] = stat
}

// OnEvent handles receiving of an event.
// To be called by the manager of this ConsumerThing, the one that receives
// all subscribed events from the hub client.
// This updates the latest event value and invokes the registered event subscriber, if any.
//
//	tm is the event message received from the hub. This isn't standard WoT so
//	the objective is to remove the need for it.
func (ct *ConsumedThing) OnEvent(msg *hubclient.ThingMessage) {
	ct.eventValues[msg.Name] = msg
	subscr, found := ct.subscribers[msg.Name]
	if found {
		io := ct.buildInteractionOutput(msg)
		subscr(io)
	}
}

// OnPropertyUpdate handles receiving of a property value update.
// To be called by the manager of this ConsumerThing, the one that receives
// all subscribed events from the hub client.
// This updates the latest property value and invokes the registered property observer, if any.
//
//	msg is the property message received from the hub. This isn't standard WoT so
//	the objective is to remove the need for it.
func (ct *ConsumedThing) OnPropertyUpdate(msg *hubclient.ThingMessage) {
	ct.propValues[msg.Name] = msg
	observer, found := ct.observers[msg.Name]
	if found {
		io := ct.buildInteractionOutput(msg)
		observer(io)
	}
}

// ReadEvent returns the last known Thing event value
// Call ReadAllEvents to refresh the values.
func (ct *ConsumedThing) ReadEvent(name string) *InteractionOutput {
	tm := ct.eventValues.Get(name)
	if tm == nil {
		return nil
	}
	io := ct.buildInteractionOutput(tm)
	return io
}

// ReadHistory returns the history for the given name
// If the number of values exceed the maximum then this returns itemsRemaining
// as true. An additional call can be made using the last returned timestamp to
// get the remaining values.
func (ct *ConsumedThing) ReadHistory(name string, timestamp time.Time, duration time.Duration) (values []*hubclient.ThingMessage, itemsRemaining bool, err error) {

	hist := historyclient.NewReadHistoryClient(ct.hc)
	values, itemsRemaining, err = hist.ReadHistory(
		ct.td.ID, name, timestamp, duration, 0)

	return values, itemsRemaining, err
}

// ReadProperty returns the last known Thing property value
// Call ReadAllProperties to refresh the property values.
func (ct *ConsumedThing) ReadProperty(name string) *InteractionOutput {

	aff := ct.td.GetProperty(name)
	if aff == nil {
		return nil
	}
	tm := ct.propValues.Get(name)
	o := ct.buildInteractionOutput(tm)
	return o
}

// ReadAllEvents reads all Thing event and action values.
func (ct *ConsumedThing) ReadAllEvents() map[string]*InteractionOutput {
	tmsJson, err := digitwin.OutboxReadLatest(
		ct.hc, vocab.MessageTypeEvent, nil, "", ct.td.ID)
	if err != nil {
		return nil
	}
	ct.eventValues, err = hubclient.NewThingMessageMapFromSource(tmsJson)
	if err != nil {
		return nil
	}
	return ct.buildInteractionOutputMap(ct.eventValues)
}

// ReadAllProperties reads all Thing property values and returns them in a
// map of InteractionOutputs.
func (ct *ConsumedThing) ReadAllProperties() map[string]*InteractionOutput {
	tmsJson, err := digitwin.OutboxReadLatest(
		ct.hc, vocab.MessageTypeProperty, nil, "", ct.td.ID)
	if err != nil {
		return nil
	}
	ct.propValues, err = hubclient.NewThingMessageMapFromSource(tmsJson)
	if err != nil {
		return nil
	}
	return ct.buildInteractionOutputMap(ct.propValues)
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
// This takes place asynchronously.
// cb is invoked with the InteractionOutput containing the delivery progress.
//
// Since writing a property can take some time, especially if the device is
// asleep, the callback receives the first response containing a messageID.
// If the request is not yet complete
func (ct *ConsumedThing) WriteProperty(name string, value InteractionInput) hubclient.DeliveryStatus {

	stat := ct.hc.PubProperty(ct.td.ID, name, value)
	// TODO: receive updates
	return stat
}

// WriteMultipleProperties requests a change to multiple property values.
// This takes place asynchronously.
// cb is invoked with the InteractionOutput containing the delivery progress.
func (ct *ConsumedThing) WriteMultipleProperties(
	values map[string]InteractionInput, cb InteractionListener) {

}

// NewConsumedThing creates a new instance of a Thing
// Call Stop() when done
func NewConsumedThing(td *tdd.TD, hc hubclient.IHubClient) *ConsumedThing {
	c := ConsumedThing{
		td:           td,
		hc:           hc,
		observers:    make(map[string]InteractionListener),
		subscribers:  make(map[string]InteractionListener),
		actionValues: make(map[string]hubclient.DeliveryStatus),
		eventValues:  make(map[string]*hubclient.ThingMessage),
		propValues:   make(map[string]*hubclient.ThingMessage),
	}
	return &c
}