package consumedthing

import (
	"github.com/araddon/dateparse"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
	"time"
)

type InteractionOutputMap map[string]*InteractionOutput

// InteractionOutput to expose the data returned from WoT Interactions to applications.
// Use NewInteractionOutput to initialize
//
// TODO: this seems like a whole lot of processing just to get a value...
// might want to do some performance and resource benchmarking. What is the cost
// and efficiency of using this io in an application vs getting raw data from a server?
type InteractionOutput struct {
	// ID of the Thing whose output is exposed
	ThingID string
	// The property, event or action affordance name
	Name string
	// Title with the human name provided by the interaction affordance
	Title string

	// Schema describing the data from property, event or action affordance
	// This is an empty schema without type, if none is known
	Schema td.DataSchema

	// decoded data in its native format as described by the schema
	// eg string, int, array, object
	Value DataSchemaValue

	// RFC822 timestamp this was last updated.
	// Use GetUpdated(format) to format.
	Updated string

	// Type of affordance: "property", "action", "event"
	AffordanceType string

	// Error value in case reading the value failed
	Err error

	// senderID of last update (action, write property)
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

// SetSchemaFromTD updates the dataschema fields in this interaction output.
// This first looks for events, then property then action output
func (io *InteractionOutput) SetSchemaFromTD(td *td.TD) (found bool) {
	// if name is that of an event then use it
	eventAff, found := td.Events[io.Name]
	if found {
		//io.Operation = wot.HTOpPublishEvent
		io.AffordanceType = "event"
		// an event might not have data associated with it
		if eventAff.Data != nil {
			io.Schema = *eventAff.Data
		}
		io.Title = eventAff.Title
		if len(eventAff.Forms) > 0 {
			//io.Form = &eventAff.Forms[0]
		}
		return true
	}
	// if name is that of a property then use it
	propAff, found := td.Properties[io.Name]
	if found {
		//io.Operation = wot.HTOpUpdateProperty
		io.AffordanceType = "property"
		io.Schema = propAff.DataSchema
		io.Title = propAff.Title
		if len(propAff.Forms) > 0 {
			//io.Form = &propAff.Forms[0]
		}
		return true
	}
	// last, if name is that of an action then use its output schema
	actionAff, found := td.Actions[io.Name]
	if found {
		//io.Operation = wot.OpInvokeAction
		io.AffordanceType = "action"
		// an action might not have any output data
		if actionAff.Output != nil {
			io.Schema = *actionAff.Output
		}
		io.Title = actionAff.Title
		if len(actionAff.Forms) > 0 {
			//io.Form = &actionAff.Forms[0]
		}
		return true
	}
	slog.Warn("SetSchemaFromTD: value without schema in the TD",
		"thingID", io.ThingID, "name", io.Name)
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
func NewInteractionOutputFromValueList(values []transports.ThingMessage, td *td.TD) InteractionOutputMap {
	ioMap := make(map[string]*InteractionOutput)
	for _, tv := range values {
		io := NewInteractionOutputFromValue(&tv, td)
		// property values only contain completed changes.
		io.Err = nil
		io.SetSchemaFromTD(td)
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
func NewInteractionOutputFromValue(tv *transports.ThingMessage, td *td.TD) *InteractionOutput {
	io := &InteractionOutput{
		ThingID:  td.ID,
		Name:     tv.Name,
		SenderID: tv.SenderID,
		Updated:  tv.Timestamp,
		Value:    NewDataSchemaValue(tv.Data),
		Err:      nil,
	}
	if td == nil {
		return io
	}
	io.ThingID = td.ID
	io.SetSchemaFromTD(td)
	return io
}

// NewInteractionOutput creates a new immutable interaction output from
// schema and raw value.
//
// As events are used to update property values, this uses the message Name to
// determine whether this is a property, event or action IO.
//
//	 thingID whose value is contained
//		name is the interaction affordance name the output belongs to
//		schema is the schema info for data, or nil if not known
//		raw is the raw data
//		created is the timestamp the data is created
func NewInteractionOutput(tdi *td.TD, affType string, name string, raw any, created string) *InteractionOutput {
	var schema *td.DataSchema
	switch affType {
	case AffordanceTypeAction:
		aff := tdi.Actions[name]
		if aff == nil {
			break
		}
		schema = aff.Output
	case AffordanceTypeEvent:
		aff := tdi.Events[name]
		if aff == nil {
			break
		}
		schema = aff.Data
	case AffordanceTypeProperty:
		aff := tdi.Properties[name]
		if aff == nil {
			break
		}
		schema = &aff.DataSchema
	}
	if schema == nil {
		schema = &td.DataSchema{
			Title: "unknown schema",
		}
	}
	io := &InteractionOutput{
		ThingID: tdi.ID,
		Name:    name,
		Updated: created,
		Schema:  *schema,
		Value:   NewDataSchemaValue(raw),
	}
	return io
}
