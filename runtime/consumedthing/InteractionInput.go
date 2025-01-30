package consumedthing

import (
	"github.com/hiveot/hub/wot/td"
	"log/slog"
)

// InteractionInput contains the last value and data schema of an input
// Use NewInteractionInput to initialize
type InteractionInput struct {
	// The property, event or action name
	Name string
	// Title with the human name provided by the interaction affordance
	Title string
	// Schema describing the data from property, event or action affordance
	Schema *td.DataSchema
	// decoded data in its native format as described by the schema
	// eg string, int, array, object
	Value DataSchemaValue
}

// NewInteractionInput creates a new interaction input for property or action
//
//	tdi Thing Description instance with schemas for the value. Use nil if schema is unknown.
//	name of the input property or action
func NewInteractionInput(td *td.TD, name string, defaultValue any) *InteractionInput {
	ii := &InteractionInput{
		Name:  name,
		Value: NewDataSchemaValue(defaultValue),
	}
	if td == nil {
		return ii
	}

	actionAff, found := td.Actions[name]
	if found {
		if actionAff.Input != nil {
			ii.Schema = actionAff.Input
		}
		ii.Title = actionAff.Title
		return ii
	}
	propAff, found := td.Properties[name]
	if found {
		ii.Schema = &propAff.DataSchema
		ii.Title = propAff.Title
		return ii
	}

	slog.Warn("NewInteractionInput: name not a property or action input in TD",
		"thingID", td.ID, "name", name)
	return ii
}
