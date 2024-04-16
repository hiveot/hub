// Package httpsbinding with handling of messaging to and from the consumer
package httpsbinding

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"io"
	"log/slog"
	"net/http"
)

// getSessionParams reads the message and identifies the sender's session.
//
// This returns the URL parameters 'thingID' and 'key' from the path along
// with the body payload.
//
// If the session is invalid then write an unauthorized response and return an error
func (svc *HttpsBinding) getRequestParams(w http.ResponseWriter, r *http.Request) (
	session *ClientSession, thingID string, key string, body []byte, err error) {
	// get the required client session of this agent
	ctxSession := r.Context().Value(SessionContextID)
	if ctxSession == nil {
		err = fmt.Errorf("missing session for request '%s' from '%s'",
			r.RequestURI, r.RemoteAddr)
		slog.Warn(err.Error())
		return nil, "", "", nil, err
	}
	cs := ctxSession.(*ClientSession)

	// build a message from the URL and payload
	// URLParam names are defined by the path variables set in the router.
	thingID = chi.URLParam(r, "thingID")
	key = chi.URLParam(r, "key")
	body, _ = io.ReadAll(r.Body)

	return cs, thingID, key, body, err
}

// Forward a message to the runtime router for delivery to the destination thingID.
// The reply is written as-is to the response writer.
// If the handler returns an error, an 'internal server error' is written
func (svc *HttpsBinding) forwardRequest(w http.ResponseWriter, msg *things.ThingMessage) {

	reply, err := svc.handleMessage(msg)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if reply != nil {
		_, _ = w.Write(reply)
	}
	w.WriteHeader(http.StatusOK)
}

// handleConsumerDeleteThing sends a delete thing action request to the directory service
func (svc *HttpsBinding) handleConsumerRemoveThing(w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, _, err := svc.getRequestParams(w, r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// create a ThingMessage containing an action for the directory service.
	// The thingID from the request is used for the message payload.
	args := api.RemoveThingArgs{ThingID: thingID}
	data, _ := json.Marshal(args)
	msg := things.NewThingMessage(
		vocab.MessageTypeAction, api.DigiTwinServiceID, api.RemoveThingMethod,
		data, cs.clientID)
	svc.forwardRequest(w, msg)
}

func (svc *HttpsBinding) handleConsumerReadThing(w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, _, err := svc.getRequestParams(w, r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// populate the ThingMessage with an action for the directory service
	// the thingID from the request is used for the message payload.
	args := api.ReadThingArgs{ThingID: thingID}
	data, _ := json.Marshal(args)
	msg := things.NewThingMessage(
		vocab.MessageTypeAction, api.DigiTwinServiceID, api.ReadThingMethod,
		data, cs.clientID)
	svc.forwardRequest(w, msg)
}

func (svc *HttpsBinding) handleConsumerGetThings(w http.ResponseWriter, r *http.Request) {
	// rpc message with thingID=directory, key=getthings and payload
	// is {query:"expression"} or {things: []thingID} or {agent:id} or {updated:...}...
	// response is {[tdd,...]} with each of the things
}

// handleConsumerGetEvent get historical events of a specific key
func (svc *HttpsBinding) handleConsumerGetEvent(w http.ResponseWriter, r *http.Request) {
	// rpc message with thingID=history, key=getevent and payload
	// is {thingID:thingID,key:eventkey,start:isotime, duration:seconds}
	// reponse is {[event-key: {data:value,timestamp:isotime}, ...]}
}

// handleConsumerGetEvents get latest events of each key
func (svc *HttpsBinding) handleConsumerGetEvents(w http.ResponseWriter, r *http.Request) {
	// rpc message with thingID=history, key=getevents and payload
	// is {thingID:thingID}
	// reponse is {[event-key: {data:value,timestamp:isotime}, ...]}
}
func (svc *HttpsBinding) handleConsumerGetProperties(w http.ResponseWriter, r *http.Request) {
	// rpc message with thingID=valuesvc, key=getproperties and payload
	// is {thingID:thingID}
	// response is {key:value,...}
}

// handleConsumerPostAction handles a consumer's request to post a thing action
func (svc *HttpsBinding) handleConsumerPostAction(w http.ResponseWriter, r *http.Request) {

	cs, thingID, key, data, err := svc.getRequestParams(w, r)
	if err != nil {
		return
	}
	msg := things.NewThingMessage(vocab.MessageTypeAction, thingID, key, data, cs.clientID)
	svc.forwardRequest(w, msg)
}

// handleConsumerPostProperties handles a consumer's request to modify one or more properties
// @param {thingID}   thing to update
func (svc *HttpsBinding) handleConsumerPostProperties(w http.ResponseWriter, r *http.Request) {
	// convert request into a standard message format
	// action message with key $properties
	//svc.onRequest(vocab.MessageTypeProperties, w, r)
}

// SendEvent an event message to subscribers
// This passes it to SSE handlers of active sessions
func (svc *HttpsBinding) SendEvent(message *things.ThingMessage) {
	//sessions := sessionmanager.GetSessions()
	// TODO: track subscriptions
	// TODO: publish to SSE handlers of subscribed clients
}
