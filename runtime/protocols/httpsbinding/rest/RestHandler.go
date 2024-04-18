package rest

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/runtime/protocols/httpsbinding/sessions"
	"github.com/hiveot/hub/runtime/router"
	"io"
	"log/slog"
	"net/http"
)

// RestHandler contains the digitwin HTTP/REST API for agents and consumers.
// This is intended as a convenience API for REST based clients.
// This handler only handles incoming requests. For a push api see the SSE handler.
//
// The handlers convert the rest request parameters to the messaging format used
// internally by the embedded services. The router handles the messages in
// exactly the same way as those sent through other protocols.
type RestHandler struct {
	// handleMessage passes messages in the ThingMessage format to the runtime router
	// for processing, and returns the resulting message or an error.
	handleMessage router.MessageHandler
}

// getSessionParams reads the client session, URL parameters and body payload from the request.
//
// If the session is invalid then return an error.
func (svc *RestHandler) getRequestParams(r *http.Request) (
	session *sessions.ClientSession, thingID string, key string, body []byte, err error) {
	// get the required client session of this agent
	ctxSession := r.Context().Value(sessions.SessionContextID)
	if ctxSession == nil {
		err = fmt.Errorf("missing session for request '%s' from '%s'",
			r.RequestURI, r.RemoteAddr)
		slog.Warn(err.Error())
		return nil, "", "", nil, err
	}
	cs := ctxSession.(*sessions.ClientSession)

	// build a message from the URL and payload
	// URLParam names are defined by the path variables set in the router.
	thingID = chi.URLParam(r, "thingID")
	key = chi.URLParam(r, "key")
	body, _ = io.ReadAll(r.Body)

	return cs, thingID, key, body, err
}

// RegisterMethods registers the digitwin REST API methods for agents and consumers.
// These routes should be added after verifying authentication.
func (svc *RestHandler) RegisterMethods(r chi.Router) {

	// handlers for agent messages
	r.Get(vocab.AgentDeleteThingPath, svc.handleAgentDeleteThing)
	r.Get(vocab.AgentGetActionsPath, svc.handleAgentGetActions)
	r.Post(vocab.AgentPostEventPath, svc.handleAgentPostEvent)
	r.Put(vocab.AgentPutPropertiesPath, svc.handleAgentPutProperties)
	r.Put(vocab.AgentPutThingPath, svc.handleAgentPutThing)

	// handlers for consumer requests
	r.Post(vocab.ConsumerPostActionPath, svc.handleConsumerPostAction)
	r.Post(vocab.ConsumerPostPropertiesPath, svc.handleConsumerPostProperties)
	r.Delete(vocab.ConsumerDeleteThingPath, svc.handleConsumerRemoveThing)
	r.Get(vocab.ConsumerGetThingPath, svc.handleConsumerReadThing)
	r.Get(vocab.ConsumerGetThingsPath, svc.handleConsumerGetThings)
	r.Get(vocab.ConsumerGetEventPath, svc.handleConsumerGetEvent)
	r.Get(vocab.ConsumerGetEventsPath, svc.handleConsumerGetEvents)
	r.Get(vocab.ConsumerGetPropertiesPath, svc.handleConsumerGetProperties)
}

// NewRestHandler creates an instance of a REST API handler for use by agents and
// consumers.
func NewRestHandler(handleMessage router.MessageHandler) *RestHandler {
	handler := RestHandler{
		handleMessage: handleMessage,
	}
	return &handler
}
