// Package thing with API interface definitions for the ExposedThing and ConsumedThing classes
package thing

// ActionAffordance metadata that defines how to invoke a function of a Thing to manipulate
// its state, eg toggle lamp on/off or trigger a process
type ActionAffordance struct {
	//---InteractionAffordance starts

	// ActionType is the JSON-LD @type keyword to classify the action using standard vocabulary, or "" if not known
	// Intended for grouping and querying similar actions, and standardized presentation such as icons
	ActionType string `json:"@type,omitempty"`
	// Provides a human-readable title in the default language
	Title string `json:"title,omitempty"`
	// Provides a multi-language human-readable titles
	Titles []string `json:"titles,omitempty"`
	// Provides additional (human-readable) information based on a default language
	Description string `json:"description,omitempty"`
	// Provides additional nulti-language information
	Descriptions []string `json:"descriptions,omitempty"`

	// Form hypermedia controls to describe how an operation can be performed
	// Forms are serializations of Protocol Bindings.
	Forms []Form `json:"forms"`

	// Define URI template variables according to [RFC6570] as collection based on DataSchema declarations.
	// ... right
	UriVariables map[string]DataSchema `json:"uriVariables,omitempty"`
	//---InteractionAffordance ends

	// Define the input data Schema of the action or nil if the action doesn't take input data
	Input *DataSchema `json:"input,omitempty"`

	// Defines the output data Schema of the action or nil if the action doesn't have outputs
	Output *DataSchema `json:"output,omitempty"`

	// Signals if the Action is state safe (=true) or not
	// Safe actions do not change the internal state of a resource
	Safe bool `json:"safe,omitempty" default:"false"`

	// Indicate whether the action is idempotent, eg repeated calls with the same result
	Idempotent bool `json:"idempotent,omitempty" default:"false"`
}
