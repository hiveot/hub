package httpstransport

import (
	"github.com/hiveot/hub/lib/hubclient/httpsse"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
)

// AddTDForms add forms to the given TD containing protocol information to access the digital twin things
// In short:
// 1. TD level forms for readallproperties, readproperty, writeproperty
// 2. Event level form to subscribe to events via SSE
// 3. Action level for to invoke an action over TLS
// AddTDForms add TD top level forms
// This adds operations to read all property values
func (svc *HttpsTransport) AddTDForms(td *things.TD) {
	svc.AddTopLevelForms(td)
	svc.AddPropertiesForms(td)
	svc.AddEventsForms(td)
	svc.AddActionForms(td)
}

// AddActionForms add forms Thing action affordance
// intended for consumers of the digitwin Thing
func (svc *HttpsTransport) AddActionForms(td *things.TD) {
	for key, propAff := range td.Actions {
		// the only allowed protocol method is to set the property
		params := map[string]string{"thingID": td.ID, "key": key}
		methodPath := utils.Substitute(httpsse.PostInvokeActionPath, params)
		propAff.Forms = append(propAff.Forms, things.Form{
			"op":             "write",
			"href":           methodPath,
			"contentType":    "application/json",
			"htv:methodName": "Post",
		})
	}
}

// AddEventsForms add forms to subscribe to Thing events
// intended for consumers of the digitwin Thing
func (svc *HttpsTransport) AddEventsForms(td *things.TD) {
	for key, propAff := range td.Events {
		// the only allowed protocol method is to subscribe to events
		params := map[string]string{"thingID": td.ID, "key": key}
		methodPath := utils.Substitute(httpsse.PostPublishEventPath, params)
		propAff.Forms = append(propAff.Forms, things.Form{
			"op":             "publishevent",
			"href":           methodPath,
			"contentType":    "application/json",
			"htv:methodName": "Post",
		})
	}
}

// AddPropertiesForms add forms to read Thing property affordance
// intended for consumers of the digitwin Thing
func (svc *HttpsTransport) AddPropertiesForms(td *things.TD) {
	for propKey, propAff := range td.Properties {
		// the only allowed protocol method is to set the property
		params := map[string]string{"thingID": td.ID, "key": propKey}
		methodPath := utils.Substitute(httpsse.PostWritePropertyPath, params)
		propForm := things.Form{
			"op":             "writepropery",
			"href":           methodPath,
			"contentType":    "application/json",
			"htv:methodName": "Post",
		}
		propAff.Forms = append(propAff.Forms, propForm)
	}
}

// AddTopLevelForms adds forms with protocol info to the TD, and its properties, events and actions
// HiveOT mostly uses top level forms.
func (svc *HttpsTransport) AddTopLevelForms(td *things.TD) {
	params := map[string]string{"thingID": td.ID}
	methodPath := utils.Substitute(httpsse.GetReadAllEventsPath, params)
	td.Forms = append(td.Forms, things.Form{
		"op":             "readallevents",
		"href":           methodPath,
		"contentType":    "application/json",
		"htv:methodName": "Get",
	})
	methodPath = utils.Substitute(httpsse.GetReadAllPropertiesPath, params)
	td.Forms = append(td.Forms, things.Form{
		"op":             "readallproperties",
		"href":           methodPath,
		"contentType":    "application/json",
		"htv:methodName": "Get",
	})
	methodPath = utils.Substitute(httpsse.PostSubscribeAllEventsPath, params)
	td.Forms = append(td.Forms, things.Form{
		"op":             "subscribeallevents",
		"href":           methodPath,
		"contentType":    "application/json",
		"htv:methodName": "Post",
	})
	methodPath = utils.Substitute(httpsse.PostUnsubscribeAllEventsPath, params)
	td.Forms = append(td.Forms, things.Form{
		"op":             "unsubscribeallevents",
		"href":           methodPath,
		"contentType":    "application/json",
		"htv:methodName": "Post",
	})
}
