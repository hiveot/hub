package consumedthing

import (
	"github.com/araddon/dateparse"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"time"
)

// InteractionOutput to expose the data returned from WoT Interactions to applications.
// Use NewInteractionOutput to initialize
type InteractionOutput struct {
	// ID of the Thing whose output is exposed
	ThingID string
	// The property, event or action name
	Name string
	// Title with the human name provided by the interaction affordance
	Title string
	// Schema describing the data from property, event or action affordance
	Schema tdd.DataSchema
	//Form *tdd.Form
	// decoded data in its native format as described by the schema
	// eg string, int, array, object
	Value DataSchemaValue
	// The interaction progress
	Progress hubclient.DeliveryStatus
	// RFC822 timestamp this was last updated.
	// Use GetUpdated(format) to format.
	Updated string
	// senderID of last update
	SenderID string

	// tm contains the message with the value received for this output
	tm hubclient.ThingMessage
}

// GetUpdated is a helper function to return the formatted time the data was last updated.
// The default format is RFC822 ("02 Jan 06 15:04 MST")
// Optionally "WT" is weekday, time (Mon, 14:31:01 PDT)
// or, provide the time format directly, eg: "02 Jan 06 15:04 MST" for rfc822
func (iout *InteractionOutput) GetUpdated(format ...string) (updated string) {
	if iout.Updated == "" {
		return ""
	}
	createdTime, _ := dateparse.ParseAny(iout.Updated)
	if format != nil && len(format) == 1 {
		if format[0] == "WT" {
			// Format weekday, time if less than a week old
			age := time.Now().Sub(createdTime)
			if age < time.Hour*24*7 {
				updated = createdTime.Format("Mon, 15:04:05 MST")
			} else {
				updated = createdTime.Format(time.RFC822)
			}
		} else {
			updated = createdTime.Format(format[0])
		}
	} else {
		updated = createdTime.Format(time.RFC822)
	}
	return updated
}

// NewInteractionOutputFromTM creates a new immutable interaction output from
// a thing message.
//
// This determines the dataschema by looking for the schema in the events, properties and
// actions (output) section of the TD.
//
//	tm is the received message with the data for this output
//	td Thing Description document with schemas for the value. Use nil if schema is unknown.
func NewInteractionOutputFromTM(tm *hubclient.ThingMessage, td *tdd.TD) *InteractionOutput {
	io := &InteractionOutput{
		ThingID:  tm.ThingID,
		Name:     tm.Key,
		SenderID: tm.SenderID,
		Updated:  tm.Created,
		Value:    NewDataSchemaValue(tm.Data),
	}
	if td == nil {
		return io
	}
	// if name is that of an event then use it
	eventAff, found := td.Events[tm.Key]
	if found {
		io.Schema = eventAff.Data
		io.Title = eventAff.Title
		if len(eventAff.Forms) > 0 {
			//io.Form = &eventAff.Forms[0]
		}
		return io
	}
	// if name is that of a property then use it
	propAff, found := td.Properties[tm.Key]
	if found {
		io.Schema = propAff.DataSchema
		io.Title = propAff.Title
		if len(propAff.Forms) > 0 {
			//io.Form = &propAff.Forms[0]
		}
		return io
	}
	// last, if name is that of an action then use its output schema
	actionAff, found := td.Actions[tm.Key]
	if found {
		if actionAff.Output != nil {
			io.Schema = *actionAff.Output
		} else if actionAff.Input != nil {
			// Fallback to the input schema if no output is registed
			io.Schema = *actionAff.Input
		}
		io.Title = actionAff.Title
		if len(actionAff.Forms) > 0 {
			//io.Form = &actionAff.Forms[0]
		}
		return io
	}

	slog.Warn("message name not found in TD", "thingID", td.ID, "name", tm.Key, "messageType", tm.MessageType)
	return io
}

// NewInteractionOutput creates a new immutable interaction output from
// schema and raw value.
//
// As events are used to update property values, this uses the message Name to
// determine whether this is a property, event or action IO.
//
//	name is the interaction affordance name the output belongs to
//	schema is the schema info for data
//	raw is the raw data
//	created is the timestamp the data is created
func NewInteractionOutput(thingID string, key string, schema *tdd.DataSchema, raw any, created string) *InteractionOutput {
	if schema == nil {
		schema = &tdd.DataSchema{
			Title: "unknown schema",
		}
	}
	io := &InteractionOutput{
		ThingID: thingID,
		Name:    key,
		Updated: created,
		Value:   NewDataSchemaValue(raw),
	}
	return io
}
