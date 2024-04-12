// Package httpsbinding with handling of messaging to and from the agent
package httpsbinding

import (
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	thing "github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/directory"
	"net/http"
)

// handleAgentDeleteThing handles an agent's delete request of a Thing from the digital twin
func (svc *HttpsBinding) handleAgentDeleteThing(w http.ResponseWriter, r *http.Request) {
	msg, session, err := svc.getMessageSession(vocab.MessageTypeAction, r)
	_ = session
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	thingID := msg.ThingID
	msg.ThingID = api.DirectoryServiceID
	msg.Key = directory.DirectoryRemoveThingMethod
	msg.Data = []byte(fmt.Sprintf("{thingID:%s}", thingID))
	_, err = svc.handleMessage(msg)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
}

// handleAgentGetActions handles the agent's request to read outstanding actions
func (svc *HttpsBinding) handleAgentGetActions(w http.ResponseWriter, r *http.Request) {

	msg, session, err := svc.getMessageSession(vocab.MessageTypeAction, r)
	//re-target the message to the directory service
	_ = session
	thingID := msg.ThingID
	msg.ThingID = api.ValueServiceID
	msg.Key = api.ValueServiceMethodGetActions
	msg.Data = []byte(fmt.Sprintf("{thingID:%s,}", thingID))
	reply, err := svc.handleMessage(msg)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.Write(reply)
}

// handleAgentPostEvent handles the agent's request to post a new event
func (svc *HttpsBinding) handleAgentPostEvent(w http.ResponseWriter, r *http.Request) {
	svc.onRequest(vocab.MessageTypeEvent, w, r)
}

// handleAgentPutThing handles an agent's request to update a TD document
// @urlparam {thingID}   thing to update
func (svc *HttpsBinding) handleAgentPutThing(w http.ResponseWriter, r *http.Request) {
	// turn the request in an TD update event
	msg, session, err := svc.getMessageSession(vocab.MessageTypeEvent, r)
	_ = session
	msg.Key = vocab.EventTypeTD
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = svc.handleMessage(msg)
	w.WriteHeader(http.StatusOK)
}

// handleAgentPutProperties handles an agent's request to update property values
// @param {thingID}   thing to update
func (svc *HttpsBinding) handleAgentPutProperties(w http.ResponseWriter, r *http.Request) {
	// turn the request in a properties update event
	msg, session, err := svc.getMessageSession(vocab.MessageTypeEvent, r)
	_ = session
	msg.Key = vocab.EventTypeProperties
	_, err = svc.handleMessage(msg)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// SendActionToAgent sends the action request to the agent and return the result
func (svc *HttpsBinding) SendActionToAgent(agentID string, action *thing.ThingMessage) (resp []byte, err error) {
	// this requires an sse or WS connection from that agent
	return nil, fmt.Errorf("not yet implemented")
}
