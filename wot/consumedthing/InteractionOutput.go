package consumedthing

import (
	"github.com/hiveot/hub/api/go/vocab"
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

	// tm contains the message with the value received for this output
	//tm hubclient.ThingMessage
}

// GetUpdated is a helper function to return the formatted time the data was last updated.
// The default format is RFC822 ("02 Jan 06 15:04 MST")
// Optionally "WT" is weekday, time (Mon, 14:31:01 PDT)
// or, provide the time format directly, eg: "02 Jan 06 15:04 MST" for rfc822
func (iout *InteractionOutput) GetUpdated() string {
	return iout.updated
}

// ValueAsArray returns the value as an array
// The result depends on the Schema type
//
//	array: returns array of values as describe ni the Schema
//	boolean: returns a single element true/false
//	bytes: return an array of bytes
//	int: returns a single element with integer
//	object: returns a single element with object
//	string: returns a single element with string
func (iout *InteractionOutput) ValueAsArray() []interface{} {
	objArr := make([]interface{}, 0)
	err := utils.DecodeAsObject(iout.value, &objArr)
	_ = err
	return objArr
}

// ValueAsString returns the value as a string
func (iout *InteractionOutput) ValueAsString() string {
	return utils.DecodeAsString(iout.value)
}

// ValueAsBoolean returns the value as a boolean
func (iout *InteractionOutput) ValueAsBoolean() bool {
	return utils.DecodeAsBool(iout.value)
}

// ValueAsInt returns the value as an integer
func (iout *InteractionOutput) ValueAsInt() int {
	return utils.DecodeAsInt(iout.value)
}

// ValueAsMap returns the value as a key-value map
// Returns nil if no data was provided.
func (iout *InteractionOutput) ValueAsMap() map[string]interface{} {
	o := make(map[string]interface{})
	err := utils.DecodeAsObject(iout.value, &o)
	if err != nil {
		slog.Error("Can't convert value to a map", "value", iout.value)
	}
	return o
}

// NewInteractionOutput creates a new immutable interaction output from object data.
//
//	tm is the received message with the data for this output
//	td Thing Description document with schemas for the value. Use nil if schema is unknown.
func NewInteractionOutput(tm *hubclient.ThingMessage, td *tdd.TD) *InteractionOutput {
	io := &InteractionOutput{
		key:     tm.Key,
		updated: tm.GetUpdated(),
		value:   tm.Data,
	}
	if td == nil {
		return io
	}
	if tm.MessageType == vocab.MessageTypeEvent {
		eventAff, found := td.Events[tm.Key]
		if found {
			io.Schema = eventAff.Data
			if len(eventAff.Forms) > 0 {
				io.Form = &eventAff.Forms[0]
			}
		}
	} else {
		propAff, found := td.Properties[tm.Key]
		if found {
			io.Schema = propAff.DataSchema
			if len(propAff.Forms) > 0 {
				io.Form = &propAff.Forms[0]
			}
		}
	}
	return io
}
