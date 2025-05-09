package consumedthing

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/tputils"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/wot/td"
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
	// Use utils.FormatDateTime(Updated,format) to format.
	Updated string

	// Type of affordance: "property", "action", "event"
	AffordanceType string

	// Error value in case reading the value failed
	Err error

	// senderID of last update (action, write property)
	SenderID string
}

//// FormatTime is a helper function to return the formatted time of the given ISO timestamp
//// The default format is RFC822 ("02 Jan 06 15:04 MST")
//func (iout *InteractionOutput) FormatTime(stamp string) (formattedTime string) {
//	formattedTime = utils.FormatAuto(stamp)
//	return formattedTime
//}

// Substr is a simple helper that returns a substring of the given input
func (iout *InteractionOutput) Substr(data any, maxLen int) string {
	text := tputils.DecodeAsString(data, maxLen)
	return text
}

// setSchemaFromTD updates the dataschema fields in this interaction output.
// This first looks for events, then property then action output
//func (iout *InteractionOutput) setSchemaFromTD(td *td.TD) (found bool) {
//	// if name is that of an event then use it
//	eventAff, found := td.Events[iout.Name]
//	if found {
//		//iout.Operation = wot.HTOpPublishEvent
//		iout.AffordanceType = AffordanceTypeEvent
//		// an event might not have data associated with it
//		if eventAff.Data != nil {
//			iout.Schema = *eventAff.Data
//		}
//		iout.Title = eventAff.Title
//		if len(eventAff.Forms) > 0 {
//			//iout.Form = &eventAff.Forms[0]
//		}
//		return true
//	}
//	// if name is that of a property then use it
//	propAff, found := td.Properties[iout.Name]
//	if found {
//		//iout.Operation = wot.HTOpUpdateProperty
//		iout.AffordanceType = AffordanceTypeProperty
//		iout.Schema = propAff.DataSchema
//		iout.Title = propAff.Title
//		if len(propAff.Forms) > 0 {
//			//iout.Form = &propAff.Forms[0]
//		}
//		return true
//	}
//	// last, if name is that of an action with an output schema then use that instead
//	// also update the iout title if it has none.
//	actionAff, found := td.Actions[iout.Name]
//	if found {
//		// an action might not have any output data
//		if actionAff.Output != nil {
//			iout.AffordanceType = AffordanceTypeAction
//			iout.Schema = *actionAff.Output
//		}
//		if iout.Title == "" {
//			iout.Title = actionAff.Title
//		}
//		return true
//	}
//	slog.Warn("setSchemaFromTD: value without schema in the TD",
//		"thingID", iout.ThingID, "name", iout.Name)
//	return false
//}

// NewInteractionOutputFromResponseMap creates a new immutable interaction map from
// a thing value list.
//
// This determines the dataschema by looking for the schema in the events, properties and
// actions (output) section of the TD.
//
//	values is the property or event value map
//	tdi Thing Description instance with schemas for the value. Use nil if schema is unknown.
//func NewInteractionOutputFromValueList(values []digitwin.ThingValue, td *td.TD) InteractionOutputMap {
//	ioMap := make(map[string]*InteractionOutput)
//	for _, tv := range values {
//		io := NewInteractionOutputFromResponse(&tv, td)
//		// property values only contain completed changes.
//		io.Err = nil
//		io.setSchemaFromTD(td)
//		ioMap[tv.Name] = io
//
//	}
//	return ioMap
//}

// UnitSymbol returns the symbol of the unit of this schema using the vocabulary unit map
func (iout *InteractionOutput) UnitSymbol() string {
	if iout.Schema.Unit == "" {
		return ""
	}
	unit, found := vocab.UnitClassesMap[iout.Schema.Unit]
	if !found {
		return iout.Schema.Unit
	}
	return unit.Symbol
}

func NewInteractionOutputFromValueList(ct *ConsumedThing, affType string, values []digitwin.ThingValue) InteractionOutputMap {
	ioMap := make(map[string]*InteractionOutput)
	for _, tv := range values {
		iout := NewInteractionOutput(ct, affType, tv.Name, tv.Output, tv.Updated)
		ioMap[tv.Name] = iout

	}
	return ioMap
}

func NewInteractionOutputFromValue(ct *ConsumedThing, affType string, tv digitwin.ThingValue) *InteractionOutput {
	iout := NewInteractionOutput(ct, affType, tv.Name, tv.Output, tv.Updated)
	return iout
}

// NewInteractionOutputFromNotification creates a new immutable interaction output from
// a NotificationMessage (event,property) and optionally its associated TD.
//
// If no td is available, this value conversion will still be usable but it won't
// contain any schema information.
//
// Intended for use when a property, or event value update is received from
// a subscription.
//
// See also the digitwin Values Service, which provides a ThingValue result that
// includes metadata such as a timestamp when it was last updated and who updated it.
//
//	tm contains the received ThingMessage data
//	tdi is the associated thing description
func NewInteractionOutputFromNotification(
	ct *ConsumedThing, affType string, notif *messaging.NotificationMessage) *InteractionOutput {

	iout := NewInteractionOutput(ct, affType, notif.Name, notif.Data, notif.Timestamp)
	iout.SenderID = notif.SenderID
	return iout
}

// NewInteractionOutputFromActionStatus creates a new immutable interaction output from
// an action status and optionally its associated TD.
//
// If no td is available, this value conversion will still be usable but it won't
// contain any schema information.
//
//	tm contains the received ThingMessage data
//	tdi is the associated thing description
func NewInteractionOutputFromActionStatus(
	ct *ConsumedThing, as messaging.ActionStatus) *InteractionOutput {

	iout := NewInteractionOutput(ct, messaging.AffordanceTypeAction, as.Name, as.Output, as.Updated)
	iout.SenderID = as.SenderID
	return iout
}

// NewInteractionOutput creates a new immutable interaction output from the
// affordance type and raw value.
//
//	tdi TD instance whose output this is
//	affType is one of AffordanceTypeAction, event or property
//	name is the interaction affordance name the output belongs to
//	raw is the raw data
//	updated is the timestamp the data is last updated
func NewInteractionOutput(ct *ConsumedThing, affType string, name string, raw any, updated string) *InteractionOutput {

	var schema *td.DataSchema
	var title string

	switch affType {
	case messaging.AffordanceTypeAction:
		aff := ct.GetActionAff(name)

		if aff == nil {
			// fall back to properties that contain the action output
			break
		}
		title = aff.Title
		schema = aff.Output
	case messaging.AffordanceTypeEvent:
		aff := ct.GetEventAff(name)
		if aff == nil {
			break
		}
		title = aff.Title
		schema = aff.Data
	case messaging.AffordanceTypeProperty:
		aff := ct.GetPropertyAff(name)
		if aff == nil {
			break
		}
		title = aff.Title
		schema = &aff.DataSchema
	}
	if schema == nil {
		schema = &td.DataSchema{
			Title: "NewInteractionOutput: tdi has no " + affType + " output schema: " + name,
		}
		schema.ReadOnly = true // default unless proven otherwise
		raw = ""

	}
	if title == "" {
		title = schema.Title
	}
	io := &InteractionOutput{
		AffordanceType: affType,
		ThingID:        ct.ThingID,
		Name:           name,
		Title:          title,
		Updated:        updated,
		Schema:         *schema,
		Value:          NewDataSchemaValue(raw),
	}

	return io
}
