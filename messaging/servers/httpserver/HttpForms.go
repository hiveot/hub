package httpserver

import "github.com/hiveot/hub/wot/td"

// AddTDForms adds forms for use of this protocol to the given TD.
// 'includeAffordances' adds forms to all affordances to be compliant with the specifications.
// This adds nosec operations for logging in.
func (svc *HttpTransportServer) AddTDForms(tdoc *td.TD, includeAffordances bool) {
	// special http operations have speci forms
	// TODO: login with nosec security
	// TODO: ...
}
