// Package rest with handling of the digitwin REST API for use by consumers
package rest

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/digitwin/digitwinclient"
	"github.com/hiveot/hub/runtime/protocols/direct"
	"github.com/hiveot/hub/runtime/router"
	"net/http"
)

// DigiTwinValues contains the DigiTwin REST API handlers for reading current values
type DigiTwinValues struct {
	RestHandler
}

// handleRead handles the getActions/Events/Properties request
func (svc *DigiTwinValues) handleRead(method string, w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	since, keys := svc.getSinceKeysParams(r)
	mt := direct.NewDirectTransport(cs.GetClientID(), svc.handleMessage)
	messages, err := digitwinclient.ReadEvents(mt, thingID, keys, since)
	resp, _ := json.Marshal(messages)
	svc.writeReply(w, resp, err)
}

// HandleGetActions handles the agent's request to read outstanding actions
//
//	request body is an optional json encoded ReadActionArgs message
//	This returns a json encoded ReadActionsResp struct data.
func (svc *DigiTwinValues) HandleGetActions(w http.ResponseWriter, r *http.Request) {
	//svc.handleRead(api.ReadActionsMethod, w, r)
	cs, thingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	since, keys := svc.getSinceKeysParams(r)
	mt := direct.NewDirectTransport(cs.GetClientID(), svc.handleMessage)
	messages, err := digitwinclient.ReadActions(mt, thingID, keys, since)
	resp, _ := json.Marshal(messages)
	svc.writeReply(w, resp, err)
}

// HandleGetEvents returns the latest events of each key
func (svc *DigiTwinValues) HandleGetEvents(w http.ResponseWriter, r *http.Request) {
	svc.handleRead(api.ReadEventsMethod, w, r)
}

// HandleGetProperties returns a map of thing property values
func (svc *DigiTwinValues) HandleGetProperties(w http.ResponseWriter, r *http.Request) {
	svc.handleRead(api.ReadPropertiesMethod, w, r)
}

// HandlePostAction handles a consumer's REST call to post a thing action request
// The action can be requested of any Thing. The router will handle it as appropriate.
func (svc *DigiTwinValues) HandlePostAction(w http.ResponseWriter, r *http.Request) {
	cs, thingID, key, rawData, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// simply convert it to an action message
	resp, err := direct.WriteActionMessage(
		thingID, key, rawData, cs.GetClientID(), svc.handleMessage)
	// the reply is a json encoded based on the action definition
	svc.writeReply(w, resp, err)
}

// HandlePostEvent handles an agent's request to send an event
// @param {thingID}   thing to update
// @param {key}       key of event to sent
// @param data        event payload
func (svc *DigiTwinValues) HandlePostEvent(w http.ResponseWriter, r *http.Request) {
	cs, thingID, key, data, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	tm := things.NewThingMessage(
		vocab.MessageTypeEvent, thingID, key, data, cs.GetClientID())
	resp, err := svc.handleMessage(tm)
	svc.writeReply(w, resp, err)
}

// HandlePostProperties handles a consumer's request to modify one or more properties
// @param {thingID}   thing to update
// @param data        map of property key-value paris
func (svc *DigiTwinValues) HandlePostProperties(w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, data, err := svc.getRequestParams(r)
	if err != nil {
		return
	}
	tm := things.NewThingMessage(
		vocab.MessageTypeEvent, thingID, vocab.EventTypeProperties, data, cs.GetClientID())
	resp, err := svc.handleMessage(tm)
	svc.writeReply(w, resp, err)
}

// RegisterMethods registers the digitwin methods available to consumers
// These routes require an authenticated session.
func (svc *DigiTwinValues) RegisterMethods(r chi.Router) {
	// handlers for consumer requests
	r.Get(vocab.GetActionsPath, svc.HandleGetActions)
	r.Get(vocab.GetEventsPath, svc.HandleGetEvents)
	r.Get(vocab.GetPropertiesPath, svc.HandleGetProperties)

	r.Post(vocab.PostActionPath, svc.HandlePostAction)
	r.Post(vocab.PostEventPath, svc.HandlePostEvent)
	r.Post(vocab.PostPropertiesPath, svc.HandlePostProperties)
}

// NewDigiTwinValues creates a new instance of the DigiTwin values REST API
func NewDigiTwinValues(handleMessage router.MessageHandler) *DigiTwinValues {
	svc := &DigiTwinValues{
		RestHandler{handleMessage: handleMessage},
	}
	return svc
}
