// Package things with API interface definitions for the ExposedThing and ConsumedThing classes
package td

// EventAffordance with metadata that describes an event source, which asynchronously pushes
// event data to MemberRoles (e.g., overheating alerts).
type EventAffordance struct {
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
	// ... right
	UriVariables map[string]DataSchema `json:"uriVariables,omitempty"`
	//--- InteractionAffordance ends ---

	// Data Schema of the event payload, if any. nil if the event comes without data
	Data *DataSchema `json:"data,omitempty"`

	// subscription is not applicable
	// dataResponse is not applicable
	// cancellation is not applicable
}

// AddForm adds an interaction form to the event affordance
// this is not thread-safe.
func (aff *EventAffordance) AddForm(form Form) {
	aff.Forms = append(aff.Forms, form)
}

// GetAtTypeString returns the @type field from the affordance as a single string.
// If @type is an array then the first item is returned.
func (aff *EventAffordance) GetAtTypeString() string {
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

// SetAtType sets the event @type field from the HT vocabulary
func (aff *EventAffordance) SetAtType(atType string) *EventAffordance {
	aff.AtType = atType
	return aff
}
