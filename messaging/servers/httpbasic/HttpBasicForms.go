package httpbasic

import (
	"fmt"
	"net/http"

	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
)

var validOperations = []string{
	wot.OpQueryAllActions, wot.OpReadAllProperties, wot.OpWriteMultipleProperties,
	wot.OpReadProperty, wot.OpWriteProperty, wot.OpInvokeAction, wot.OpQueryAction}

// AddTDForms adds forms for use of this protocol to the given TD.
// 'includeAffordances' adds forms for all affordances to be compliant with the specifications.
func (srv *HttpBasicServer) AddTDForms(tdoc *td.TD, includeAffordances bool) {
	baseURL := srv.GetConnectURL()

	// add thing level forms that apply to all things. http-basic only supports read/write
	tdoc.Forms = append(tdoc.Forms,
		srv.createThingLevelForm(wot.OpQueryAllActions, http.MethodGet, baseURL, tdoc.ID),
		//srv.createThingLevelForm(wot.OpReadAllEvents, http.MethodGet, baseURL, tdoc.ID),
		srv.createThingLevelForm(wot.OpReadAllProperties, http.MethodGet, baseURL, tdoc.ID),
		srv.createThingLevelForm(wot.OpWriteMultipleProperties, http.MethodPost, baseURL, tdoc.ID),
	)
	if includeAffordances {
		srv.AddAffordanceForms(tdoc)
	}
}

// AddAffordanceForms adds forms to affordances for interacting using the websocket protocol binding
// http-basic only supports read-write
func (srv *HttpBasicServer) AddAffordanceForms(tdoc *td.TD) {
	baseURL := srv.GetConnectURL()
	for name, aff := range tdoc.Actions {
		f := srv.createAffordanceForm(wot.OpInvokeAction, http.MethodPost, baseURL, tdoc.ID, name)
		aff.AddForm(f)
		f = srv.createAffordanceForm(wot.OpQueryAction, http.MethodGet, baseURL, tdoc.ID, name)
		aff.AddForm(f)
	}
	//for name, aff := range tdoc.Events {
	//f = srv.createAffordanceForm(wot.HTOpReadEvent, http.MethodPut, baseURL, tdoc.ID, name)
	//aff.AddForm(f)
	//}
	for name, aff := range tdoc.Properties {
		f := srv.createAffordanceForm(wot.OpReadProperty, http.MethodGet, baseURL, tdoc.ID, name)
		aff.AddForm(f)
		f = srv.createAffordanceForm(wot.OpWriteProperty, http.MethodPut, baseURL, tdoc.ID, name)
		aff.AddForm(f)
	}

}

// createAffordanceForm returns a form for a thing action/event/property affordance operation
// the href in the form has the format "https://host:port/things/{op}/{thingID}/{name}
// Note: in theory these can be replaced with thing level forms using URI variables, except
// for the issue that WoT doesn't support this.
func (srv *HttpBasicServer) createAffordanceForm(op string, httpMethod string, baseURL string,
	thingID string, propName string) td.Form {

	href := fmt.Sprintf("%s/things/%s/%s/%s", baseURL, op, thingID, propName)
	form := td.NewForm(op, href)
	form.SetMethodName(httpMethod)
	form["contentType"] = "application/json"
	return form
}

// createThingLevelForm returns a form for a thing level http operation
// the href in the form has the format "https://host:port/things/{op}/{thingID}
func (srv *HttpBasicServer) createThingLevelForm(op string, httpMethod string, baseURL string, thingID string) td.Form {
	href := fmt.Sprintf("%s/things/%s/%s", baseURL, op, thingID)
	form := td.NewForm(wot.OpQueryAllActions, href)
	form.SetMethodName(httpMethod)
	form["contentType"] = "application/json"
	return form
}
