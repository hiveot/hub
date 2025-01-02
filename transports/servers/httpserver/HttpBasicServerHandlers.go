// Package httpserver with handlers for the http protocol
package httpserver

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"io"
	"log/slog"
	"net/http"
)

// HandleActionStatus handles a received action status message from an agent client
// and forwards this as a ResponseMessage to the server response handler.
//func (svc *HttpTransportServer) HandleActionStatus(w http.ResponseWriter, r *http.Request) {
//	//svc.HandleNotification(wot.HTOpActionStatus, w, r)
//	rp, err := GetRequestParams(r)
//	if err != nil {
//		slog.Error(err.Error())
//		w.WriteHeader(http.StatusUnauthorized)
//		return
//	}
//	actionStatus := HttpActionStatus{}
//	err = tputils.Decode(rp.Data, &actionStatus)
//	if err != nil {
//		slog.Warn("HandleActionStatus. Payload is not an HttpActionStatus object",
//			"agentID", rp.ClientID,
//			"correlationID", rp.CorrelationID)
//	}
//	// on the server the action status is handled using a standardized ResponseMessage instance
//	// This is converted to the transport protocol used to send it to the client.
//	response := transports.ResponseMessage{
//		MessageType: transports.MessageTypeResponse,
//		Operation:   wot.OpInvokeAction,
//		ThingID:     rp.ThingID,
//		Name:        rp.Name,
//		CorrelationID:   rp.CorrelationID,
//		SenderID:    rp.ClientID,
//		Status:      actionStatus.Status, // todo map names (they are the same)
//		Error:       actionStatus.Error,
//		Output:      actionStatus.Output,
//		Received:    actionStatus.TimeRequested,
//		Updated:     actionStatus.TimeEnded,
//	}
//	if svc.serverResponseHandler == nil {
//		slog.Error("No response handler registered",
//			"op", response.Operation)
//	} else {
//		err = svc.serverResponseHandler(response)
//	}
//	svc.writeReply(w, nil, "", err)
//}

// HandleNotification receive a notification message from an agent and pass it on to the server handler.
func (svc *HttpTransportServer) HandleNotification(w http.ResponseWriter, r *http.Request) {
	rp, err := GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	msg := transports.NewNotificationMessage(rp.Op, rp.ThingID, rp.Name, rp.Data)
	if svc.serverNotificationHandler == nil {
		slog.Error("HandleNotification not registered")
	} else {
		svc.serverNotificationHandler(msg)
	}
	svc.writeReply(w, nil, "", err)
}

// HandleRequestMessage handles requests that expect a response.
// This first builds a ThingMessage instance containing the connectionID, correlationID,
// operation and payload; Next it passes this to the registered handler for processing.
// Finally, the result is included in the response payload.
//
// Note: If result is async then the response will be sent separately by agent using an
// ActionStatus message.
func (svc *HttpTransportServer) HandleRequestMessage(w http.ResponseWriter, r *http.Request) {

	var response transports.ResponseMessage
	rp, err := GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// an action request should have a cid when used with SSE.
	// without a connection-id this request can not receive an async reply
	if r.Header.Get(ConnectionIDHeader) == "" {
		slog.Info("HandleRequestMessage request has no 'cid' header.",
			"clientID", rp.ClientID, "op", rp.Op)
	}

	// pass the event to the digitwin service for further processing
	correlationID := r.Header.Get(CorrelationIDHeader)
	if correlationID == "" {
		correlationID = shortid.MustGenerate()
	}

	request := transports.NewRequestMessage(rp.Op, rp.ThingID, rp.Name, rp.Data, correlationID)
	request.SenderID = rp.ClientID

	// ping is handled internally
	if rp.Op == wot.HTOpPing {
		// regular http server returns with pong -
		// used only when no sub-protocol is used as return channel
		response = request.CreateResponse("pong", nil)
	} else if svc.serverRequestHandler == nil {
		slog.Error("No request handler registered")
		response = request.CreateResponse("", errors.New("no request handler registered"))
	} else {
		// forward the request to the internal handler for further processing.
		// If a result is available immediately, it will be embedded into the http
		// response body, otherwise a status pending is returned.
		response = svc.serverRequestHandler(request, rp.ConnectionID)
	}
	replyHeader := w.Header()
	if replyHeader == nil {
		// this happened a few times during testing. perhaps a broken connection while debugging?
		err = fmt.Errorf("HandleRequest: Can't return result."+
			" Write header is nil. This is unexpected. clientID='%s", rp.ClientID)
		svc.writeError(w, err, http.StatusInternalServerError)
		return
	}
	// hiveot used headers
	replyHeader.Set(CorrelationIDHeader, correlationID)

	// progress is complete, return the default output
	svc.writeReply(w, response.Output, response.Status, err)
}

// HandleInvokeAction requests an action from the digital twin.
// NOTE: This returns a header with a dataschema if a schema from
// additionalResponses is returned.
//
// The sender must include the connection-id header of the connection it wants to
// receive the response.
//func (svc *HttpTransportServer) HandleInvokeAction(w http.ResponseWriter, r *http.Request) {
//
//	svc.HandleRequestMessage(wot.OpInvokeAction, w, r)
//}

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
	svc.writeReply(w, reply, transports.StatusCompleted, nil)
}

// HandleLoginRefresh refreshes the auth token using the session authenticator.
// The session authenticator is that of the authn service. This allows testing with a dummy
// authenticator without having to run the authn service.
func (svc *HttpTransportServer) HandleLoginRefresh(w http.ResponseWriter, r *http.Request) {
	var newToken string
	var oldToken string
	rp, err := GetRequestParams(r)
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
	svc.writeReply(w, newToken, transports.StatusCompleted, nil)
}

// HandleLogout ends the session and closes all client connections
func (svc *HttpTransportServer) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// use the authenticator
	rp, err := GetRequestParams(r)
	if err == nil {
		svc.authenticator.Logout(rp.ClientID)
	}
	svc.writeReply(w, nil, transports.StatusCompleted, err)
}

// HandlePing with http handler of ping as a request
func (svc *HttpTransportServer) HandlePing(w http.ResponseWriter, r *http.Request) {
	// simply return a pong message
	rp, err := GetRequestParams(r)
	replyHeader := w.Header()
	replyHeader.Set(CorrelationIDHeader, rp.CorrelationID)
	svc.writeReply(w, "pong", transports.StatusCompleted, err)

	//svc.HandleRequestMessage(wot.HTOpPing, w, r)
}

//// HandlePublishEvent update digitwin with event published by agent
//// FIXME: remove from http basic? this only works agents
//func (svc *HttpTransportServer) HandlePublishEvent(w http.ResponseWriter, r *http.Request) {
//	svc.HandleNotification(wot.HTOpEvent, w, r)
//}

//// HandleQueryAction returns a list of latest action requests of a Thing
//// Parameters: thingID
//func (svc *HttpTransportServer) HandleQueryAction(w http.ResponseWriter, r *http.Request) {
//	svc.HandleRequestMessage(wot.OpQueryAction, w, r)
//}
//
//// HandleQueryAllActions returns a list of latest action requests of a Thing
//// Parameters: thingID
//func (svc *HttpTransportServer) HandleQueryAllActions(w http.ResponseWriter, r *http.Request) {
//	svc.HandleRequestMessage(wot.OpQueryAllActions, w, r)
//}
//
//// HandleReadAllEvents returns a list of latest event values from a Thing
//// Parameters: thingID
//func (svc *HttpTransportServer) HandleReadAllEvents(w http.ResponseWriter, r *http.Request) {
//	svc.HandleRequestMessage(wot.HTOpReadAllEvents, w, r)
//}
//
//// HandleReadAllProperties was added to the top level TD form. Handle it here.
//func (svc *HttpTransportServer) HandleReadAllProperties(w http.ResponseWriter, r *http.Request) {
//	svc.HandleRequestMessage(wot.OpReadAllProperties, w, r)
//}
//
//// HandleReadEvent returns the latest event value from a Thing
//// Parameters: {thingID}, {name}
//func (svc *HttpTransportServer) HandleReadEvent(w http.ResponseWriter, r *http.Request) {
//	svc.HandleRequestMessage(wot.HTOpReadEvent, w, r)
//}
//
//func (svc *HttpTransportServer) HandleReadProperty(w http.ResponseWriter, r *http.Request) {
//	svc.HandleRequestMessage(wot.OpReadProperty, w, r)
//}
//
//// HandleReadTD returns the TD of a thing in the directory
//// URL parameter {thingID}
//func (svc *HttpTransportServer) HandleReadTD(w http.ResponseWriter, r *http.Request) {
//	svc.HandleRequestMessage(wot.HTOpReadTD, w, r)
//}

// HandleReadAllTDs returns the list of digital twin TDs in the directory.
// this is a REST api for convenience. Consider using directory action instead.
//func (svc *HttpTransportServer) HandleReadAllTDs(w http.ResponseWriter, r *http.Request) {
//	svc.HandleRequestMessage(wot.HTOpReadAllTDs, w, r)
//}
//
//// HandlePublishMultipleProperties agent sends a map with multiple property
//// FIXME: remove from http basic? this only works with sse[-sc]
//func (svc *HttpTransportServer) HandlePublishMultipleProperties(w http.ResponseWriter, r *http.Request) {
//	svc.HandleNotification(wot.HTOpUpdateMultipleProperties, w, r)
//}
//
//// HandlePublishProperty agent sends single or multiple property updates
//// FIXME: remove from http basic? this only works with sse[-sc]
//func (svc *HttpTransportServer) HandlePublishProperty(w http.ResponseWriter, r *http.Request) {
//	// this
//	svc.HandleNotification(wot.HTOpUpdateProperty, w, r)
//}
//
//// HandlePublishTD agent sends a new TD document
//// FIXME: remove from http basic? this only works with hiveot
//func (svc *HttpTransportServer) HandlePublishTD(w http.ResponseWriter, r *http.Request) {
//	svc.HandleNotification(wot.HTOpUpdateTD, w, r)
//}

// HandleWriteProperty consumer requests to update a Thing property
//func (svc *HttpTransportServer) HandleWriteProperty(w http.ResponseWriter, r *http.Request) {
//	svc.HandleRequestMessage(wot.OpWriteProperty, w, r)
//}
