package httpbasic

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	jsoniter "github.com/json-iterator/go"
)

const SessionContextID = "session"
const ContextClientID = "clientID"

// RequestParams contains the parameters read from the HTTP request
type RequestParams struct {
	ClientID      string // authenticated client ID
	ThingID       string // the thing ID if defined in the URL as {thingID}
	CorrelationID string // tentative as it isn't in the spec
	Name          string // the affordance name if defined in the URL as {name}
	ConnectionID  string // connectionID as provided by the client
	Op            string // the operation if defined in the URL as {op}
	Payload       []byte // the raw request payload (body)
}

// GetRequestParams reads the client session, URL parameters and body payload from the
// http request context.
//
// The session context is set by the http middleware. If the session is not available then
// this returns an error. Note that the session middleware handler will block any request
// that requires a session.
//
// This protocol binding determines three variables, {thingID}, {name} and {op} from the path.
// It unmarshal's the request body into 'data', if given.
//
//	{operation} is the operation
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
	correlationID := r.Header.Get(CorrelationIDHeader)
	reqParam.CorrelationID = correlationID

	// A connection ID distinguishes between different connections from the same client.
	// This is used to correlate http requests with out-of-band responses like a SSE
	// return channel.
	// If a 'cid' header exists, use it as the connection ID.
	headerCID := r.Header.Get(ConnectionIDHeader)
	if headerCID == "" {
		// FIXME: this is only an issue with hiveot-sse. Maybe time to retire it?
		// alt: use a session-id from the auth token - two browser connections would
		// share this however.

		// http-basic isn't be bothered. Each WoT sse connection is the subscription
		//  (only a single subscription per sse connection which is nearly useless)
		slog.Info("GetRequestParams: missing connection-id, only a single " +
			"connection is supported")
	}

	reqParam.ConnectionID = headerCID

	// URLParam names must match the  path variables set in the router.
	reqParam.ThingID = chi.URLParam(r, HttpBasicThingIDURIVar)
	reqParam.Name = chi.URLParam(r, HttpBasicNameURIVar)
	reqParam.Op = chi.URLParam(r, HttpBasicOperationURIVar)
	if r.Body != nil {
		reqParam.Payload, _ = io.ReadAll(r.Body)
	}

	return reqParam, err
}

// GetClientIdFromContext returns the authenticated clientID for the given request
func GetClientIdFromContext(r *http.Request) (clientID string, err error) {
	ctxClientID := r.Context().Value(ContextClientID)
	if ctxClientID == nil {
		return "", errors.New("no clientID in context")
	}
	clientID = ctxClientID.(string)
	return clientID, nil
}

// Convenience function to unmarshal the payload into the given data struct
// If no payload is available this does nothing and data remains unchanged
func (rp *RequestParams) Unmarshal(data any) error {
	if len(rp.Payload) == 0 {
		return nil
	}
	err := jsoniter.Unmarshal(rp.Payload, data)
	return err
}
