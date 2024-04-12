// Package httpsbinding with handling of messaging to and from the consumer
package httpsbinding

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/directory"
	"io"
	"log/slog"
	"net/http"
)

// getMessageSession reads the message and identifies the sender's session
// This reads the URL parameters 'thingID' and 'key' as used in the paths in ht-vocab.yaml
func (svc *HttpsBinding) getMessageSession(msgType string, r *http.Request) (
	msg *things.ThingMessage, session *ClientSession, err error) {

	// get the required client session of this agent
	ctxSession := r.Context().Value(SessionContextID)
	if ctxSession == nil {
		err = fmt.Errorf("missing session")
		return nil, nil, err
	}
	cs := ctxSession.(*ClientSession)

	// build a message from the URL and payload
	thingID := chi.URLParam(r, "thingID")
	key := chi.URLParam(r, "key")
	data, err := io.ReadAll(r.Body)
	if err != nil {
		err = fmt.Errorf("failed reading message body: %w", err)
	}
	msg = things.NewThingMessage(msgType, thingID, key, data, cs.clientID)
	return msg, cs, err
}

// handleMessage construct a message from the request and passes it to the universal handler.
// This reads the URL parameters 'thingID' and 'key' as used in the paths in ht-vocab.yaml
func (svc *HttpsBinding) onRequest(msgType string, w http.ResponseWriter, r *http.Request) {

	msg, session, err := svc.getMessageSession(msgType, r)
	if err != nil {
		slog.Warn("handlePostAction", "err", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	_ = session
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

func (svc *HttpsBinding) handleConsumerGetThing(w http.ResponseWriter, r *http.Request) {
	msg, session, err := svc.getMessageSession(vocab.MessageTypeAction, r)
	_ = session
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	thingID := msg.ThingID
	msg.ThingID = api.DirectoryServiceID
	msg.Key = directory.DirectoryReadThingMethod
	params := map[string]string{"thingID": thingID}
	msg.Data, _ = json.Marshal(params)
	reply, err := svc.handleMessage(msg)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.Write(reply)
	// response is {tdd}
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
	//msg, session, err := svc.getMessageSession(vocab.MessageTypeAction,r)
	// convert request into a standard message format
	// action message with key the action type
	// response is reply data
	svc.onRequest(vocab.MessageTypeAction, w, r)
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
