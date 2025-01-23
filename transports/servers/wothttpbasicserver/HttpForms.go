package wothttpbasicserver

import (
	httpserver2 "github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"golang.org/x/exp/slices"
	"log/slog"
	"net/http"
)

type HttpOperation struct {
	ops         []string
	method      string
	subprotocol string
	url         string
	handler     http.HandlerFunc
	//isThingLevel bool
}

// AddTDForms add WoT forms to the given TD containing protocol information to
// access the digital twin things.
// In short:
// 1. TD level forms for
//   - readallproperties
//   - readproperty, writeproperty
//   - observeallproperties
//   - invokeaction
//   - subscribeallevents
//
// 2. Event level form to subscribe to events via SSE
// 3. Action level for to invoke an action over TLS
// AddTDForms add TD top level forms
// This adds operations to read all property values
//
// Note: The WoT TD specification does not support connection sharing for sse based subscriptions.
// hiveot works around this by adding separate hrefs for the sse connection and subscription/observation requests.
//
// ```json
//
//	{
//		  "op": "observeallproperties",
//		  "href": "/observe/thingID",      // this is the subscription request address and not the sse connection address
//		  "htv:methodName": "POST",
//		  "subprotocol": "sse",            // implies href is the shared connection address
//		  "sse:href": "/sse"               // the sse connection address that can be shared with multiple subscriptions
//	}
//
// ```
func (svc *httpserver2.HttpTransportServer) AddTDForms(td *td.TD) error {
	svc.AddDigitwinForms(td)
	svc.AddThingLevelForms(td)
	//svc.AddPropertiesForms(td)
	//svc.AddEventsForms(td)
	//svc.AddActionForms(td)
	return nil
}

// GetForm returns a new HTTP form for the given operation
// Intended for Thing level operations
func (svc *httpserver2.HttpTransportServer) GetForm(op, thingID, name string) td.Form {

	// all operations use URI variables for selecting things
	// HTTP operations
	for _, httpOp := range svc.operations {
		if slices.Contains(httpOp.ops, op) {
			form := td.NewForm(op, httpOp.url)
			form["htv:methodName"] = httpOp.method
			return form
		}
	}

	slog.Warn("GetForm. No form found for operation",
		"op", op)
	return nil
}

// AddActionForms add forms Thing action affordance
// intended for consumers of the digitwin Thing
//func (svc *HttpBinding) AddActionForms(td *tdd.TD) {
//for name, propAff := range td.Actions {
//	params := map[string]string{"thingID": td.ID, "name": name}
//	methodPath := utils.Substitute(httpsse.PostInvokeActionPath, params)
//	propAff.Forms = append(propAff.Forms, tdd.Form{
//		"op":   vocab.WotOpInvokeAction,
//		"href": methodPath,
//		//"contentType":    "application/json",  // default
//		"htv:methodName": "POST",
//	})
//}
//}

// AddEventsForms add forms to subscribe to Thing events
// intended for consumers of the digitwin Thing
//func (svc *HttpBinding) AddEventsForms(td *tdd.TD) {
//	for name, propAff := range td.Events {
//		// the only allowed protocol method is to subscribe to events
//		params := map[string]string{"thingID": td.ID, "name": name}
//		methodPath := utils.Substitute(httpsse.PostSubscribeEventPath, params)
//		propAff.Forms = append(propAff.Forms, tdd.Form{
//			"op":             vocab.WotOpSubscribeEvent,
//			"href":           methodPath,
//			"htv:methodName": "PUT",
//			"subprotocol":    "sse",
//			"contentType":    "text/event-stream",
//		})
//	}
//}

// AddPropertiesForms add forms to read Thing property affordance
// intended for consumers of the digitwin Thing
//func (svc *HttpBinding) AddPropertiesForms(td *tdd.TD) {
//for name, propAff := range td.Properties {
//	propAff.Forms = make([]tdd.Form, 0)
//
//	// the allowed protocol methods are to get and set the property
//	params := map[string]string{"thingID": td.ID, "name": name}
//
//	//propAff.Forms = append(propAff.Forms, propForm)
//	methodPath := utils.Substitute(httpsse.FormPropertyPath, params)
//	if !propAff.WriteOnly {
//		propForm := tdd.Form{
//			"op":   vocab.WoTOpReadProperty,
//			"href": methodPath,
//			// contentType defaults to application/json
//			// htv:methodName defaults to GET
//		}
//		propAff.Forms = append(propAff.Forms, propForm)
//	}
//	if !propAff.ReadOnly {
//		propForm := tdd.Form{
//			"op":   vocab.WoTOpWriteProperty,
//			"href": methodPath,
//			// contentType defaults to application/json
//			// htv:methodName defaults to PUT
//		}
//		propAff.Forms = append(propAff.Forms, propForm)
//	}
//}
//}

// AddDigitwinForms adds Thing level forms for all digitwin read and write operations
func (svc *httpserver2.HttpTransportServer) AddDigitwinForms(tdi *td.TD) {
	methodPath := httpserver2.HttpGetDigitwinPath
	f := td.Form{
		"op": []string{
			wot.OpReadProperty, wot.OpReadAllProperties,
			wot.HTOpReadEvent, wot.HTOpReadAllEvents,
			wot.OpQueryAction, wot.OpQueryAllActions,
			wot.HTOpReadTD, wot.HTOpReadAllTDs,
		},
		"href":           methodPath,
		"contentType":    "application/json",
		"htv:methodName": http.MethodGet,
	}
	tdi.Forms = append(tdi.Forms, f)
	methodPath = httpserver2.HttpGetDigitwinPath
	f = td.Form{
		"op":             []string{wot.OpInvokeAction, wot.OpWriteProperty},
		"href":           methodPath,
		"contentType":    "application/json",
		"htv:methodName": http.MethodPost,
	}
	tdi.Forms = append(tdi.Forms, f)
}

// AddThingLevelForms adds forms with protocol info to the TD, and its properties, events and actions
// HiveOT mostly uses top level forms.
func (svc *httpserver2.HttpTransportServer) AddThingLevelForms(tdi *td.TD) {
	// iterate the thing level operations
	params := map[string]string{"thingID": tdi.ID}
	for _, opInfo := range svc.operations {
		methodPath := tputils.Substitute(opInfo.url, params)
		f := td.Form{
			"op":             opInfo.ops, // not a WoT operation
			"href":           methodPath,
			"contentType":    "application/json",
			"htv:methodName": opInfo.method,
		}
		if opInfo.subprotocol != "" {
			f["subprotocol"] = opInfo.subprotocol
		}
		tdi.Forms = append(tdi.Forms, f)
	}
	// this binding uses the BearerSecurityScheme
	tdi.Security = "bearer"
	tdi.SecurityDefinitions = map[string]td.SecurityScheme{
		"bearer": {Scheme: "bearer", Alg: "ES256", Format: "jwt", In: "header"}}
}
