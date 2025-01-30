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
//const HiveOTPostRequestHRef = "/hiveot/request"

// generic HTTP path for sending requests to the server
const HiveOTPostRequestHRef = "/hiveot/request/{operation}/{thingID}/{name}"

// generic HTTP path for agents to send async responses to the server (for use with sse)
const HiveOTPostResponseHRef = "/hiveot/response/{operation}/{thingID}/{name}"

// CreateRoutes add the routes used in SSE-SC sub-protocol
// This is simple, one endpoint to connect, and one to pass requests, using URI variables
func (srv *HiveotSseServer) CreateRoutes() {
	// Connect serves the SSE-SC protocol
	srv.httpTransport.AddOps(nil, []string{SSEOpConnect},
		http.MethodGet, srv.ssePath, srv.Serve)

	// Handle request messages using hiveot request message envelope.
	// Responses are passed to the sse endpoint
	srv.httpTransport.AddOps(nil,
		[]string{"*"},
		http.MethodPost, HiveOTPostRequestHRef, srv.HandleRequestMessage)
}

// HandleRequestMessage handles requests that expect a response.
// This first builds a RequestMessage envelope; Next it passes this to the registered
// handler for processing. Finally, the result is included in the response payload.
//
// Note: If result is async then the response will be sent separately by agent using an
// ActionStatus message.
func (srv *HiveotSseServer) HandleRequestMessage(w http.ResponseWriter, r *http.Request) {

	// 1. Decode the request message
	rp, err := httpserver.GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// 1. handle ping internally
	if rp.Op == wot.HTOpPing {
		srv.httpTransport.WriteReply(w, "pong", transports.StatusCompleted, nil)
		return
	}
	// 2. Handle SSE subscriptions. This needs a SSE connection
	c := srv.GetSseConnection(rp.ConnectionID)
	if c != nil {
		handled := true
		switch rp.Op {
		case wot.OpSubscribeEvent, wot.OpSubscribeAllEvents:
			c.subscriptions.Subscribe(rp.ThingID, rp.Name, rp.CorrelationID)
		case wot.OpUnsubscribeEvent, wot.OpUnsubscribeAllEvents:
			c.subscriptions.Unsubscribe(rp.ThingID, rp.Name)
		case wot.OpObserveProperty, wot.OpObserveAllProperties:
			c.observations.Subscribe(rp.ThingID, rp.Name, rp.CorrelationID)
		case wot.OpUnobserveProperty, wot.OpUnobserveAllProperties:
			c.observations.Unsubscribe(rp.ThingID, rp.Name)
		default:
			handled = false
		}
		if handled {
			srv.httpTransport.WriteReply(w, nil, transports.StatusCompleted, nil)
			return
		}
	}
	// 3. pass it on to the application
	req := transports.NewRequestMessage(rp.Op, rp.ThingID, rp.Name, rp.Data, rp.CorrelationID)
	req.SenderID = rp.ClientID
	var resp *transports.ResponseMessage

	if srv.serverRequestHandler == nil {
		err = fmt.Errorf("No request handler registered for operation '%s'", rp.Op)
		resp = req.CreateResponse(nil, err)
	} else {
		// forward the request to the internal handler for further processing.
		// If a result is available immediately, it will be embedded into the http
		// response body, otherwise a status pending is returned.
		resp = srv.serverRequestHandler(req, c)
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
	// hiveot used headers
	replyHeader.Set(httpserver.CorrelationIDHeader, rp.CorrelationID)

	// progress is complete, return the default output
	srv.httpTransport.WriteReply(w, resp.Output, resp.Status, err)
}

// Serve a new incoming hiveot sse connection.
// This doesn't return until the connection is closed by either client or server.
func (srv *HiveotSseServer) Serve(w http.ResponseWriter, r *http.Request) {

	//An active session is required before accepting the request. This is created on
	//authentication/login. Until then SSE connections are blocked.
	rp, err := httpserver.GetRequestParams(r)

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

	err = srv.connections.AddConnection(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	// don't return until the connection is closed
	c.Serve(w, r)

	// finally cleanup the connection
	srv.connections.RemoveConnection(rp.ConnectionID)
}
