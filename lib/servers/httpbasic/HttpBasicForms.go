package httpbasic

import (
	"fmt"
	"net/http"

	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
)

var HttpKnownOperations = []string{
	wot.OpQueryAllActions, wot.OpReadAllProperties, wot.OpWriteMultipleProperties,
	wot.OpReadProperty, wot.OpWriteProperty, wot.OpInvokeAction, wot.OpQueryAction}

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
		srv.createThingLevelForm(wot.OpQueryAllActions, http.MethodGet, tdoc.ID),
		//srv.createThingLevelForm(wot.OpReadAllEvents, http.MethodGet, tdoc.ID),
		srv.createThingLevelForm(wot.OpReadAllProperties, http.MethodGet, tdoc.ID),
		srv.createThingLevelForm(wot.OpWriteMultipleProperties, http.MethodPost, tdoc.ID),
	)
	if includeAffordances {
		srv.AddAffordanceForms(tdoc)
	}
}

// AddAffordanceForms adds forms to affordances for interacting using the websocket protocol binding
// http-basic only supports read-write
func (srv *HttpWoTBasicServer) AddAffordanceForms(tdoc *td.TD) {
	for name, aff := range tdoc.Actions {
		f := srv.createAffordanceForm(wot.OpInvokeAction, http.MethodPost, tdoc.ID, name)
		aff.AddForm(f)
		f = srv.createAffordanceForm(wot.OpQueryAction, http.MethodGet, tdoc.ID, name)
		aff.AddForm(f)
	}
	//for name, aff := range tdoc.Events {
	//f = srv.createAffordanceForm(wot.HTOpReadEvent, http.MethodPut, baseURL, tdoc.ID, name)
	//aff.AddForm(f)
	//}
	for name, aff := range tdoc.Properties {
		if !aff.WriteOnly {
			// http-basic doesn't support observe/unobserve
			//f := srv.createAffordanceForm(wot.OpObserveProperty, http.MethodGet, tdoc.ID, name)
			//aff.AddForm(f)
			//f = srv.createAffordanceForm(wot.OpUnobserveProperty, http.MethodGet, tdoc.ID, name)
			//aff.AddForm(f)
			f := srv.createAffordanceForm(wot.OpReadProperty, http.MethodGet, tdoc.ID, name)
			aff.AddForm(f)
		}
		if !aff.ReadOnly {
			f := srv.createAffordanceForm(wot.OpWriteProperty, http.MethodPut, tdoc.ID, name)
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
