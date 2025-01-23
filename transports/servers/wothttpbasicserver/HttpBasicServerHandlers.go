// Package httpserver with handlers for the http protocol
package wothttpbasicserver

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/servers/httpserver"
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

// HandleRequestMessage handles requests that expect a response.
// This first builds a RequestMessage envelope; Next it passes this to the registered
// handler for processing. Finally, the result is included in the response payload.
//
// Note: If result is async then the response will be sent separately by agent using an
// ActionStatus message.
func (svc *httpserver.HttpTransportServer) HandleRequestMessage(w http.ResponseWriter, r *http.Request) {

	var response transports.ResponseMessage
	rp, err := httpserver.GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// an action request should have a cid when used with SSE.
	// without a connection-id this request can not receive an async reply
	if r.Header.Get(httpserver.ConnectionIDHeader) == "" {
		slog.Info("HandleRequestMessage request has no 'cid' header.",
			"clientID", rp.ClientID, "op", rp.Op)
	}

	// pass the event to the digitwin service for further processing
	correlationID := r.Header.Get(httpserver.CorrelationIDHeader)
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
	replyHeader.Set(httpserver.CorrelationIDHeader, correlationID)

	// progress is complete, return the default output
	svc.writeReply(w, response.Output, response.Status, err)
}
