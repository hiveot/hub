package consumedthing

import (
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
)

// InteractionInput contains the last value and data schema of an input
// Use NewInteractionInput to initialize
type InteractionInput struct {
	// The property, event or action name
	name string
	// Title with the human name provided by the interaction affordance
	Title string
	// Schema describing the data from property, event or action affordance
	Schema *tdd.DataSchema
	// decoded data in its native format as described by the schema
	// eg string, int, array, object
	value DataSchemaValue
}

// Value parses the value and returns it as the given type
func (iin *InteractionInput) Value() DataSchemaValue {
	// generics on methods would be nice
	return iin.value
}

// NewInteractionInput creates a new interaction input for property or action
//
//	td Thing Description document with schemas for the value. Use nil if schema is unknown.
//	name of the input property or action
func NewInteractionInput(td *tdd.TD, key string, defaultValue any) *InteractionInput {
	io := &InteractionInput{
		name:  key,
		value: NewDataSchemaValue(defaultValue),
	}
	if td == nil {
		return io
	}

	actionAff, found := td.Actions[key]
	if found {
		if actionAff.Output != nil {
			io.Schema = actionAff.Output
		}
		io.Title = actionAff.Title
		return io
	}
	eventAff, found := td.Events[key]
	if found {
		io.Schema = eventAff.Data
		io.Title = eventAff.Title
		return io
	}

	propAff, found := td.Properties[key]
	if found {
		io.Schema = &propAff.DataSchema
		io.Title = propAff.Title
		return io
	}
	slog.Warn("message name not found in TD", "thingID", td.ID, "name", key)
	return io
}
