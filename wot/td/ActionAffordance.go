// Package td with API interface definitions for the ExposedThing and ConsumedThing classes
package td

// ActionAffordance metadata that defines how to invoke a function of a Thing to manipulate
// its state, eg toggle lamp on/off or trigger a process
type ActionAffordance struct {
	//--- InteractionAffordance starts ---

	// AtType is the JSON-LD @type keyword to classify the action using standard vocabulary, or nil if not known
	// Intended for grouping and querying similar actions, and standardized presentation such as icons
	AtType any `json:"@type,omitempty"`

	// Multiline description
	Comments []string `json:"comments,omitempty"`
	// Provides additional (human-readable) information based on a default language
	Description string `json:"description,omitempty"`
	// Provides additional nulti-language information
	Descriptions []string `json:"descriptions,omitempty"`

	// Form hypermedia controls to describe how an operation can be performed
	// Forms are serializations of Protocol Bindings.
	Forms []Form `json:"forms,omitempty"`

	// Provides a human-readable title in the default language
	Title string `json:"title,omitempty"`
	// Provides a multi-language human-readable titles
	Titles []string `json:"titles,omitempty"`
	// Define URI template variables according to [RFC6570] as collection based on DataSchema declarations.
	// HiveOT supports linking uri variable to input
	UriVariables map[string]DataSchema `json:"uriVariables,omitempty"`
	// If this is a dataschema of a URIvariable, then:
	//  Hiveot:input links this as a URI variable schema to the affordance input
	//  If empty or '.', the matching URI variable will be pass as the default input
	//  Otherwise the content is a field name in the input dataschema.
	HiveotInput string `json:"hiveot:input,omitempty"`

	//--- InteractionAffordance ends ---

	//--- Action Affordance fields ---

	// Indicate whether the action is idempotent, eg repeated calls have the
	// same result.
	Idempotent bool `json:"idempotent,omitempty" default:"false"`

	// Define the input data Schema of the action or nil if the action doesn't take input data
	Input *DataSchema `json:"input,omitempty"`

	// Defines the output data Schema of the action or nil if the action doesn't have outputs
	Output *DataSchema `json:"output,omitempty"`

	// Signals if the Action is state safe or not.
	// Safe actions do not change the internal state of a resource.
	// Unsafe action are likely to affect one or more properties or emit events.
	Safe bool `json:"safe,omitempty" default:"false"`

	// A synchronous action means that the response of action contains all the information
	// about the result of the action and no further querying about the status of the action
	// is needed.
	// Lack of this keyword means that no claim on the synchronicity of the action
	// can be made.
	Synchronous bool `json:"synchronous,omitempty"`

	// Allow is a HiveOT extension to list which roles are allowed to invoke this action
	// Intended for present only allowable actions.
	// Without it, the default role permissions apply.
	//Allow []string `json:"allow,omitempty"`
	// Deny is a HiveOT extension to list which roles are denied to invoke this action
	//Deny []string `json:"deny,omitempty"`
}

// AddForm adds an interaction form to the action affordance
// this is not thread-safe.
func (aff *ActionAffordance) AddForm(form Form) {
	aff.Forms = append(aff.Forms, form)
	return
}

// GetAtTypeString returns the @type field from the affordance as a single string.
// If @type is an array then the first item is returned.
func (aff *ActionAffordance) GetAtTypeString() string {
	switch t := aff.AtType.(type) {
	case string:
		return t
	case []string:
		if len(t) > 0 {
			return t[0]
		}
	}
	return ""
}

// Return 'yes'/no if the action is stateful (unsafe)
func (aff *ActionAffordance) GetStateful() string {
	if aff.Safe {
		return "no"
	}
	return "yes"
}

// SetAtType sets the event @type field from the HT vocabulary
func (aff *ActionAffordance) SetAtType(vocabType string) *ActionAffordance {
	aff.AtType = vocabType
	return aff
}
