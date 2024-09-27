package httpstransport_old

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
)

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
func (svc *HttpsTransport) AddTDForms(td *tdd.TD) {
	svc.AddThingLevelForms(td)
	//svc.AddPropertiesForms(td)
	//svc.AddEventsForms(td)
	//svc.AddActionForms(td)
}

// AddActionForms add forms Thing action affordance
// intended for consumers of the digitwin Thing
//func (svc *HttpsTransport) AddActionForms(td *tdd.TD) {
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
//func (svc *HttpsTransport) AddEventsForms(td *tdd.TD) {
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
//func (svc *HttpsTransport) AddPropertiesForms(td *tdd.TD) {
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

// AddThingLevelForms adds forms with protocol info to the TD, and its properties, events and actions
// HiveOT mostly uses top level forms.
func (svc *HttpsTransport) AddThingLevelForms(td *tdd.TD) {
	//--- actions
	params := map[string]string{"thingID": td.ID}
	methodPath := utils.Substitute(GetReadAllActionsPath, params)
	td.Forms = append(td.Forms, tdd.Form{
		"op":             "readallactions", // not a WoT operation
		"href":           methodPath,
		"contentType":    "application/json",
		"htv:methodName": "GET",
	})
	// read latest action is an inbox service operation
	methodPath = utils.Substitute(GetReadActionPath, params)
	td.Forms = append(td.Forms, tdd.Form{
		"op":             "readaction", // not a WoT operation
		"href":           methodPath,
		"contentType":    "application/json",
		"htv:methodName": "GET",
	})
	methodPath = utils.Substitute(PostInvokeActionPath, params)
	td.Forms = append(td.Forms, tdd.Form{
		"op":             vocab.WotOpInvokeAction, // not a WoT operation
		"href":           methodPath,
		"contentType":    "application/json",
		"htv:methodName": "POST",
	})

	//--- events
	methodPath = utils.Substitute(GetReadAllEventsPath, params)
	td.Forms = append(td.Forms, tdd.Form{
		"op":             "readallevents", // not a WoT operation
		"href":           methodPath,
		"contentType":    "application/json",
		"htv:methodName": "GET",
	})
	methodPath = utils.Substitute(PostSubscribeAllEventsPath, params)
	td.Forms = append(td.Forms, tdd.Form{
		"op":              vocab.WotOpSubscribeAllEvents,
		"href":            methodPath,
		"htv:methodName":  "POST",
		"contentType":     "application/json",
		"subprotocol":     "sse",
		"sse:href":        "/sse",
		"sse:contentType": "text/event-stream",
	})
	methodPath = utils.Substitute(PostSubscribeEventPath, params)
	td.Forms = append(td.Forms, tdd.Form{
		"op":              vocab.WotOpSubscribeEvent,
		"href":            methodPath,
		"htv:methodName":  "POST",
		"contentType":     "application/json",
		"subprotocol":     "sse",
		"sse:href":        "/sse",
		"sse:contentType": "text/event-stream",
	})

	//--- properties
	methodPath = utils.Substitute(GetReadAllPropertiesPath, params)
	td.Forms = append(td.Forms, tdd.Form{
		"op":             vocab.WotOpReadAllProperties,
		"href":           methodPath,
		"contentType":    "application/json",
		"htv:methodName": "GET",
	})

	methodPath = utils.Substitute(PostUnsubscribeAllEventsPath, params)
	td.Forms = append(td.Forms, tdd.Form{
		"op":             vocab.WotOpUnsubscribeAllEvents,
		"href":           methodPath,
		"contentType":    "application/json",
		"htv:methodName": "POST",
	})
	methodPath = utils.Substitute(PostObserveAllPropertiesPath, params)
	td.Forms = append(td.Forms, tdd.Form{
		"op":              vocab.WotOpObserveAllProperties,
		"href":            methodPath,
		"htv:methodName":  "POST",
		"contentType":     "application/json",
		"subprotocol":     "sse",
		"sse:href":        "/sse",
		"sse:contentType": "text/event-stream",
	})
	methodPath = utils.Substitute(PostObservePropertyPath, params)
	td.Forms = append(td.Forms, tdd.Form{
		"op":              vocab.WoTOpObserveProperty,
		"href":            methodPath,
		"htv:methodName":  "POST",
		"contentType":     "application/json",
		"subprotocol":     "sse",
		"sse:href":        "/sse",
		"sse:contentType": "text/event-stream",
	})
	methodPath = utils.Substitute(GetReadPropertyPath, params)
	td.Forms = append(td.Forms, tdd.Form{
		"op":             vocab.WoTOpReadProperty,
		"href":           methodPath,
		"contentType":    "application/json",
		"htv:methodName": "GET",
	})
	methodPath = utils.Substitute(PostWritePropertyPath, params)
	td.Forms = append(td.Forms, tdd.Form{
		"op":             vocab.WoTOpWriteProperty,
		"href":           methodPath,
		"contentType":    "application/json",
		"htv:methodName": "PUT",
	})

	// this binding uses the BearerSecurityScheme
	td.Security = "bearer"
	td.SecurityDefinitions = map[string]tdd.SecurityScheme{
		"bearer": {Scheme: "bearer", Alg: "ES256", Format: "jwt", In: "header"}}
}
