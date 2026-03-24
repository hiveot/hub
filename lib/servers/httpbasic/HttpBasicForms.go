package httpbasic

import (
	"fmt"
	"net/http"

	"github.com/hiveot/hivekit/go/wot/td"
)

var HttpKnownOperations = []string{
	td.OpQueryAllActions, td.OpReadAllProperties, td.OpWriteMultipleProperties,
	td.OpReadProperty, td.OpWriteProperty, td.OpInvokeAction, td.OpQueryAction}

// AddTDForms sets the forms for use of http-basic to the given TD.
//
// This sets the base to the http connect URL and uses relative hrefs for the forms.
// The href MUST match the route defined in: HttpBasicRoutes.HttpBasicAffordanceOperationPath
// e.g.: "https://host:port/things/{op}/{thingID}/{name}"
//
// As content-Type is the default application/json it is omitted.
//
// If the method names for get operations is default GET then it is omitted from the form.
//
// includeAffordances adds forms for all affordances to be compliant with the specifications.
func (srv *HttpWoTBasicServer) AddTDForms(tdoc *td.TD, includeAffordances bool) {
	// defaults to https://host:port/
	tdoc.Base = fmt.Sprintf("%s", srv.GetConnectURL())

	// add thing level forms that apply to all things. http-basic only supports read/write
	tdoc.Forms = append(tdoc.Forms,
		srv.createThingLevelForm(td.OpQueryAllActions, http.MethodGet, tdoc.ID),
		//srv.createThingLevelForm(td.OpReadAllEvents, http.MethodGet, tdoc.ID),
		srv.createThingLevelForm(td.OpReadAllProperties, http.MethodGet, tdoc.ID),
		srv.createThingLevelForm(td.OpWriteMultipleProperties, http.MethodPost, tdoc.ID),
	)
	if includeAffordances {
		srv.AddAffordanceForms(tdoc)
	}
}

// AddAffordanceForms adds forms to affordances for interacting using the websocket protocol binding
// http-basic only supports read-write
func (srv *HttpWoTBasicServer) AddAffordanceForms(tdoc *td.TD) {
	for name, aff := range tdoc.Actions {
		f := srv.createAffordanceForm(td.OpInvokeAction, http.MethodPost, tdoc.ID, name)
		aff.AddForm(f)
		f = srv.createAffordanceForm(td.OpQueryAction, http.MethodGet, tdoc.ID, name)
		aff.AddForm(f)
	}
	//for name, aff := range tdoc.Events {
	//f = srv.createAffordanceForm(td.HTOpReadEvent, http.MethodPut, baseURL, tdoc.ID, name)
	//aff.AddForm(f)
	//}
	for name, aff := range tdoc.Properties {
		if !aff.WriteOnly {
			// http-basic doesn't support observe/unobserve
			//f := srv.createAffordanceForm(td.OpObserveProperty, http.MethodGet, tdoc.ID, name)
			//aff.AddForm(f)
			//f = srv.createAffordanceForm(td.OpUnobserveProperty, http.MethodGet, tdoc.ID, name)
			//aff.AddForm(f)
			f := srv.createAffordanceForm(td.OpReadProperty, http.MethodGet, tdoc.ID, name)
			aff.AddForm(f)
		}
		if !aff.ReadOnly {
			f := srv.createAffordanceForm(td.OpWriteProperty, http.MethodPut, tdoc.ID, name)
			aff.AddForm(f)
		}
	}
}

// createAffordanceForm returns a form for a thing action/event/property affordance operation
// the href in the form has the format "{base}/{op}/{thingID}/{name}
// where {base} is https://
// Note: in theory these can be replaced with thing level forms using URI variables, except
// for the issue that WoT doesn't support this.
//
// The baseURL is the URL
func (srv *HttpWoTBasicServer) createAffordanceForm(op string, httpMethod string,
	thingID string, name string) td.Form {

	href := fmt.Sprintf("%s/%s/%s/%s", HttpBaseFormOp, op, thingID, name)
	form := td.NewForm(op, href)
	if httpMethod != "" && httpMethod != http.MethodGet {
		form.SetMethodName(httpMethod)
	}
	// contentType has a default of application/json
	//form["contentType"] = "application/json"
	return form
}

// createThingLevelForm returns a form for a thing level http operation
// the href in the form has the format "https://host:port/things/{op}/{thingID}
func (srv *HttpWoTBasicServer) createThingLevelForm(op string, httpMethod string, thingID string) td.Form {
	// href is relative to base
	href := fmt.Sprintf("%s/%s/%s", HttpBaseFormOp, op, thingID)
	form := td.NewForm(op, href)
	form.SetMethodName(httpMethod)
	//form["contentType"] = "application/json"
	return form
}
