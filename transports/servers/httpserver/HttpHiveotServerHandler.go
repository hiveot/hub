// Package httpserver containing the hiveot protocol handlers
package httpserver

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"net/http"
)

// HandleHiveotRequest is the hiveot simplified messaging handler for handling
// client requests.
// The payload is a RequestMessage envelope that contains all request information.
//
// This complements the hiveot response and notification handlers which are used by
// agents as the WoT Http binding doesn't cater to thing agents connected as clients.
func (svc *HttpTransportServer) HandleHiveotRequest(w http.ResponseWriter, r *http.Request) {

	var response transports.ResponseMessage
	clientID, connID, payload, err := GetHiveotParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	request := transports.RequestMessage{}
	err = jsoniter.Unmarshal(payload, &request)
	// enforce a correct sender ID
	request.SenderID = clientID

	// ping and auth are handled internally
	if request.Operation == wot.HTOpPing {
		// regular http server returns with pong -
		// used only when no sub-protocol is used as return channel
		response = request.CreateResponse("pong", nil)
	} else if request.Operation == wot.HTOpRefresh {
		oldToken := request.ToString()
		newToken, err := svc.authenticator.RefreshToken(
			request.SenderID, request.SenderID, oldToken)
		response = request.CreateResponse(newToken, err)
	} else if svc.serverRequestHandler == nil {
		slog.Error("No request handler registered")
		response = request.CreateResponse("", errors.New("no request handler registered"))
	} else {
		// forward the request to the internal handler for further processing.
		// If a result is available immediately, it will be embedded into the http
		// response body, otherwise a status pending is returned.
		// a return channel with the same connection ID is required.
		response = svc.serverRequestHandler(request, connID)
	}
	replyHeader := w.Header()
	if replyHeader == nil {
		// this happened a few times during testing. perhaps a broken connection while debugging?
		err = fmt.Errorf("HandleActionRequest: Can't return result."+
			" Write header is nil. This is unexpected. clientID='%s", clientID)
		svc.writeError(w, err, http.StatusInternalServerError)
		return
	}
	// progress is complete, return the default output
	svc.writeReply(w, response, response.Status, err)
}

// HandleHiveotResponse is the hiveot simplified messaging handler for handling
// agent responses.
// The payload is a ResponseMessage envelope that contains all response information.
// Intended for agents to send responses asynchronously.
func (svc *HttpTransportServer) HandleHiveotResponse(w http.ResponseWriter, r *http.Request) {
	var response transports.ResponseMessage
	var status string

	clientID, connID, payload, err := GetHiveotParams(r)
	_ = clientID
	_ = connID
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err = jsoniter.Unmarshal(payload, &response)

	if svc.serverResponseHandler == nil {
		err = fmt.Errorf("No server response handler registered. Response ignored.")
	} else {
		// forward the response to the internal handler for further processing.
		err = svc.serverResponseHandler(clientID, response)
	}
	status = transports.StatusCompleted
	if err != nil {
		status = transports.StatusFailed
	}
	// response handling is complete, return the default output
	svc.writeReply(w, nil, status, err)
}

// HandleHiveotNotification is the hiveot simplified messaging handler for
// handling agent notifications.
// The payload is a ResponseMessage envelope that contains all response information.
// Intended for agents to send responses asynchronously.
func (svc *HttpTransportServer) HandleHiveotNotification(w http.ResponseWriter, r *http.Request) {
	var notification transports.NotificationMessage
	var status string

	clientID, connID, payload, err := GetHiveotParams(r)
	_ = clientID
	_ = connID
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err = jsoniter.Unmarshal(payload, &notification)

	if svc.serverNotificationHandler == nil {
		err = fmt.Errorf("No server notification handler registered. Notification ignored.")
		slog.Error(err.Error())
	} else {
		// forward the response to the internal handler for further processing.
		svc.serverNotificationHandler(clientID, notification)
	}
	status = transports.StatusCompleted
	if err != nil {
		status = transports.StatusFailed
	}
	// response handling is complete, return the default output
	svc.writeReply(w, nil, status, err)
}
