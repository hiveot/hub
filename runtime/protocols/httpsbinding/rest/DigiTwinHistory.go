// Package rest with handling of the digitwin REST API for use by consumers
package rest

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/runtime/digitwin/digitwinclient"
	"github.com/hiveot/hub/runtime/protocols/direct"
	"github.com/hiveot/hub/runtime/router"
	"net/http"
)

// DigiTwinHistory contains the REST API for reading thing value history
type DigiTwinHistory struct {
	RestHandler
}

// HandleGetActionHistory returns the history of a thing action
func (svc *DigiTwinHistory) HandleGetActionHistory(w http.ResponseWriter, r *http.Request) {
	cs, thingID, key, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	start := ""
	end := ""
	mt := direct.NewDirectTransport(cs.GetClientID(), svc.handleMessage)
	msgList, err := digitwinclient.ReadActionHistory(mt, thingID, key, start, end)
	reply, err := json.Marshal(msgList)
	svc.writeReply(w, reply, err)
}

// HandleGetEventHistory returns the history of a thing event
func (svc *DigiTwinHistory) HandleGetEventHistory(w http.ResponseWriter, r *http.Request) {
	cs, thingID, key, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	start := ""
	end := ""
	mt := direct.NewDirectTransport(cs.GetClientID(), svc.handleMessage)
	msgList, err := digitwinclient.ReadEventHistory(mt, thingID, key, start, end)
	reply, err := json.Marshal(msgList)
	svc.writeReply(w, reply, err)
}

// RegisterMethods registers the history methods available to consumers
// These routes require an authenticated session.
func (svc *DigiTwinHistory) RegisterMethods(r chi.Router) {
	r.Get(vocab.GetEventHistoryPath, svc.HandleGetEventHistory)
	r.Get(vocab.GetActionHistoryPath, svc.HandleGetActionHistory)
}

// NewDigiTwinHistory creates a new instance of the history REST API
func NewDigiTwinHistory(handleMessage router.MessageHandler) *DigiTwinHistory {
	svc := &DigiTwinHistory{
		RestHandler{handleMessage: handleMessage},
	}
	return svc
}
