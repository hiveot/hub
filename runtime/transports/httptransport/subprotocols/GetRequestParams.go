package subprotocols

import (
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/lib/hubclient"
	jsoniter "github.com/json-iterator/go"
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
	Data      any
	CLCID     string
}

// GetRequestParams reads the client session, URL parameters and body payload from the request.
//
// The session context is set by the http middleware. If the session is not available then
// this returns an error. Note that the session middleware handler will block any request
// that requires a session.
//
// This protocol binding determines two variables, {thingID} and {name} from the path.
// It unmarshals the request body if given.
//
//	{thingID} is the agent or digital twin thing ID
//	{name} is the property, event or action name. '+' means 'all'
func GetRequestParams(r *http.Request) (reqParam RequestParams, err error) {

	// get the required client session of this agent
	reqParam.ClientID, err = GetClientIdFromContext(r)
	if err != nil {
		// This is an internal error. The middleware session handler would have blocked
		// a request that required a session before getting here.
		slog.Error(err.Error())
		return reqParam, err
	}
	// the connection ID distinguishes between different connections from the same client.
	// this is needed to correlate http requests with the sub-protocol connection.
	// this is intended to solve for unidirectional SSE connections from multiple devices.
	// if no connectionID is provided then only single device connection is allowed.
	headerCID := r.Header.Get(hubclient.ConnectionIDHeader)

	// the connection ID is the clientID + provided cid
	reqParam.CLCID = reqParam.ClientID + "-" + headerCID
	reqParam.MessageID = r.Header.Get(hubclient.MessageIDHeader)

	// build a message from the URL and payload
	// URLParam names are defined by the path variables set in the router.
	reqParam.ThingID = chi.URLParam(r, "thingID")
	reqParam.Name = chi.URLParam(r, "name")
	if r.Body != nil {
		payload, _ := io.ReadAll(r.Body)
		if payload != nil && len(payload) > 0 {
			err = jsoniter.Unmarshal(payload, &reqParam.Data)
		}
	}

	return reqParam, err
}
