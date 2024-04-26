package rest

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/runtime/protocols/httpsbinding/sessions"
	"github.com/hiveot/hub/runtime/router"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// RestHandler base for rest handlers that contains convenience functions.
type RestHandler struct {
	// handleMessage passes messages in the ThingMessage format to the runtime router
	// for processing, and returns the resulting message or an error.
	handleMessage router.MessageHandler

	// authenticator for handling login
	//sessionAuth api.IAuthenticator
}

// getRequestParams reads the client session, URL parameters and body payload from the request.
//
// The session context is set by the http middleware. If the session is not available then
// this returns an error. Note that the session middleware handler will block any request
// that requires a session.
func (svc *RestHandler) getRequestParams(r *http.Request) (
	session *sessions.ClientSession, thingID string, key string, body []byte, err error) {
	// get the required client session of this agent
	ctxSession := r.Context().Value(sessions.SessionContextID)
	if ctxSession == nil {
		// This is an internal error. The middleware session handler would have blocked
		// a request that required a session before getting here.
		err = fmt.Errorf("Missing session for request '%s' from '%s'",
			r.RequestURI, r.RemoteAddr)
		slog.Error(err.Error())
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

// return optional query parameters
// This parses some standardized parameters including 'since=ISO' and 'keys=a,b,c'
func (svc *RestHandler) getSinceKeysParams(r *http.Request) (since string, keys []string) {
	since = r.URL.Query().Get("since")
	keysParam := r.URL.Query().Get("keys")
	if keysParam != "" {
		keys = strings.Split(keysParam, ",")
	}
	return since, keys
}

// writeReply is a convenience function that writes a reply to a request.
// If the reply has an error then write a bad request with the error as payload
func (svc *RestHandler) writeReply(w http.ResponseWriter, payload []byte, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if payload != nil {
		_, _ = w.Write(payload)
	}
	w.WriteHeader(http.StatusOK)
}

// RegisterOpenMethods registers methods that do not require authentication.
// This includes login
//func (svc *RestHandler) RegisterOpenMethods(r chi.Router) {
//	// TODO: this should be handled by the session authenticator
//	r.Post(vocab.PostLoginPath, svc.handlePostLogin)
//}

// RegisterSecuredMethods registers the digitwin REST API methods for agents and consumers.
// These routes require an authenticated session.
//func (svc *RestHandler) RegisterSecuredMethods(r chi.Router) {
//// handlers for agent messages
//r.Get(vocab.AgentDeleteThingPath, svc.handleAgentDeleteThing)
//r.Get(vocab.AgentGetActionsPath, svc.handleAgentGetActions)
//r.Post(vocab.AgentPostEventPath, svc.handleAgentPostEvent)
//r.Put(vocab.AgentPutPropertiesPath, svc.handleAgentPutProperties)
//r.Put(vocab.AgentPutThingPath, svc.handleAgentPutThing)

//// handlers for consumer requests
//r.Delete(vocab.ConsumerDeleteThingPath, svc.handleConsumerDeleteThing)
//r.Get(vocab.ConsumerGetThingPath, svc.handleConsumerGetThing)
//r.Get(vocab.ConsumerGetThingsPath, svc.handleConsumerGetThings)
//r.Get(vocab.ConsumerGetEventsPath, svc.handleConsumerGetEvents)
//r.Get(vocab.ConsumerGetPropertiesPath, svc.handleConsumerGetProperties)
//r.Post(vocab.ConsumerPostActionPath, svc.handleConsumerPostAction)
//r.Post(vocab.ConsumerPostPropertiesPath, svc.handleConsumerPostProperties)

//// handlers for both agent and consumer messages
//// TODO: this should be handled by the session authenticator
//r.Post(vocab.PostRefreshPath, svc.handlePostRefresh)
//r.Post(vocab.PostLogoutPath, svc.handlePostLogout)
//}

// NewRestHandler creates an instance of a REST API handler for use by agents and
// consumers.
//func NewRestHandler(sessionAuth api.IAuthenticator, handleMessage router.MessageHandler) *RestHandler {
//	handler := RestHandler{
//		sessionAuth:   sessionAuth,
//		handleMessage: handleMessage,
//	}
//	return &handler
//}
