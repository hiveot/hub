// Package things with API interface definitions for forms
package tdd

// Form can be viewed as a statement of "To perform an operation type operation on form context, make a
// request method request to submission target" where the optional form fields may further describe the required
// request. In Thing Descriptions, the form context is the surrounding Object, such as Properties, Actions, and
// Events or the Thing itself for meta-interactions.
type Form map[string]any

// GetHRef returns the form's href field
func (f Form) GetHRef() (string, bool) {
	val, found := f["href"]
	return val.(string), found
}

// GetOperation returns the form's operation name
func (f Form) GetOperation() string {
	val, _ := f["op"]
	return val.(string)
}

// GetMethodName returns the form's HTTP "htv:methodName" field
func (f Form) GetMethodName() (string, bool) {
	val, found := f["htv:methodName"]
	return val.(string), found
}

//Href        string `json:"href"`
//ContentType string `json:"contentType"`
//
//// operations types of a form as per https://www.w3.org/TR/wot-thing-description11/#form
//// readproperty, writeproperty, ...
//Op string `json:"op"`
//}
