package subprotocols

import (
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/lib/hubclient"
	"io"
	"log/slog"
	"net/http"
)

// RequestParams contains the parameters read from the HTTP request
type RequestParams struct {
	ClientID  string
	ThingID   string
	MessageID string
	Name      string
	Body      []byte
	SessionID string
	ConnID    string
}

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
func GetRequestParams(r *http.Request) (reqParam RequestParams, err error) {

	// get the required client session of this agent
	reqParam.SessionID, reqParam.ClientID, err = GetSessionIdFromContext(r)
	if err != nil {
		// This is an internal error. The middleware session handler would have blocked
		// a request that required a session before getting here.
		slog.Error(err.Error())
		return reqParam, err
	}
	// the connection ID is the sessionID + provided connectionID
	reqParam.ConnID = reqParam.SessionID + "-" + r.Header.Get(hubclient.ConnectionIDHeader)
	// build a message from the URL and payload
	// URLParam names are defined by the path variables set in the router.
	reqParam.ThingID = chi.URLParam(r, "thingID")
	reqParam.Name = chi.URLParam(r, "name")
	reqParam.Body, _ = io.ReadAll(r.Body)

	return reqParam, err
}
