package hiveotsseserver

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/servers/httpbasic"
	"github.com/hiveot/hub/messaging/tputils/tlsserver"
	"github.com/hiveot/hub/wot"
)

// routes for handling http server requests

// HiveOTPostResponseHRef is the HTTP path that accepts HiveOT ResponseMessage envelopes
// intended for agents that post responses.
//const HiveOTPostResponseHRef = "/hiveot/response"
//const HiveOTGetSseConnectHRef = "/hiveot/sse-sc"

// CreateRoutes add the routes used in SSE-SC sub-protocol
// This is simple, one endpoint to connect, and one to pass requests, using URI variables
func (srv *HiveotSseServer) CreateRoutes(ssePath string, r chi.Router) {
	r.Get(ssePath, srv.Serve)
	r.Post(DefaultHiveotPostNotificationHRef, srv.HandleNotificationMessage)
	r.Post(DefaultHiveotPostRequestHRef, srv.HandleRequestMessage)
	r.Post(DefaultHiveotPostResponseHRef, srv.HandleResponseMessage)

	// Connect serves the SSE-SC protocol
	//srv.httpBasicServer.AddOps(nil, []string{SSEOpConnect},
	//	http.MethodGet, srv.ssePath, srv.Serve)
	//
	//// Handle notification messages from agents, containing a notification message envelope.
	//srv.httpBasicServer.AddOps(nil,
	//	[]string{"*"},
	//	http.MethodPost, DefaultHiveotPostNotificationHRef, srv.HandleNotificationMessage)
	//
	//// Handle request messages using a single path with URI variables.
	//srv.httpBasicServer.AddOps(nil,
	//	[]string{"*"},
	//	http.MethodPost, DefaultHiveotPostRequestHRef, srv.HandleRequestMessage)
	//
	//// Handle response messages from agents, containing a response message envelope.
	//srv.httpBasicServer.AddOps(nil,
	//	[]string{"*"},
	//	http.MethodPost, DefaultHiveotPostResponseHRef, srv.HandleResponseMessage)
}

// DeleteRoutes removes the routes used in SSE-SC sub-protocol
func (srv *HiveotSseServer) DeleteRoutes(ssePath string, r chi.Router) {
	r.Delete(ssePath, srv.Serve)
	r.Delete(DefaultHiveotPostNotificationHRef, srv.HandleNotificationMessage)
	r.Delete(DefaultHiveotPostRequestHRef, srv.HandleRequestMessage)
	r.Delete(DefaultHiveotPostResponseHRef, srv.HandleResponseMessage)
}

// HandleNotificationMessage handles responses sent by agents.
//
// As WoT doesn't support reverse connections this is only used by hiveot agents
// that connect as clients. In that case the server is the consumer.
//
// This receives a NotificationMessage envelope and passes it to the corresponding
// connection as if the connection received the response itself.
//
// Message flow: agent POST response -> server forwards to -> connection ->
// forwards to subscriber (which is the server again, or a consumer)
//
// The message body is unmarshalled and included as the response.
func (srv *HiveotSseServer) HandleNotificationMessage(w http.ResponseWriter, r *http.Request) {
	notif := messaging.NotificationMessage{}
	notif.MessageType = messaging.MessageTypeNotification

	// 1. Decode the message
	rp, err := httpbasic.GetRequestParams(r, &notif)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	} else if notif.Operation == "" {
		err = fmt.Errorf("HandleResponseMessage: missing ResponseMessage in payload")
		slog.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	notif.SenderID = rp.ClientID

	// A NotificationMessage was received but the client doesn't have an SSE connection
	// http requests without connectionID should not receive responses.
	// why is there a connectionID?
	c := srv.GetSseConnection(rp.ClientID, rp.ConnectionID)
	if c == nil {
		// this is possible when:
		// 1. a connectionID is provided in a request but no SSE has been established.
		//    if the connectionID is provided you'd expect an SSE connection so this is a warning
		// 2. the server restarted and the client reconnects with the same connectionID?
		//    hmm, did the agent sse connection get lost?
		slog.Warn("HandleNotificationMessage. No connection to handle the notification/subscription.",
			"senderID", rp.ClientID,
			"connectionID", rp.ConnectionID,
			"thingID", rp.ThingID,
			"name", rp.Name,
			"operation", rp.Op,
		)
	} else {
		// pass it on to the server notification flow handler
		h := c.notificationHandlerPtr.Load()
		if h != nil {
			(*h)(&notif)
		}
	}

	tlsserver.WriteReply(w, true, nil, err)
}

// HandleRequestMessage handles request messages sent by consumers or agents.
//
// This endpoint only handles requests for the SSE subprotocol.
// An SSE connection must already be established.
//
// This locates the corresponding connection and passes the request to the connection
// to make it seem like the connection received the request message itself.
// The connection then processes the request, handles subscriptions, or forwards
// the request to the connection subscriber, which by default is this server.
//
// Note: If the result status isn't completed or failed then a separate response
// message will be sent asynchronously by the agent, containing an ActionStatus message payload.
func (srv *HiveotSseServer) HandleRequestMessage(w http.ResponseWriter, r *http.Request) {
	var output *messaging.ResponseMessage
	var handled bool
	var req messaging.RequestMessage

	// 1. Decode the request message
	rp, err := httpbasic.GetRequestParams(r, &req)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// Use the authenticated clientID as the sender
	req.SenderID = rp.ClientID
	connectionID := rp.ConnectionID

	// 1. handle ping operation internally
	if req.Operation == wot.HTOpPing {
		//resp := req.CreateResponse("pong", nil)
		output = req.CreateResponse("pong", nil)
		handled = true
		err = nil
		// debugger bug not stopping on WriteReply when at the bottom?
	} else {
		// 2. locate the connection that handles the request.
		c := srv.GetSseConnection(req.SenderID, connectionID)
		if c == nil {
			// When using the sse subprotocol endpoint this is an error.
			err = fmt.Errorf("HandleRequestMessage: no corresponding connection")
			handled = true
			slog.Error("HandleRequestMessage. No connection to handle the request.",
				"clientID", rp.ClientID, "connectionID", connectionID,
				"correlationID", req.CorrelationID)
		} else {
			// 3. pass it on to the application
			handled, output, err = c.onRequestMessage(&req)
		}
	}
	// 4. Return the response
	tlsserver.WriteReply(w, handled, output, err)
}

// HandleResponseMessage handles responses sent by agents.
//
// As WoT doesn't support reverse connections this is only used by hiveot agents
// that connect as clients. In that case the server is the consumer.
//
// This receives a ResponseMessage envelope and passes it to the corresponding
// connection as if the connection received the response itself.
//
// Message flow: agent POST response -> server forwards to -> connection ->
// forwards to subscriber (which is the server again, or a consumer)
//
// The message body is unmarshalled and included as the response.
func (srv *HiveotSseServer) HandleResponseMessage(w http.ResponseWriter, r *http.Request) {
	resp := messaging.ResponseMessage{}
	resp.MessageType = messaging.MessageTypeResponse

	// 1. Decode the request message
	rp, err := httpbasic.GetRequestParams(r, &resp)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	} else if resp.Operation == "" {
		err = fmt.Errorf("HandleResponseMessage: missing ResponseMessage in payload")
		slog.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//resp := transports.NewResponseMessage(rp.Op, rp.ThingID, rp.Name, rp.Data, err, rp.CorrelationID)
	resp.SenderID = rp.ClientID

	// FIXME: a ResponseMessage (notification) was received but the client doesnt have an SSE connection
	// http requests without connectionID should not receive responses.
	// why is there a connectionID?
	c := srv.GetSseConnection(rp.ClientID, rp.ConnectionID)
	if c == nil {
		// this is possible when a connectionID is provided in a request but no SSE
		// has been established. Not an error.
		//err = fmt.Errorf("HandleResponseMessage: no corresponding connection")
		slog.Info("HandleResponseMessage. No connection to handle the response/subscription.",
			"clientID", rp.ClientID,
			"connectionID", rp.ConnectionID,
			"name", rp.Name,
		)
	} else {
		h := c.responseHandlerPtr.Load()
		if h != nil {
			err = (*h)(&resp)
		}
	}
	//if srv.serverResponseHandler == nil {
	//	err = fmt.Errorf("No response handler registered for operation '%s'", rp.Op)
	//} else {
	//	// forward the response to the internal handler for further processing.
	//	// If a result is available immediately, it will be embedded into the http
	//	// response body, otherwise a status pending is returned.
	//	err = srv.serverResponseHandler(resp)
	//	if resp.Error != "" {
	//		err = errors.New(resp.Error)
	//	}
	//}
	//
	tlsserver.WriteReply(w, true, nil, err)
}

// Serve a new incoming hiveot sse connection.
// This doesn't return until the connection is closed by either client or server.
func (srv *HiveotSseServer) Serve(w http.ResponseWriter, r *http.Request) {

	//An active session is required before accepting the request. This is created on
	//authentication/login. Until then SSE cm are blocked.
	rp, err := httpbasic.GetRequestParams(r, nil)

	if err != nil {
		slog.Warn("SSESC Serve. No session available yet, telling client to delay retry to 10 seconds",
			"remoteAddr", r.RemoteAddr)

		// set retry to a large number
		// see https://javascript.info/server-sent-events#reconnection
		errMsg := fmt.Sprintf("retry: %s\nevent:%s\n\n",
			"10000", "logout")
		http.Error(w, errMsg, http.StatusUnauthorized)
		//w.Write([]byte(errMsg))
		w.(http.Flusher).Flush()
		return
	}
	// add the new sse connection
	sseFallback := false // TODO
	// the sse connection can only be used to *send* messages to the remote client
	c := NewHiveotSseConnection(
		rp.ClientID, rp.ConnectionID, r.RemoteAddr, r, sseFallback)

	// By default the server collects the requests/responses to pass it to subscribers
	// If a consumer takes over the connection (connection reversal) it will register
	// its own handlers.
	c.SetNotificationHandler(srv.serverNotificationHandler)
	c.SetRequestHandler(srv.serverRequestHandler)
	c.SetResponseHandler(srv.serverResponseHandler)
	err = srv.cm.AddConnection(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	// don't return until the connection is closed
	c.Serve(w, r)

	// finally cleanup the connection
	srv.cm.RemoveConnection(c)
}
