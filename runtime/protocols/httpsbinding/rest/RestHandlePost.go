package rest

import (
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/router"
	"net/http"
)

// RestHandlePost contains the REST API for handle posting of event and action messages.
// Requests posted as events and action are simply forwarded to the router as a message.
type RestHandlePost struct {
	RestHandler
}

// HandlePostAction passes a posted action to the router
func (svc *RestHandlePost) HandlePostAction(w http.ResponseWriter, r *http.Request) {
	cs, thingID, key, body, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// this request can simply be turned into an action message.
	msg := things.NewThingMessage(
		vocab.MessageTypeAction, thingID, key, body, cs.GetClientID())

	reply, err := svc.handleMessage(msg)
	svc.writeReply(w, reply, err)
}

// HandlePostEvent passes a posted event to the router
func (svc *RestHandlePost) HandlePostEvent(w http.ResponseWriter, r *http.Request) {
	cs, thingID, key, body, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// this request can simply be turned into an event message.
	msg := things.NewThingMessage(
		vocab.MessageTypeEvent, thingID, key, body, cs.GetClientID())

	_, err = svc.handleMessage(msg)
	svc.writeReply(w, nil, err)
}

// RegisterMethods registers the digitwin methods available to agents
// These routes require an authenticated session.
func (svc *RestHandlePost) RegisterMethods(r chi.Router) {
	r.Post(vocab.PostActionPath, svc.HandlePostAction)
	r.Post(vocab.PostEventPath, svc.HandlePostEvent)
}

// NewTransportRest creates a new instance of the event/action rest handler
func NewTransportRest(handleMessage router.MessageHandler) *RestHandlePost {
	svc := &RestHandlePost{
		RestHandler: RestHandler{handleMessage: handleMessage},
	}
	return svc
}
