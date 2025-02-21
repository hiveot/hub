package hiveotsseserver

import (
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/wot"
	"log/slog"
	"net/http"
)

// routes for handling http server requests

// HTTP endpoint that accepts HiveOT RequestMessage envelopes
//const HiveOTPostRequestHRef = "/hiveot/request"

// HTTP endpoint that accepts HiveOT ResponseMessage envelopes
//const HiveOTPostResponseHRef = "/hiveot/response"

const HiveOTGetSseConnectHRef = "/hiveot/sse-sc"

// CreateRoutes add the routes used in SSE-SC sub-protocol
// This is simple, one endpoint to connect, and one to pass requests, using URI variables
func (srv *HiveotSseServer) CreateRoutes() {
	// Connect serves the SSE-SC protocol
	srv.httpTransport.AddOps(nil, []string{SSEOpConnect},
		http.MethodGet, srv.ssePath, srv.Serve)

	// Handle request messages using a single path with URI variables.
	srv.httpTransport.AddOps(nil,
		[]string{"*"},
		http.MethodPost, DefaultHiveotPostRequestHRef, srv.HandleRequestMessage)

	// Handle response messages from agents, containing a response message envelope.
	srv.httpTransport.AddOps(nil,
		[]string{"*"},
		http.MethodPost, DefaultHiveotPostResponseHRef, srv.HandleResponseMessage)
}

// HandleRequestMessage handles request messages sent by consumers or agents.
//
// This endpoint only handles requests when an SSE connection is already established.
//
// This locates the corresponding connection and passes the request to the connection
// to make it seem like the connection received the request message itself.
// The connection then processes the request, handles subscriptions, or forwards
// the request to the connection subscriber, which by default is this server.
//
// Note: If the result status isn't completed or failed then a separate response
// message will be sent asynchronously by the agent, containing an ActionStatus message payload.
func (srv *HiveotSseServer) HandleRequestMessage(w http.ResponseWriter, r *http.Request) {
	var output any
	var status string
	var req transports.RequestMessage

	// 1. Decode the request message
	rp, err := httpserver.GetRequestParams(r, &req)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// Use the authenticated clientID as the sender
	req.SenderID = rp.ClientID
	connectionID := rp.ConnectionID

	// FIXME: handle ping in http basic
	// 1. handle ping internally??
	if req.Operation == wot.HTOpPing {
		//resp := req.CreateResponse("pong", nil)
		output = "pong"
		status = transports.StatusCompleted
		err = nil
	} else {
		// locate the connection that handles the request.
		c := srv.GetSseConnection(req.SenderID, connectionID)
		if c == nil {
			// When using the sse subprotocol endpoint this is an error.
			err = fmt.Errorf("HandleRequestMessage: no corresponding connection")
			status = transports.StatusFailed
			slog.Error("HandleRequestMessage. No connection to handle the request.",
				"clientID", rp.ClientID, "connectionID", connectionID,
				"correlationID", req.CorrelationID)
		} else {
			// 3. pass it on to the application
			output, status, err = c.onRequestMessage(&req)
		}
	}
	// 3. Return the response
	replyHeader := w.Header()
	if replyHeader == nil {
		// this happened a few times during testing. perhaps a broken connection while debugging?
		err = fmt.Errorf("HandleRequest: Can't return result."+
			" Write header is nil. This is unexpected. clientID='%s", rp.ClientID)
		srv.httpTransport.WriteError(w, err, http.StatusInternalServerError)
		return
	}
	// hiveot used headers for interoperability
	//replyHeader.Set(httpserver.CorrelationIDHeader, req.CorrelationID)

	// progress is complete, return the default output
	resp := req.CreateResponse(output, err)
	resp.Status = status
	//respJSON, _ := jsoniter.MarshalToString(resp)
	srv.httpTransport.WriteReply(w, resp, status, err)
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
	resp := transports.ResponseMessage{}
	resp.MessageType = transports.MessageTypeResponse

	// 1. Decode the request message
	rp, err := httpserver.GetRequestParams(r, &resp)
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
	srv.httpTransport.WriteReply(w, nil, "", err)
}

// Serve a new incoming hiveot sse connection.
// This doesn't return until the connection is closed by either client or server.
func (srv *HiveotSseServer) Serve(w http.ResponseWriter, r *http.Request) {

	//An active session is required before accepting the request. This is created on
	//authentication/login. Until then SSE cm are blocked.
	rp, err := httpserver.GetRequestParams(r, nil)

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
