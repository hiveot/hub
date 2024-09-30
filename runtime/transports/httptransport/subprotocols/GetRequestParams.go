package subprotocols

import (
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/runtime/transports/httptransport/sessions"
	"io"
	"log/slog"
	"net/http"
)

// GetRequestParams reads the client session, URL parameters and body payload from the request.
//
// The session context is set by the http middleware. If the session is not available then
// this returns an error. Note that the session middleware handler will block any request
// that requires a session.
//
// This protocol binding reads two variables, {thingID} and {name} in the path.
//
//	{thingID} is the agent or digital twin thing ID
//	{name} is the property, event or action name. '+' means 'all'
//	{messageType} is a legacy variable that is phased out
func GetRequestParams(r *http.Request) (clientID string, thingID string, name string, body []byte, err error) {

	// get the required client session of this agent
	sessID, clientID, err := sessions.GetSessionIdFromContext(r)
	_ = sessID
	if err != nil {
		// This is an internal error. The middleware session handler would have blocked
		// a request that required a session before getting here.
		slog.Error(err.Error())
		return "", "", "", nil, err
	}

	// build a message from the URL and payload
	// URLParam names are defined by the path variables set in the router.
	thingID = chi.URLParam(r, "thingID")
	name = chi.URLParam(r, "name")
	//messageType = chi.URLParam(r, "messageType")
	body, _ = io.ReadAll(r.Body)

	return clientID, thingID, name, body, err
}
