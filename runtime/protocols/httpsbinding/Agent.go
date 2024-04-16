// Package httpsbinding with handling of messaging to and from the agent
package httpsbinding

import (
	"encoding/json"
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"net/http"
)

// handleAgentDeleteThing handles an agent's delete request of a Thing from the digital twin
func (svc *HttpsBinding) handleAgentDeleteThing(w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, _, err := svc.getRequestParams(w, r)
	if err != nil {
		return
	}
	args := api.RemoveThingArgs{ThingID: thingID}
	data, _ := json.Marshal(args)
	msg := things.NewThingMessage(vocab.MessageTypeAction, api.DigiTwinServiceID, api.RemoveThingMethod,
		data, cs.clientID)
	svc.forwardRequest(w, msg)
}

// handleAgentGetActions handles the agent's request to read outstanding actions
func (svc *HttpsBinding) handleAgentGetActions(w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, _, err := svc.getRequestParams(w, r)
	if err != nil {
		return
	}
	keys := []string{}
	args := api.ReadActionsArgs{ThingID: thingID, Keys: keys}
	data, _ := json.Marshal(args)
	msg := things.NewThingMessage(vocab.MessageTypeAction, api.DigiTwinServiceID, api.ReadActionsMethod,
		data, cs.clientID)
	svc.forwardRequest(w, msg)
}

// handleAgentPostEvent handles the agent's request to post a new event
func (svc *HttpsBinding) handleAgentPostEvent(w http.ResponseWriter, r *http.Request) {
	cs, thingID, key, data, err := svc.getRequestParams(w, r)
	if err != nil {
		return
	}
	msg := things.NewThingMessage(vocab.MessageTypeEvent, thingID, key, data, cs.clientID)
	svc.forwardRequest(w, msg)
}

// handleAgentPutThing handles an agent's request to update a TD document
// @urlparam {thingID}   thing to update
func (svc *HttpsBinding) handleAgentPutThing(w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, data, err := svc.getRequestParams(w, r)
	if err != nil {
		return
	}
	msg := things.NewThingMessage(
		vocab.MessageTypeEvent, thingID, vocab.EventTypeTD, data, cs.clientID)
	svc.forwardRequest(w, msg)
}

// handleAgentPutProperties handles an agent's request to update property values
// @param {thingID}   thing to update
// data
func (svc *HttpsBinding) handleAgentPutProperties(w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, data, err := svc.getRequestParams(w, r)
	if err != nil {
		return
	}
	msg := things.NewThingMessage(
		vocab.MessageTypeEvent, thingID, vocab.EventTypeProperties, data, cs.clientID)
	svc.forwardRequest(w, msg)
}

// SendActionToAgent sends the action request to the agent and return the result
func (svc *HttpsBinding) SendActionToAgent(agentID string, action *things.ThingMessage) (resp []byte, err error) {
	// this requires an sse or WS connection from that agent
	return nil, fmt.Errorf("not yet implemented")
}
