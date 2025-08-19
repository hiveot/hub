package httpbasic

import (
	"errors"
	"fmt"
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
	ClientID string
	ThingID  string // the thing ID if defined in the URL as {thingID}
	//CorrelationID string
	Name         string // the affordance name if defined in the URL as {name}
	Data         any
	ConnectionID string // connectionID as provided by the client
	Op           string // the operation if defined in the URL as {op}
}

// GetRequestParams reads the client session, URL parameters and body payload from the
// http request context.
//
// The session context is set by the http middleware. If the session is not available then
// this returns an error. Note that the session middleware handler will block any request
// that requires a session.
//
// This protocol binding determines three variables, {thingID}, {name} and {op} from the path.
// It unmarshals the request body into 'data', if given.
//
//	{operation} is the operation
//	{thingID} is the agent or digital twin thing ID
//	{name} is the property, event or action name. '+' means 'all'
func GetRequestParams(r *http.Request, data any) (reqParam RequestParams, err error) {

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
	// if no connectionID is provided then only a single device connection is allowed.
	headerCID := r.Header.Get(ConnectionIDHeader)
	if headerCID == "" {
		slog.Info("GetRequestParams: missing connection-id, only a single " +
			"connection is supported")
	}

	// the connection ID is the clientID + provided clcid
	reqParam.ConnectionID = headerCID
	//reqParam.CorrelationID = r.Header.Get(CorrelationIDHeader)

	// build a message from the URL and payload
	// URLParam names are defined by the path variables set in the router.
	reqParam.ThingID = chi.URLParam(r, "thingID")
	reqParam.Name = chi.URLParam(r, "name")
	reqParam.Op = chi.URLParam(r, "operation")
	if r.Body != nil {
		payload, _ := io.ReadAll(r.Body)
		if payload != nil && len(payload) > 0 {
			// unmarshal in the data type, if given
			if data != nil {
				err = jsoniter.Unmarshal(payload, data)
				reqParam.Data = data
			} else {
				err = jsoniter.Unmarshal(payload, &reqParam.Data)
			}
		}
	}

	return reqParam, err
}

// GetHiveotParams reads the client session, URL parameters and body payload from the
// http request context for use in the hiveot protocol.
//
// The session context is set by the http middleware. If the session is not available then
// this returns an error. Note that the session middleware handler will block any request
// that requires a session.
func GetHiveotParams(r *http.Request) (clientID, connID string, payload []byte, err error) {

	// get the required client session of this agent
	clientID, err = GetClientIdFromContext(r)
	if err != nil {
		// This is an internal error. The middleware session handler would have blocked
		// a request that required a session before getting here.
		slog.Error(err.Error())
		return
	}
	// the connection ID distinguishes between different connections from the same client.
	// this is needed to correlate http requests with the sub-protocol connection.
	// this is intended to solve for unidirectional SSE connections from multiple devices.
	// if no connectionID is provided then only single device connection is allowed.
	headerCID := r.Header.Get(ConnectionIDHeader)
	if headerCID == "" {
		err = fmt.Errorf("Missing the 'cid' header with the connection ID")
		slog.Error(err.Error())
		return
	}

	// build a message from the URL and payload
	// URLParam names are defined by the path variables set in the router.
	if r.Body != nil {
		payload, _ = io.ReadAll(r.Body)
	}

	return clientID, headerCID, payload, err
}

// GetClientIdFromContext returns the clientID for the given request
func GetClientIdFromContext(r *http.Request) (clientID string, err error) {
	ctxClientID := r.Context().Value(ContextClientID)
	if ctxClientID == nil {
		return "", errors.New("no clientID in context")
	}
	clientID = ctxClientID.(string)
	return clientID, nil
}
