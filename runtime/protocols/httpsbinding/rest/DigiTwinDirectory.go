// Package rest with handling of the digitwin REST API for use by agents
package rest

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/digitwin/digitwinclient"
	"github.com/hiveot/hub/runtime/protocols/direct"
	"github.com/hiveot/hub/runtime/router"
	"net/http"
	"strconv"
)

// DigiTwinDirectory contains the digitwin REST API handlers for use by agents
type DigiTwinDirectory struct {
	RestHandler
}

// HandleDeleteThing handles an agent's delete request of a Thing from the digital twin
func (svc *DigiTwinDirectory) HandleDeleteThing(w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	mt := direct.NewDirectTransport(cs.GetClientID(), svc.handleMessage)
	err = digitwinclient.RemoveThing(mt, thingID)
	svc.writeReply(w, nil, err)
}

// HandleGetThing returns a json encoded TD document
func (svc *DigiTwinDirectory) HandleGetThing(w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	mt := direct.NewDirectTransport(cs.GetClientID(), svc.handleMessage)
	tdList, err := digitwinclient.ReadThing(mt, thingID)
	resp, _ := json.Marshal(tdList)
	svc.writeReply(w, resp, err)
}

// HandleGetThings get /things?offset={offset}&limit={limit}
func (svc *DigiTwinDirectory) HandleGetThings(w http.ResponseWriter, r *http.Request) {
	offset := 0
	limit := 1000
	cs, _, _, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	offsetStr := r.URL.Query().Get("offset")
	if offsetStr != "" {
		offset64, _ := strconv.ParseInt(offsetStr, 10, 32)
		offset = int(offset64)
	}
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		limit64, _ := strconv.ParseInt(limitStr, 10, 32)
		limit = int(limit64)
	}
	mt := direct.NewDirectTransport(cs.GetClientID(), svc.handleMessage)
	tdList, err := digitwinclient.ReadThings(mt, offset, limit)
	resp, _ := json.Marshal(tdList)
	svc.writeReply(w, resp, err)

}

// HandlePostThing handles an agent's request to update a TD document.
// This is the same as sending an event of type $td
// @urlparam {thingID}   thing to update
func (svc *DigiTwinDirectory) HandlePostThing(w http.ResponseWriter, r *http.Request) {
	cs, thingID, _, data, err := svc.getRequestParams(r)
	if err != nil {
		return
	}
	// this request can simply be turned into an event message.
	tv := things.NewThingMessage(vocab.MessageTypeEvent, thingID, vocab.EventTypeTD, data, cs.GetClientID())
	_, err = svc.handleMessage(tv)
	svc.writeReply(w, nil, err)
}

// RegisterMethods registers the digitwin methods available to agents
// These routes require an authenticated session.
func (svc *DigiTwinDirectory) RegisterMethods(r chi.Router) {
	r.Delete(vocab.DeleteThingPath, svc.HandleDeleteThing)
	r.Get(vocab.GetThingPath, svc.HandleGetThing)
	r.Get(vocab.GetThingsPath, svc.HandleGetThings)
	r.Post(vocab.PostThingPath, svc.HandlePostThing)
}

// NewDigiTwinDirectory creates a new instance of the DigiTwin directory REST API
func NewDigiTwinDirectory(handleMessage router.MessageHandler) *DigiTwinDirectory {
	svc := &DigiTwinDirectory{
		RestHandler{handleMessage: handleMessage},
	}
	return svc
}
