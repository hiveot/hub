// Package rest with handling of the digitwin REST API for use by consumers
package rest

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/thingValues"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/runtime/protocols/direct"
	"github.com/hiveot/hub/runtime/router"
	"net/http"
)

// ThingValuesRest contains the DigiTwin REST API handlers for reading current values
type ThingValuesRest struct {
	RestHandler
}

// HandleGetActions handles the agent's request to read outstanding actions
//
//	request body is an optional json encoded ReadActionArgs message
//	This returns a json encoded key-value map
func (svc *ThingValuesRest) HandleGetActions(w http.ResponseWriter, r *http.Request) {
	//svc.handleRead(api.ReadActionsMethod, w, r)
	cs, thingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	since, keys := svc.getSinceKeysParams(r)
	mt := direct.NewDirectTransport(cs.GetClientID(), svc.handleMessage)
	messages, err := thingValues.ReadLatest(mt,
		vocab.MessageTypeAction, thingID, keys, since)
	resp, _ := json.Marshal(messages)
	svc.writeReply(w, resp, err)
}

// HandleGetEvents returns the latest events of each key
func (svc *ThingValuesRest) HandleGetEvents(w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	since, keys := svc.getSinceKeysParams(r)
	mt := direct.NewDirectTransport(cs.GetClientID(), svc.handleMessage)
	messages, err := thingValues.ReadLatest(mt, vocab.MessageTypeEvent,
		thingID, keys, since)
	resp, _ := json.Marshal(messages)
	svc.writeReply(w, resp, err)
}

// HandleGetProperties returns a map of thing property values
func (svc *ThingValuesRest) HandleGetProperties(w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// FIXME: this returns events, not properties
	since, keys := svc.getSinceKeysParams(r)
	mt := direct.NewDirectTransport(cs.GetClientID(), svc.handleMessage)
	messages, err := thingValues.ReadLatest(mt, vocab.MessageTypeEvent,
		thingID, keys, since)
	resp, _ := json.Marshal(messages)
	svc.writeReply(w, resp, err)
}

// RegisterMethods registers the digitwin methods available to consumers
// These routes require an authenticated session.
func (svc *ThingValuesRest) RegisterMethods(r chi.Router) {
	// handlers for consumer requests
	r.Get(vocab.GetActionsPath, svc.HandleGetActions)
	r.Get(vocab.GetEventsPath, svc.HandleGetEvents)
	r.Get(vocab.GetPropertiesPath, svc.HandleGetProperties)
}

// NewValuesRest creates a new instance of the DigiTwin values REST API
func NewValuesRest(handleMessage router.MessageHandler) *ThingValuesRest {
	svc := &ThingValuesRest{
		RestHandler{handleMessage: handleMessage},
	}
	return svc
}
