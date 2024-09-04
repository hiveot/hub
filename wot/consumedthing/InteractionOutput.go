package consumedthing

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
)

// InteractionOutput to expose the data returned from WoT Interactions to applications.
// Use NewInteractionOutput to initialize
type InteractionOutput struct {
	// The property, event or action key
	key string
	// Title with the human name provided by the interaction affordance
	Title string
	// Schema describing the data from property, event or action affordance
	Schema tdd.DataSchema
	//
	Form *tdd.Form
	// decoded data in its native format as described by the schema
	// eg string, int, array, object
	value interface{}
	// The interaction progress
	Progress hubclient.DeliveryStatus
	// timestamp this was last updated
	updated string
	// senderID of last update
	senderID string

	// tm contains the message with the value received for this output
	//tm hubclient.ThingMessage
}

// GetSenderID is a helper function to ID of the message sender
func (iout *InteractionOutput) GetSenderID() string {
	return iout.senderID
}

// GetUpdated is a helper function to return the formatted time the data was last updated.
// The default format is RFC822 ("02 Jan 06 15:04 MST")
// Optionally "WT" is weekday, time (Mon, 14:31:01 PDT)
// or, provide the time format directly, eg: "02 Jan 06 15:04 MST" for rfc822
func (iout *InteractionOutput) GetUpdated() string {
	return iout.updated
}

// ToArray returns the value as an array
// The result depends on the Schema type
//
//	array: returns array of values as describe ni the Schema
//	boolean: returns a single element true/false
//	bytes: return an array of bytes
//	int: returns a single element with integer
//	object: returns a single element with object
//	string: returns a single element with string
func (iout *InteractionOutput) ToArray() []interface{} {
	objArr := make([]interface{}, 0)
	err := utils.DecodeAsObject(iout.value, &objArr)
	_ = err
	return objArr
}

// ToString returns the value as a string
func (iout *InteractionOutput) ToString() string {
	return utils.DecodeAsString(iout.value)
}

// ToBoolean returns the value as a boolean
func (iout *InteractionOutput) ToBoolean() bool {
	return utils.DecodeAsBool(iout.value)
}

// ToInt returns the value as an integer
func (iout *InteractionOutput) ToInt() int {
	return utils.DecodeAsInt(iout.value)
}

// ToMap returns the value as a key-value map
// Returns nil if no data was provided.
func (iout *InteractionOutput) ToMap() map[string]interface{} {
	o := make(map[string]interface{})
	err := utils.DecodeAsObject(iout.value, &o)
	if err != nil {
		slog.Error("Can't convert value to a map", "value", iout.value)
	}
	return o
}

// NewInteractionOutput creates a new immutable interaction output from object data.
//
// As events are used to update property values, this uses the message Key to
// determine whether this is a property, event or action IO.
//
//	tm is the received message with the data for this output
//	td Thing Description document with schemas for the value. Use nil if schema is unknown.
func NewInteractionOutput(tm *hubclient.ThingMessage, td *tdd.TD) *InteractionOutput {
	io := &InteractionOutput{
		key:      tm.Key,
		senderID: tm.SenderID,
		updated:  tm.GetUpdated("WT"),
		value:    tm.Data,
	}
	if td == nil {
		return io
	}

	actionAff, found := td.Actions[tm.Key]
	if found {
		if actionAff.Output != nil {
			io.Schema = *actionAff.Output
		}
		io.Title = actionAff.Title
		if len(actionAff.Forms) > 0 {
			io.Form = &actionAff.Forms[0]
		}
		return io
	}
	eventAff, found := td.Events[tm.Key]
	if found {
		io.Schema = eventAff.Data
		io.Title = eventAff.Title
		if len(eventAff.Forms) > 0 {
			io.Form = &eventAff.Forms[0]
		}
		return io
	}

	propAff, found := td.Properties[tm.Key]
	if found {
		io.Schema = propAff.DataSchema
		io.Title = propAff.Title
		if len(propAff.Forms) > 0 {
			io.Form = &propAff.Forms[0]
		}
		return io
	}
	slog.Warn("message key not found in TD", "thingID", td.ID, "key", tm.Key, "messageType", tm.MessageType)
	return io
}
