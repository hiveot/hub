package consumedthing

import (
	"github.com/araddon/dateparse"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"time"
)

type InteractionOutputMap map[string]*InteractionOutput

// InteractionOutput to expose the data returned from WoT Interactions to applications.
// Use NewInteractionOutput to initialize
type InteractionOutput struct {
	// ID of the Thing whose output is exposed
	ThingID string `json:"thing-id,omitempty"`
	// The property, event or action name
	Name string `json:"name,omitempty"`
	// Title with the human name provided by the interaction affordance
	Title string `json:"title,omitempty"`

	// Schema describing the data from property, event or action affordance
	Schema *tdd.DataSchema `json:"schema"`

	// decoded data in its native format as described by the schema
	// eg string, int, array, object
	Value DataSchemaValue `json:"value"`

	// RFC822 timestamp this was last updated.
	// Use GetUpdated(format) to format.
	Updated string `json:"updated,omitempty"`

	//--- non-WoT fields ---
	// Type of output: MessageTypeEvent/Action/Property/TD
	MessageType string `json:"messageType"`

	// ID of the interaction flow of this output
	MessageID string `json:"message-id,omitempty"`

	// The interaction progress
	Progress hubclient.DeliveryStatus `json:"progress"`

	// senderID of last update
	SenderID string
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

// UpdateSchemaFromTD updates the dataschema fields in this interaction output
func (io *InteractionOutput) UpdateSchemaFromTD(td *tdd.TD) (found bool) {
	// if name is that of an event then use it
	eventAff, found := td.Events[io.Name]
	if found {
		io.Schema = eventAff.Data
		io.Title = eventAff.Title
		if len(eventAff.Forms) > 0 {
			//io.Form = &eventAff.Forms[0]
		}
		return true
	}
	// if name is that of a property then use it
	propAff, found := td.Properties[io.Name]
	if found {
		io.Schema = &propAff.DataSchema
		io.Title = propAff.Title
		if len(propAff.Forms) > 0 {
			//io.Form = &propAff.Forms[0]
		}
		return true
	}
	// last, if name is that of an action then use its output schema
	actionAff, found := td.Actions[io.Name]
	if found {
		if actionAff.Output != nil {
			io.Schema = actionAff.Output
		} else {
			io.Schema = nil
		}
		io.Title = actionAff.Title
		if len(actionAff.Forms) > 0 {
			//io.Form = &actionAff.Forms[0]
		}
		return true
	}
	slog.Warn("UpdateSchemaFromTD: Name is not a known event, property or action",
		"name", io.Name, "thingID", io.ThingID)
	return false
}

// NewInteractionOutputFromValueList creates a new immutable interaction map from
// a thing value list.
//
// This determines the dataschema by looking for the schema in the events, properties and
// actions (output) section of the TD.
//
//	values is the property or event value map
//	td Thing Description document with schemas for the value. Use nil if schema is unknown.
func NewInteractionOutputFromValueList(values []digitwin.ThingValue, td *tdd.TD) InteractionOutputMap {
	ioMap := make(map[string]*InteractionOutput)
	for _, tv := range values {
		io := NewInteractionOutputFromValue(&tv, td)
		// property values only contain completed changes.
		io.Progress.Progress = vocab.ProgressStatusCompleted
		io.UpdateSchemaFromTD(td)
		ioMap[tv.Name] = io

	}
	return ioMap
}

// NewInteractionOutputFromValue creates a new immutable interaction output from
// a ThingValue and optionally its associated TD.
//
// If no td is available, this value conversion will still be usable but it won't
// contain any schema information.
//
//	tv contains the thingValue data
//	td is the associated thing description
func NewInteractionOutputFromValue(tv *digitwin.ThingValue, td *tdd.TD) *InteractionOutput {
	io := &InteractionOutput{
		//ThingID:  td.ID,
		MessageID: tv.MessageID,
		Name:      tv.Name,
		SenderID:  tv.SenderID,
		Updated:   tv.Updated,
		Value:     NewDataSchemaValue(tv.Data),
	}
	if td == nil {
		return io
	}
	io.ThingID = td.ID

	// if name is that of an event then use it
	eventAff, found := td.Events[tv.Name]
	if found {
		io.Schema = eventAff.Data
		io.Title = eventAff.Title
		io.MessageType = vocab.MessageTypeEvent
		if len(eventAff.Forms) > 0 {
			//io.Form = &eventAff.Forms[0]
		}
		return io
	}
	// if name is that of a property then use it
	propAff, found := td.Properties[tv.Name]
	if found {
		io.Schema = &propAff.DataSchema
		io.Title = propAff.Title
		io.MessageType = vocab.MessageTypeProperty
		if len(propAff.Forms) > 0 {
			//io.Form = &propAff.Forms[0]
		}
		return io
	}
	// last, if name is that of an action then use its output schema
	actionAff, found := td.Actions[tv.Name]
	if found {
		io.MessageType = vocab.MessageTypeAction
		if actionAff.Output != nil {
			io.Schema = actionAff.Output
		} else if actionAff.Input != nil {
			// Fallback to the input schema if no output is registed
			io.Schema = actionAff.Input
		}
		io.Title = actionAff.Title
		if len(actionAff.Forms) > 0 {
			//io.Form = &actionAff.Forms[0]
		}
		return io
	}

	slog.Warn("value name not found in TD", "thingID", td.ID, "name", tv.Name)
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
		Schema:  schema,
		Value:   NewDataSchemaValue(raw),
	}
	return io
}
