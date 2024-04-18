// Package rest with handling of rest API requests by agents
package rest

import (
	"encoding/json"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"net/http"
)

// handleAgentDeleteThing handles an agent's delete request of a Thing from the digital twin
func (svc *RestHandler) handleAgentDeleteThing(w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		return
	}
	args := api.RemoveThingArgs{ThingID: thingID}
	data, _ := json.Marshal(args)
	msg := things.NewThingMessage(vocab.MessageTypeAction, api.DigiTwinThingID, api.RemoveThingMethod,
		data, cs.GetClientID())
	svc.forwardRequest(w, msg)
}

// handleAgentGetActions handles the agent's request to read outstanding actions
func (svc *RestHandler) handleAgentGetActions(w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		return
	}

	keys := []string{}
	args := api.ReadActionsArgs{ThingID: thingID, Keys: keys}
	data, _ := json.Marshal(args)
	msg := things.NewThingMessage(vocab.MessageTypeAction, api.DigiTwinThingID, api.ReadActionsMethod,
		data, cs.GetClientID())
	svc.forwardRequest(w, msg)
}

// handleAgentPostEvent handles the agent's request to post a new event
func (svc *RestHandler) handleAgentPostEvent(w http.ResponseWriter, r *http.Request) {
	cs, thingID, key, data, err := svc.getRequestParams(r)
	if err != nil {
		return
	}
	msg := things.NewThingMessage(vocab.MessageTypeEvent, thingID, key, data, cs.GetClientID())
	svc.forwardRequest(w, msg)
}

// handleAgentPutThing handles an agent's request to update a TD document
// @urlparam {thingID}   thing to update
func (svc *RestHandler) handleAgentPutThing(w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, data, err := svc.getRequestParams(r)
	if err != nil {
		return
	}
	msg := things.NewThingMessage(
		vocab.MessageTypeEvent, thingID, vocab.EventTypeTD, data, cs.GetClientID())
	svc.forwardRequest(w, msg)
}

// handleAgentPutProperties handles an agent's request to update property values
// @param {thingID}   thing to update
// data
func (svc *RestHandler) handleAgentPutProperties(w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, data, err := svc.getRequestParams(r)
	if err != nil {
		return
	}
	msg := things.NewThingMessage(
		vocab.MessageTypeEvent, thingID, vocab.EventTypeProperties, data, cs.GetClientID())
	svc.forwardRequest(w, msg)
}
