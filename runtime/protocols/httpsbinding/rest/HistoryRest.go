// Package rest with handling of the digitwin REST API for use by consumers
package rest

import (
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/runtime/router"
)

// HistoryRest contains the REST API for reading thing value history
type HistoryRest struct {
	RestHandler
}

// HandleGetActionHistory returns the history of a thing action
//func (svc *HistoryRest) HandleGetActionHistory(w http.ResponseWriter, r *http.Request) {
//	cs, thingID, key, _, err := svc.getRequestParams(r)
//	if err != nil {
//		w.WriteHeader(http.StatusUnauthorized)
//		return
//	}
//	start := ""
//	end := ""
//	mt := direct.NewDirectTransport(cs.GetClientID(), svc.handleMessage)
//	msgList, err := history.ReadActionHistory(mt, thingID, key, start, end)
//	reply, err := json.Marshal(msgList)
//	svc.writeReply(w, reply, err)
//}
//

//// HandleGetEventHistory returns the history of a thing event
//func (svc *HistoryRest) HandleGetEventHistory(w http.ResponseWriter, r *http.Request) {
//	cs, thingID, key, _, err := svc.getRequestParams(r)
//	if err != nil {
//		w.WriteHeader(http.StatusUnauthorized)
//		return
//	}
//	start := ""
//	end := ""
//	mt := direct.NewDirectTransport(cs.GetClientID(), svc.handleMessage)
//	msgList, err := history.ReadEventHistory(mt, thingID, key, start, end)
//	reply, err := json.Marshal(msgList)
//	svc.writeReply(w, reply, err)
//}

// RegisterMethods registers the history methods available to consumers
// These routes require an authenticated session.
func (svc *HistoryRest) RegisterMethods(r chi.Router) {
	//r.Get(vocab.GetEventHistoryPath, svc.HandleGetEventHistory)
	//r.Get(vocab.GetActionHistoryPath, svc.HandleGetActionHistory)
}

// NewHistoryRest creates a new instance of the history REST API
func NewHistoryRest(handleMessage router.MessageHandler) *HistoryRest {
	svc := &HistoryRest{
		RestHandler{handleMessage: handleMessage},
	}
	return svc
}
