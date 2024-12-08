// Package httpserver with handlers for the http protocol
package httpserver

import (
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/servers/httpserver/httpcontext"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"io"
	"log/slog"
	"net/http"
)

// receive a notification message from a client and pass it on to the digital twin.
func (svc *HttpTransportServer) _handleNotification(op string, w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// pass the event to the digitwin service for further processing
	requestID := r.Header.Get(transports.RequestIDHeader)
	if requestID == "" {
		requestID = shortid.MustGenerate()
	}
	msg := transports.NewThingMessage(op, rp.ThingID, rp.Name, rp.Data, rp.ClientID)
	// event style messages does not return data
	svc.messageHandler(msg, nil)
	svc.writeReply(w, nil, nil)
}

// _handleRequestMessage provides the boilerplate code for reading headers,
// unmarshalling the arguments and returning a response. If no immediate
// result is available this returns an alternative RequestStatus result object.
func (svc *HttpTransportServer) _handleRequestMessage(op string, w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if op == "" {
		op = rp.Op
	}

	// an action request should have a cid when used with SSE.
	// without a connection-id this request can not receive an async reply
	if r.Header.Get(transports.ConnectionIDHeader) == "" {
		slog.Info("_handleRequestMessage request has no 'cid' header.",
			"clientID", rp.ClientID, "op", op)
	}

	// pass the event to the digitwin service for further processing
	requestID := r.Header.Get(transports.RequestIDHeader)
	if requestID == "" {
		requestID = shortid.MustGenerate()
	}

	// there are 3 possible results:
	// on status completed; return output
	// on status failed: return http ok with RequestStatus containing error
	// on status other: return  RequestStatus object with progress
	//
	// This means that the result is either out or a RequestStatus object
	// Forms will have added:
	// ```
	//  "additionalResponses": [{
	//                    "success": false,
	//                    "contentType": "application/json",
	//                    "schema": "RequestStatus"
	//                }]
	//```
	msg := transports.NewThingMessage(op, rp.ThingID, rp.Name, rp.Data, rp.ClientID)
	msg.RequestID = requestID
	// reply to the client's return channel
	replyTo := svc.cm.GetConnectionByConnectionID(rp.ConnectionID)
	svc.messageHandler(msg, replyTo)

	//
	//replyHeader := w.Header()
	//if replyHeader == nil {
	//	// this happened a few times during testing. perhaps a broken connection while debugging?
	//	err = fmt.Errorf("HandleActionRequest: Can't return result."+
	//		" Write header is nil. This is unexpected. clientID='%s", rp.ClientID)
	//	svc.writeError(w, err, http.StatusInternalServerError)
	//	return
	//}
	//replyHeader.Set(httpbinding.RequestIDHeader, requestID)
	//
	//// in case of error include the return data schema
	//// TODO: Use schema name from Forms. The action progress schema is in
	//// the forms definition as an additional response.
	//// right now the only response is an action progress.
	//if err != nil {
	//	//replyHeader.Set(httpbinding.StatusHeader, vocab.RequestFailed)
	//	svc.writeReply(w, nil, err)
	//	return
	//	//} else if stat.Status != vocab.RequestCompleted {
	//	//	// if progress isn't completed then also return the delivery progress
	//	//	replyHeader.Set(httpbinding.StatusHeader, stat.Status)
	//	//	replyHeader.Set(httpbinding.DataSchemaHeader, "RequestStatus")
	//	//	svc.writeReply(w, stat, nil)
	//	//	return
	//}
	//// progress is complete, return the default output
	//svc.writeReply(w, output, nil)
}

// HandlePublishActionStatus sends an action progress update message to the digital twin
func (svc *HttpTransportServer) HandlePublishActionStatus(w http.ResponseWriter, r *http.Request) {
	svc._handleNotification(wot.HTOpUpdateActionStatus, w, r)
}

// HandleInvokeAction requests an action from the digital twin.
// NOTE: This returns a header with a dataschema if a schema from
// additionalResponses is returned.
//
// The sender must include the connection-id header of the connection it wants to
// receive the response.
func (svc *HttpTransportServer) HandleInvokeAction(w http.ResponseWriter, r *http.Request) {

	svc._handleRequestMessage(wot.OpInvokeAction, w, r)
}

// HandleLogin handles a login request, posted by a consumer.
//
// This uses the configured session authenticator.
func (svc *HttpTransportServer) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var reply any
	var args map[string]string

	payload, err := io.ReadAll(r.Body)
	if err == nil {
		err = jsoniter.Unmarshal(payload, &args)
	}
	if err == nil {
		// the login is handled in-house and has an immediate return
		// TODO: use-case for 3rd party login? oauth2 process support? tbd
		// FIXME: hard-coded keys!? ugh
		clientID := args["login"]
		password := args["password"]
		reply, err = svc.authenticator.Login(clientID, password)
	}
	if err != nil {
		slog.Warn("HandleLogin failed:", "err", err.Error())
		svc.writeError(w, err, http.StatusUnauthorized)
		return
	}
	// TODO: set client session cookie for browser clients
	//svc.sessionManager.SetSessionCookie(cs.sessionID,token)
	svc.writeReply(w, reply, nil)
}

// HandleLoginRefresh refreshes the auth token using the session authenticator.
// The session authenticator is that of the authn service. This allows testing with a dummy
// authenticator without having to run the authn service.
func (svc *HttpTransportServer) HandleLoginRefresh(w http.ResponseWriter, r *http.Request) {
	var newToken string
	var oldToken string
	rp, err := httpcontext.GetRequestParams(r)
	if err == nil {
		err = tputils.Decode(rp.Data, &oldToken)
	}
	if err == nil {
		newToken, err = svc.authenticator.RefreshToken(rp.ClientID, rp.ClientID, oldToken)
	}
	if err != nil {
		slog.Warn("HandleLoginRefresh failed:", "err", err.Error())
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, newToken, nil)
}

// HandleLogout ends the session and closes all client connections
func (svc *HttpTransportServer) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// use the authenticator
	rp, err := httpcontext.GetRequestParams(r)
	if err == nil {
		svc.authenticator.Logout(rp.ClientID)
	}
	svc.writeReply(w, nil, err)
}

// HandlePublishEvent update digitwin with event published by agent
func (svc *HttpTransportServer) HandlePublishEvent(w http.ResponseWriter, r *http.Request) {
	svc._handleNotification(wot.HTOpPublishEvent, w, r)
}

// HandleQueryAction returns a list of latest action requests of a Thing
// Parameters: thingID
func (svc *HttpTransportServer) HandleQueryAction(w http.ResponseWriter, r *http.Request) {
	svc._handleRequestMessage(wot.OpQueryAction, w, r)
}

// HandleQueryAllActions returns a list of latest action requests of a Thing
// Parameters: thingID
func (svc *HttpTransportServer) HandleQueryAllActions(w http.ResponseWriter, r *http.Request) {
	svc._handleRequestMessage(wot.OpQueryAllActions, w, r)
}

// HandleReadAllEvents returns a list of latest event values from a Thing
// Parameters: thingID
func (svc *HttpTransportServer) HandleReadAllEvents(w http.ResponseWriter, r *http.Request) {
	svc._handleRequestMessage(wot.HTOpReadAllEvents, w, r)
}

// HandleReadAllProperties was added to the top level TD form. Handle it here.
func (svc *HttpTransportServer) HandleReadAllProperties(w http.ResponseWriter, r *http.Request) {
	svc._handleRequestMessage(wot.OpReadAllProperties, w, r)
}

// HandleReadAllThings returns a list of things in the directory
//
// Query params for paging:
//
//	limit=N, limit the number of results to N TD documents
//	offset=N, skip the first N results in the result
//func (svc *HttpBinding) HandleReadAllThings(w http.ResponseWriter, r *http.Request) {
//	rp, err := subprotocols.GetRequestParams(r)
//	if err != nil {
//		w.WriteHeader(http.StatusUnauthorized)
//		return
//	}
//	slog.Debug("HandleReadAllThings", slog.String("SenderID", rp.ClientID))
//	// this request can simply be turned into an action message for the directory.
//	limit := 100
//	offset := 0
//	if r.URL.Query().Has("limit") {
//		limitStr := r.URL.Query().Get("limit")
//		limit32, _ := strconv.ParseInt(limitStr, 10, 32)
//		limit = int(limit32)
//	}
//	if r.URL.Query().Has("offset") {
//		offsetStr := r.URL.Query().Get("offset")
//		offset32, _ := strconv.ParseInt(offsetStr, 10, 32)
//		offset = int(offset32)
//	}
//	thingsList, err := svc.dtwService.DirSvc.ReadAllTDs(rp.ClientID,
//		digitwin.DirectoryReadAllTDsArgs{Offset: offset, Limit: limit})
//	if err != nil {
//		svc.writeError(w, err, 0)
//		return
//	}
//	svc.writeReply(w, thingsList)
//}

// HandleReadEvent returns the latest event value from a Thing
// Parameters: {thingID}, {name}
func (svc *HttpTransportServer) HandleReadEvent(w http.ResponseWriter, r *http.Request) {
	svc._handleRequestMessage(wot.HTOpReadEvent, w, r)
}

func (svc *HttpTransportServer) HandleReadProperty(w http.ResponseWriter, r *http.Request) {
	svc._handleRequestMessage(wot.OpReadProperty, w, r)
}

// HandleReadTD returns the TD of a thing in the directory
// URL parameter {thingID}
func (svc *HttpTransportServer) HandleReadTD(w http.ResponseWriter, r *http.Request) {
	svc._handleRequestMessage(wot.HTOpReadTD, w, r)
}

// HandleReadAllTDs returns the list of digital twin TDs in the directory.
// this is a REST api for convenience. Consider using directory action instead.
func (svc *HttpTransportServer) HandleReadAllTDs(w http.ResponseWriter, r *http.Request) {
	svc._handleRequestMessage(wot.HTOpReadAllTDs, w, r)
}

// HandlePublishMultipleProperties agent sends a map with multiple property
func (svc *HttpTransportServer) HandlePublishMultipleProperties(w http.ResponseWriter, r *http.Request) {
	svc._handleNotification(wot.HTOpUpdateMultipleProperties, w, r)
}

// HandlePublishProperty agent sends single or multiple property updates
func (svc *HttpTransportServer) HandlePublishProperty(w http.ResponseWriter, r *http.Request) {
	svc._handleNotification(wot.HTOpUpdateProperty, w, r)
}

// HandlePublishTD agent sends a new TD document
func (svc *HttpTransportServer) HandlePublishTD(w http.ResponseWriter, r *http.Request) {
	svc._handleNotification(wot.HTOpUpdateTD, w, r)
}

// HandleWriteProperty consumer requests to update a Thing property
func (svc *HttpTransportServer) HandleWriteProperty(w http.ResponseWriter, r *http.Request) {
	svc._handleRequestMessage(wot.OpWriteProperty, w, r)
}
