package thing

import "github.com/hiveot/hub/lib/things"

// SchemaValue contains the schema and value of an input or output
// Intended for rendering a single field
type SchemaValue struct {
	ThingID string
	// Interaction affordance key the dataschema belongs to
	Key string

	// DataSchema describing the information
	DataSchema *things.DataSchema
	// The corresponding value as string, number, or other type
	Value string
}
