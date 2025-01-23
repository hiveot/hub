package consumedthing

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/services/history/historyclient"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/messaging"
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
	consumer *messaging.Consumer
	// associated thing description instance
	tdi *td.TD
	// observer of property value changes by property name
	observers map[string]InteractionListener
	// subscribers to events by eventName
	subscribers map[string]InteractionListener

	// action status values
	actionStatus map[string]*digitwin.ActionStatus
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
	//just a simple wrapper around the consumer connection
	thingID := ct.tdi.ID
	err := ct.consumer.Rpc(op, thingID, name, input, output)
	return err
}

// build a map of interaction outputs for the given values
func (ct *ConsumedThing) buildInteractionOutputMap(
	tmm map[string]*transports.ResponseMessage) map[string]*InteractionOutput {

	outMap := make(map[string]*InteractionOutput)
	for key, msg := range tmm {
		iout := NewInteractionOutputFromNotification(msg, ct.tdi)
		outMap[key] = iout
	}
	return outMap
}

// Create an interactionOutput for the given thing message
func (ct *ConsumedThing) buildInteractionOutput(msg *transports.ResponseMessage) *InteractionOutput {
	iout := NewInteractionOutputFromNotification(msg, ct.tdi)
	return iout
}

// GetActionInput returns the action input value of the given action, if available
func (ct *ConsumedThing) GetActionInput(as *digitwin.ActionStatus) *InteractionInput {
	if as == nil || as.Name == "" {
		return nil
	}
	iin := NewInteractionInput(ct.tdi, as.Name, as.Input)
	return iin
}

// GetActionOutput returns the interaction output of the latest action value.
// See also GetValue that always return an iout (for rendering purpose)
//
// This returns nil if name is not a known action
func (ct *ConsumedThing) GetActionOutput(as *digitwin.ActionStatus) (iout *InteractionOutput) {
	if as == nil || as.Name == "" {
		iout = NewInteractionOutput(ct.GetTD(), AffordanceTypeAction, "",
			"no output", "")
		iout.Err = errors.New("No record for this action")
	} else {
		iout = NewInteractionOutput(ct.GetTD(), AffordanceTypeAction, as.Name,
			as.Output, as.TimeUpdated)
		if as.Error != "" {
			iout.Err = errors.New(as.Error)
		}
	}
	return iout
}

// GetActionStatus returns the ActionStatus object of the latest action value.
//
// This returns an empty status if name is not a known action
func (ct *ConsumedThing) GetActionStatus(name string) (stat *digitwin.ActionStatus) {
	ct.mux.RLock()
	actionStatus, _ := ct.actionStatus[name]
	ct.mux.RUnlock()
	if actionStatus == nil {
		// return something
		actionStatus = &digitwin.ActionStatus{
			Name:    name,
			ThingID: ct.GetTD().ID,
		}
	}
	return actionStatus
}

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
	ct.mux.RUnlock()
	_ = found
	if iout == nil {
		// not a known prop or event value so create an empty io with a schema from the td
		iout = &InteractionOutput{
			ThingID: ct.tdi.ID,
			Name:    name,
		}
		iout.setSchemaFromTD(ct.tdi)
	}
	return iout
}

// InvokeAction requests an action on the Thing
// TODO: callback for progress updates
func (ct *ConsumedThing) InvokeAction(name string, iin InteractionInput) (*digitwin.ActionStatus, error) {
	var output any

	aff := ct.tdi.GetAction(name)
	if aff == nil {
		err := fmt.Errorf("InvokeAction. Unknown action name: %s", name)
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("InvokeAction", slog.String("name", name))
	err := ct._rpc(wot.OpInvokeAction, name, iin.Value.Raw, &output)
	if err != nil {
		slog.Warn("InvokeAction. failed",
			slog.String("name", name),
			slog.String("error", err.Error()))
		return nil, err
	}

	as := ct.QueryAction(name)
	if as != nil {
		ct.mux.Lock()
		ct.actionStatus[name] = as
		ct.mux.Unlock()
	}
	return as, nil
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

// OnEvent handles receiving a Thing event.
// To be called by the manager of this ConsumerThing, the one that receives
// all subscribed events from the hub client.
// This updates the latest event value and invokes the registered event subscriber, if any.
//
//	msg is the notification message received.
func (ct *ConsumedThing) OnEvent(msg transports.ResponseMessage) {
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
func (ct *ConsumedThing) OnPropertyUpdate(msg transports.ResponseMessage) {
	iout := ct.buildInteractionOutput(&msg)
	ct.mux.Lock()
	ct.propValues[msg.Name] = iout
	observer, found := ct.observers[msg.Name]
	ct.mux.Unlock()
	if found {
		observer(iout)
	}
}

// OnAsyncResponse handles receiving an async progress response.
// This can be the result of an async property write or invoke action.
//
// To be called by the manager of this ConsumerThing, the one that receives
// all async response messages from the server.
//
// Note that rpc calls return the response directly. These are not passed to this handler.
//
//	msg is the response message received.
func (ct *ConsumedThing) OnAsyncResponse(msg *transports.ResponseMessage) {
	if msg.Operation == wot.OpInvokeAction {
		// build an interactionoutput for the action output, if any
		iout := NewInteractionOutput(ct.tdi, AffordanceTypeAction, msg.Name, msg.Output, msg.Updated)

		// notify subscribers - of progress update
		_ = iout
	}
}

// OnTDUpdate handles receiving of an update to the TD document.
// This affects newly created interaction outputs which will be created with
// the updated affordance schema.
//
//	msg is the property message received from the hub. This isn't standard WoT so
//	the objective is to remove the need for it.
func (ct *ConsumedThing) OnTDUpdate(newTD *td.TD) {
	// FIXME: consumed thing interaction output schemas also need updating
	ct.mux.Lock()
	defer ct.mux.Unlock()
	ct.tdi = newTD
}

// QueryAction queries the action status record from the hub
//
// # The cached interaction output of this value can be obtained with GetActionOutput
//
// This returns an empty ActionStatus if not found
func (ct *ConsumedThing) QueryAction(name string) *digitwin.ActionStatus {
	var stat digitwin.ActionStatus
	err := ct._rpc(wot.OpQueryAction, name, nil, &stat)
	if err == nil {
		ct.mux.Lock()
		ct.actionStatus[name] = &stat
		ct.mux.Unlock()
	}
	return &stat
}

// ReadEvent refreshes the last event value by reading it from the hub
func (ct *ConsumedThing) ReadEvent(name string) *InteractionOutput {

	ct.mux.RLock()
	aff, _ := ct.tdi.Events[name]
	ct.mux.RUnlock()
	if aff == nil {
		return nil // not a known event
	}
	tdi := ct.GetTD()
	var raw any
	err := ct._rpc(wot.HTOpReadEvent, name, nil, &raw)
	_ = err
	iout := NewInteractionOutput(tdi, AffordanceTypeEvent, name, raw, "")
	iout.setSchemaFromTD(ct.tdi)
	ct.mux.Lock()
	ct.eventValues[name] = iout
	ct.mux.Unlock()
	return iout
}

// ReadHistory returns the history for the given name
// If the number of values exceed the maximum then this returns itemsRemaining
// as true. An additional call can be made using the last returned timestamp to
// get the remaining values.
func (ct *ConsumedThing) ReadHistory(
	name string, timestamp time.Time, duration time.Duration) (
	values []*transports.ResponseMessage, itemsRemaining bool, err error) {

	// FIXME: ReadHistory is not (yet) part of the WoT specification. Ege mentioned it would
	// be added soon so this will change to follow the WoT specification.
	// Until then this is tied to the Hub's history service.

	hist := historyclient.NewReadHistoryClient(ct.consumer)
	// todo: is there a need to read in batches? not for a single day.
	values, itemsRemaining, err = hist.ReadHistory(
		ct.tdi.ID, name, timestamp, duration, 500)

	return values, itemsRemaining, err
}

// ReadProperty reads the last known Thing property value or nil if name is not a property
// Call GetPropertyValue to get the cached value.
func (ct *ConsumedThing) ReadProperty(name string) *InteractionOutput {
	ct.mux.RLock()
	aff, _ := ct.tdi.Properties[name]
	ct.mux.RUnlock()
	if aff == nil {
		return nil // not a known event
	}
	tdi := ct.GetTD()
	var raw any
	err := ct._rpc(wot.OpReadProperty, name, nil, &raw)
	_ = err
	iout := NewInteractionOutput(tdi, AffordanceTypeProperty, name, raw, "")
	iout.setSchemaFromTD(ct.tdi)
	ct.mux.Lock()
	ct.propValues[name] = iout
	ct.mux.Unlock()
	return iout
}

// ReadAllEvents reads all Thing event values.
func (ct *ConsumedThing) ReadAllEvents() map[string]*InteractionOutput {
	var err error
	var evMsgList []transports.ResponseMessage
	err = ct._rpc(wot.HTOpReadAllEvents, "", nil, &evMsgList)
	if err != nil {
		return nil
	}
	ct.mux.Lock()
	defer ct.mux.Unlock()
	for _, tm := range evMsgList {
		// if the TD doesn't have this event then ignore it
		evAff := ct.tdi.GetEvent(tm.Name)
		if evAff != nil {
			iout := NewInteractionOutput(
				ct.tdi, AffordanceTypeEvent, tm.Name, tm.Output, tm.Updated)
			ct.eventValues[tm.Name] = iout
		}
	}
	return ct.eventValues
}

// ReadAllProperties reads all Thing property values and returns them in a
// map of InteractionOutputs.
func (ct *ConsumedThing) ReadAllProperties() map[string]*InteractionOutput {
	var propMsgList []transports.ResponseMessage

	err := ct._rpc(wot.OpReadAllProperties, "", nil, &propMsgList)
	if err != nil {
		return nil
	}
	ct.mux.Lock()
	defer ct.mux.Unlock()

	for _, msg := range propMsgList {
		// if the TD doesn't have this property then ignore it
		propAff := ct.tdi.GetProperty(msg.Name)
		if propAff != nil {
			iout := NewInteractionOutput(
				ct.tdi, AffordanceTypeProperty, msg.Name, msg.Output, msg.Updated)
			ct.propValues[msg.Name] = iout
		}
	}
	return ct.propValues
}

// ReadAllActions reads all Thing action status values and returns them in a
// map of InteractionOutputs.
func (ct *ConsumedThing) ReadAllActions() map[string]*InteractionOutput {
	var actionMsgList []digitwin.ActionStatus

	err := ct._rpc(wot.OpQueryAllActions, "", nil, &actionMsgList)
	if err != nil {
		return nil
	}
	ct.mux.Lock()
	defer ct.mux.Unlock()

	for _, msg := range actionMsgList {
		// if the TD doesn't have this action then ignore it
		actionAff := ct.tdi.GetAction(msg.Name)
		if actionAff != nil {
			ct.actionStatus[msg.Name] = &msg
		}
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
	thingID := ct.tdi.ID
	raw := ii.Value.Raw
	err = ct.consumer.WriteProperty(thingID, name, raw, true)

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
func NewConsumedThing(td *td.TD, consumer *messaging.Consumer) *ConsumedThing {
	c := ConsumedThing{
		tdi:          td,
		consumer:     consumer,
		observers:    make(map[string]InteractionListener),
		subscribers:  make(map[string]InteractionListener),
		actionStatus: make(map[string]*digitwin.ActionStatus),
		eventValues:  make(map[string]*InteractionOutput),
		propValues:   make(map[string]*InteractionOutput),
	}
	return &c
}
