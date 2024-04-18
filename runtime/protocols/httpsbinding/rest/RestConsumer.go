// Package httpsbinding with handling of messaging to and from the consumer
package rest

import (
	"encoding/json"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/digitwin/digitwinclient"
	"github.com/hiveot/hub/runtime/protocols/direct"
	"net/http"
)

// Forward a message to the runtime router for delivery to the destination thingID.
// The reply is written as-is to the response writer.
// If the handler returns an error, an 'internal server error' is written
func (svc *RestHandler) forwardRequest(w http.ResponseWriter, msg *things.ThingMessage) {

	reply, err := svc.handleMessage(msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if reply != nil {
		_, _ = w.Write(reply)
	}
	w.WriteHeader(http.StatusOK)
}

// handleConsumerDeleteThing sends a delete thing action request to the directory service
func (svc *RestHandler) handleConsumerRemoveThing(w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// create a ThingMessage containing an action for the directory service.
	// The thingID from the request is used for the message payload.
	args := api.RemoveThingArgs{ThingID: thingID}
	data, _ := json.Marshal(args)
	msg := things.NewThingMessage(
		vocab.MessageTypeAction, api.DigiTwinThingID, api.RemoveThingMethod,
		data, cs.GetClientID())
	svc.forwardRequest(w, msg)
}

// Return a TD document
func (svc *RestHandler) handleConsumerReadThing(w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	mt := direct.NewDirectTransport(cs.GetClientID(), svc.handleMessage)
	cl := digitwinclient.NewDigiTwinClient(mt)
	td, err := cl.ReadThing(thingID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if td != nil {
		reply, _ := json.Marshal(td)
		_, _ = w.Write(reply)
	}
	w.WriteHeader(http.StatusOK)
	return
}

func (svc *RestHandler) handleConsumerGetThings(w http.ResponseWriter, r *http.Request) {
	// rpc message with thingID=directory, key=getthings and payload
	// is {query:"expression"} or {things: []thingID} or {agent:id} or {updated:...}...
	// response is {[tdd,...]} with each of the things
}

// handleConsumerGetEvent get historical events of a specific key
func (svc *RestHandler) handleConsumerGetEvent(w http.ResponseWriter, r *http.Request) {
	// rpc message with thingID=history, key=getevent and payload
	// is {thingID:thingID,key:eventkey,start:isotime, duration:seconds}
	// reponse is {[event-key: {data:value,timestamp:isotime}, ...]}
}

// handleConsumerGetEvents get latest events of each key
func (svc *RestHandler) handleConsumerGetEvents(w http.ResponseWriter, r *http.Request) {
	// rpc message with thingID=history, key=getevents and payload
	// is {thingID:thingID}
	// reponse is {[event-key: {data:value,timestamp:isotime}, ...]}
}
func (svc *RestHandler) handleConsumerGetProperties(w http.ResponseWriter, r *http.Request) {
	// rpc message with thingID=valuesvc, key=getproperties and payload
	// is {thingID:thingID}
	// response is {key:value,...}
}

// handleConsumerPostAction handles a consumer's request to post a thing action
func (svc *RestHandler) handleConsumerPostAction(w http.ResponseWriter, r *http.Request) {

	cs, thingID, key, data, err := svc.getRequestParams(r)
	if err != nil {
		return
	}
	msg := things.NewThingMessage(
		vocab.MessageTypeAction, thingID, key, data, cs.GetClientID())
	svc.forwardRequest(w, msg)
}

// handleConsumerPostProperties handles a consumer's request to modify one or more properties
// @param {thingID}   thing to update
func (svc *RestHandler) handleConsumerPostProperties(w http.ResponseWriter, r *http.Request) {
	// convert request into a standard message format
	// action message with key $properties
	//svc.onRequest(vocab.MessageTypeProperties, w, r)
}
