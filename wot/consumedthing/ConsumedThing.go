package consumedthing

import (
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
)

type InteractionInput any //interface {}

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
	// observer of property value changes
	observers map[string]InteractionListener
	// subscribers to events
	subscribers map[string]InteractionListener

	// action status values
	actionValues map[string]hubclient.DeliveryStatus
	// props values
	propValues hubclient.ThingMessageMap
	// event values
	eventValues hubclient.ThingMessageMap
}

// build a map
func (c *ConsumedThing) buildInteractionOutputMap(tmm hubclient.ThingMessageMap) map[string]*InteractionOutput {
	outMap := make(map[string]*InteractionOutput)
	for key, tm := range tmm {
		iout := NewInteractionOutput(tm, c.td)
		outMap[key] = iout
	}
	return outMap
}

// Create an interactionOutput for the given thing message
func (c *ConsumedThing) buildInteractionOutput(tm *hubclient.ThingMessage) *InteractionOutput {
	iout := NewInteractionOutput(tm, c.td)
	return iout
}

// GetThingDescription return the TD document that is represented here.
func (c *ConsumedThing) GetThingDescription() *tdd.TD {
	return c.td
}

// InvokeAction requests an action on the thing
func (c *ConsumedThing) InvokeAction(
	key string, params InteractionInput) *InteractionOutput {
	aff := c.td.GetAction(key)
	if aff == nil {
		return nil
	}
	tm := hubclient.NewThingMessage(
		vocab.MessageTypeAction, c.td.ID, key, params, c.hc.ClientID())

	stat := c.hc.PubAction(c.td.ID, key, params)
	o := NewInteractionOutput(tm, c.td)
	o.Progress = stat
	return o
}

// ObserveProperty registers a handler to changes in property value.
// Only a single subscription per property is allowed. This returns an error
// if an existing observer is already registered.
func (c *ConsumedThing) ObserveProperty(key string, listener InteractionListener) error {
	if _, found := c.observers[key]; found {
		return fmt.Errorf("A property observer is already registered")
	}
	c.observers[key] = listener
	return nil
}

// OnDeliveryUpdate handles receiving of an action progress event.
// To be called by the manager of this ConsumerThing, the one that receives
// all subscribed messages from the hub client.
// This updates the action progress value and invokes the action callback, if any.
//
//	tm is the event message received from the hub. This isn't standard WoT so
//	the objective is to remove the need for it.
func (c *ConsumedThing) OnDeliveryUpdate(msg *hubclient.ThingMessage) {
	action, found := c.actionValues[msg.Key]
	_ = action
	if !found {
		slog.Error("Action update without action?",
			"thingID", msg.ThingID,
			"action", msg.Key)
		return
	}
	stat := hubclient.DeliveryStatus{}
	err := utils.DecodeAsObject(msg.Data, &stat)
	if stat.Error != "" {
		slog.Error("Delivery update invalid payload",
			"thingID", msg.ThingID,
			"action", msg.Key,
			"err", err.Error())
	}
	c.actionValues[msg.Key] = stat
}

// OnEvent handles receiving of an event.
// To be called by the manager of this ConsumerThing, the one that receives
// all subscribed events from the hub client.
// This updates the latest event value and invokes the registered event subscriber, if any.
//
//	tm is the event message received from the hub. This isn't standard WoT so
//	the objective is to remove the need for it.
func (c *ConsumedThing) OnEvent(msg *hubclient.ThingMessage) {
	c.eventValues[msg.Key] = msg
	subscr, found := c.subscribers[msg.Key]
	if found {
		io := c.buildInteractionOutput(msg)
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
func (c *ConsumedThing) OnPropertyUpdate(msg *hubclient.ThingMessage) {
	c.propValues[msg.Key] = msg
	observer, found := c.observers[msg.Key]
	if found {
		io := c.buildInteractionOutput(msg)
		observer(io)
	}
}

// ReadEvent returns the last known Thing event value
// Call ReadAllEvents to refresh the values.
func (c *ConsumedThing) ReadEvent(key string) *InteractionOutput {
	tm := c.eventValues.Get(key)
	if tm == nil {
		return nil
	}
	io := c.buildInteractionOutput(tm)
	return io
}

// ReadProperty returns the last known Thing property value
// Call ReadAllProperties to refresh the property values.
func (c *ConsumedThing) ReadProperty(key string) *InteractionOutput {

	aff := c.td.GetProperty(key)
	if aff == nil {
		return nil
	}
	tm := c.propValues.Get(key)
	o := c.buildInteractionOutput(tm)
	return o
}

// ReadAllEvents reads all Thing event values.
func (c *ConsumedThing) ReadAllEvents() map[string]*InteractionOutput {
	tmsJson, err := digitwin.OutboxReadLatest(
		c.hc, nil, vocab.MessageTypeEvent, "", c.td.ID)
	if err != nil {
		return nil
	}
	c.eventValues, err = hubclient.NewThingMessageMapFromSource(tmsJson)
	if err != nil {
		return nil
	}
	return c.buildInteractionOutputMap(c.eventValues)
}

// ReadAllProperties reads all Thing property values.
func (c *ConsumedThing) ReadAllProperties() map[string]*InteractionOutput {
	tmsJson, err := digitwin.OutboxReadLatest(
		c.hc, nil, vocab.MessageTypeProperty, "", c.td.ID)
	if err != nil {
		return nil
	}
	c.propValues, err = hubclient.NewThingMessageMapFromSource(tmsJson)
	if err != nil {
		return nil
	}
	return c.buildInteractionOutputMap(c.eventValues)
}

// SubscribeEvent sets the handler to invoke when event with the key is received
// This returns an error if an existing subscriber already exists
func (c *ConsumedThing) SubscribeEvent(key string, listener InteractionListener) error {
	if _, found := c.subscribers[key]; found {
		return fmt.Errorf("An event subscriber is already registered")
	}
	c.subscribers[key] = listener
	return nil
}

// WriteProperty requests a change to a property value.
// This takes place asynchronously.
// cb is invoked with the InteractionOutput containing the delivery progress.
//
// Since writing a property can take some time, especially if the device is
// asleep, the callback receives the first response containing a messageID.
// If the request is not yet complete
func (c *ConsumedThing) WriteProperty(key string, value InteractionInput) hubclient.DeliveryStatus {

	stat := c.hc.PubProperty(c.td.ID, key, value)
	// TODO: receive updates
	return stat
}

// WriteMultipleProperties requests a change to multiple property values.
// This takes place asynchronously.
// cb is invoked with the InteractionOutput containing the delivery progress.
func (c *ConsumedThing) WriteMultipleProperties(
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
