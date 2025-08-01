package consumedthing

import (
	"fmt"
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
	// ID of the interaction output in a format that is internally used to identify
	// properties, events and actions: {affordanceType}/{ThingID}/{Name}
	// (Also used by the hiveoview server to notify of updates using SSE)
	ID string

	// ID of the Thing whose output is exposed
	ThingID string
	// Name of the property, event or action affordance name
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
	// Use utils.FormatDateTime(Timestamp,format) to format.
	Timestamp string

	// Type of affordance: "property", "action", "event"
	AffordanceType messaging.AffordanceType

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

func NewInteractionOutputFromValueList(ct *ConsumedThing, affType messaging.AffordanceType, values []digitwin.ThingValue) InteractionOutputMap {
	ioMap := make(map[string]*InteractionOutput)
	for _, tv := range values {
		iout := NewInteractionOutput(ct, affType, tv.Name, tv.Data, tv.Timestamp)
		ioMap[tv.Name] = iout
	}
	return ioMap
}

func NewInteractionOutputFromValue(ct *ConsumedThing, affType messaging.AffordanceType, tv digitwin.ThingValue) *InteractionOutput {
	iout := NewInteractionOutput(ct, affType, tv.Name, tv.Data, tv.Timestamp)
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
	ct *ConsumedThing, affType messaging.AffordanceType, notif *messaging.NotificationMessage) *InteractionOutput {

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
//	ct Consumed Thing instance whose output this belongs to
//	affType is one of AffordanceTypeAction, event or property
//	name is the interaction affordance name the output belongs to
//	raw is the raw data
//	updated is the timestamp the data is last updated
func NewInteractionOutput(ct *ConsumedThing,
	affType messaging.AffordanceType, name string, raw any, updated string) *InteractionOutput {

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
			Title: "NewInteractionOutput: tdi has no " + string(affType) + " output schema: " + name,
		}
		schema.ReadOnly = true // default unless proven otherwise
		raw = ""

	}
	if title == "" {
		title = schema.Title
	}
	io := &InteractionOutput{
		ID:             fmt.Sprintf("%s/%s/%s", affType, ct.ThingID, name),
		AffordanceType: affType,
		ThingID:        ct.ThingID,
		Name:           name,
		Title:          title,
		Timestamp:      updated,
		Schema:         *schema,
		Value:          NewDataSchemaValue(raw),
	}

	return io
}
