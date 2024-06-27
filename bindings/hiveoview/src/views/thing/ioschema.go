package thing

import "github.com/hiveot/hub/lib/things"

// IOSchema contains the thing interaction affordance input or output data schema
// with its value.
// Intended for rendering a single field
type IOSchema struct {
	// Thing this schema belongs to
	ThingID string
	// Key the dataschema belongs to
	Key string

	// DataSchema describing the information
	DataSchema *things.DataSchema
	// The corresponding value as string, number, or other type
	Value string
}
