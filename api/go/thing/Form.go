// Package thing with API interface definitions for forms
package thing

// Form can be viewed as a statement of "To perform an operation type operation on form context, make a
// request method request to submission target" where the optional form fields may further describe the required
// request. In Thing Descriptions, the form context is the surrounding Object, such as Properties, Actions, and
// Events or the Thing itself for meta-interactions.
// (I this isn't clear then you are not alone)
type Form struct {
	Href        string `json:"href"`
	ContentType string `json:"contentType"`

	// operations types of a form as per https://www.w3.org/TR/wot-thing-description11/#form
	// readproperty, writeproperty, ...
	Op string `json:"op"`
}
