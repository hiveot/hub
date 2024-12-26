// Package things with API interface definitions for forms
package td

import "log/slog"

// Form can be viewed as a statement of "To perform an operation type operation on form context, make a
// request method request to submission target" where the optional form fields may further describe the required
// request. In Thing Descriptions, the form context is the surrounding Object, such as Properties, Actions, and
// Events or the Thing itself for meta-interactions.
type Form map[string]any

// GetHRef returns the form's href field
func (f Form) GetHRef() (href string, found bool) {
	val, found := f["href"]
	if val != nil {
		return val.(string), found
	}
	return "", found
}

// GetOperation returns the form's operation name
func (f Form) GetOperation() string {
	val, _ := f["op"]
	if val == nil {
		slog.Error("Form operation is not set")
		return ""
	}
	return val.(string)
}

// GetMethodName returns the form's HTTP "htv:methodName" field
func (f Form) GetMethodName() (method string, found bool) {
	val, found := f["htv:methodName"]
	if val != nil {
		return val.(string), found
	}
	return "", found
}

// SetMethodName sets the form's HTTP "htv:methodName" field
func (f Form) SetMethodName(method string) {
	f["htv:methodName"] = method
}

func NewForm(operation string, href string) Form {
	return Form{
		"op":   operation,
		"href": href,
	}
}

//Href        string `json:"href"`
//ContentType string `json:"contentType"`
//
//// operations types of a form as per https://www.w3.org/TR/wot-thing-description11/#form
//// readproperty, writeproperty, ...
//Op string `json:"op"`
//}
