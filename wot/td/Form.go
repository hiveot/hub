// Package things with API interface definitions for forms
package td

import "log/slog"

// Form can be viewed as a statement of "To perform an operation type operation on form context, make a
// request method request to submission target" where the optional form fields may further describe the required
// request. In Thing Descriptions, the form context is the surrounding Object, such as Properties, Actions, and
// Events or the Thing itself for meta-interactions.
type Form map[string]any

// GetHRef returns the form's href field
// Since hrefs are mandatory, this returns an empty string if not present
func (f Form) GetHRef() (href string) {
	val, found := f["href"]
	if found && val != nil {
		return val.(string)
	}
	return ""
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

// GetSubprotocol returns the form's subprotoco field
func (f Form) GetSubprotocol() (subp string, found bool) {
	val, found := f["subprotocol"]
	if val != nil {
		return val.(string), found
	}
	return "", found
}

// SetMethodName sets the form's HTTP "htv:methodName" field
func (f Form) SetMethodName(method string) Form {
	f["htv:methodName"] = method
	return f
}

// SetSubprotocol sets the form's subprotocol field
func (f Form) SetSubprotocol(subp string) Form {
	f["subprotocol"] = subp
	return f
}

// NewForm creates a new form instance
// Optionally include a sub-protocol as the third parameter
func NewForm(operation string, href string, args ...string) Form {
	f := Form{
		"op":   operation,
		"href": href,
	}
	if len(args) > 0 {
		f["subprotocol"] = args[0]
	}
	return f
}

//Href        string `json:"href"`
//ContentType string `json:"contentType"`
//
//// operations types of a form as per https://www.w3.org/TR/wot-thing-description11/#form
//// readproperty, writeproperty, ...
//Op string `json:"op"`
//}
