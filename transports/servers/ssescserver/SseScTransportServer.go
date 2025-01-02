package ssescserver

import (
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/connections"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
	"net/http"
	"sync"
)

const SSEOpConnect = "sse-connect"

// SseScTransportServer is a subprotocol binding server of http
//
// This server supports both the SSE (TODO) and the SSE-SC sub-protocols.
//
// The SSE subprotocol provides event/property resource in the path.
// See also https://w3c.github.io/wot-profile/#sec-http-sse-profile
// The SSE event 'event' field contains the event or property affordance name.
//
// The SSE-SC extension is automatically enabled if the SSE connection is made
// on the ssesc base path. Clients subscribe and observe methods to make
// REST calls to subscribe and unsubscribe.
// An SSE-SC event 'event' field contains the operation name while the ID field
// contains the concatenation of operation/thingID/affordance name.
// (todo: look into using operation/thingID/affordance in the event field instead
// so it is closer to the SSE spec)
//
// For consideration is a third variant that uses message envelopes similar to
// websockets.
type SseScTransportServer struct {
	// connection manager to add/remove connections
	cm *connections.ConnectionManager

	httpTransport *httpserver.HttpTransportServer

	// mutex for updating connections
	mux sync.RWMutex
}

func (svc *SseScTransportServer) AddTDForms(tdi *td.TD) error {
	// forms are handled through the http binding
	return svc.httpTransport.AddTDForms(tdi)
}

// GetForm returns a new SSE form for the given operation
// this returns the http form
func (svc *SseScTransportServer) GetForm(op, thingID, name string) td.Form {
	// forms are handled through the http binding
	return svc.httpTransport.GetForm(op, thingID, name)
}

// GetConnectURL returns SSE connection path of the server
func (svc *SseScTransportServer) GetConnectURL() string {
	return svc.httpTransport.GetConnectURL() + httpserver.DefaultSSESCPath
}

// GetSseConnection returns the SSE Connection with the given ID
// This returns nil if not found or if the connectionID is not
func (svc *SseScTransportServer) GetSseConnection(connectionID string) *SseScServerConnection {
	c := svc.cm.GetConnectionByConnectionID(connectionID)
	if c == nil {
		return nil
	}
	sseConn, isValid := c.(*SseScServerConnection)
	if !isValid {
		return nil
	}
	return sseConn
}

// HandleConnect handles a new sse-sc connection.
// This doesn't return until the connection is closed by either client or server.
func (svc *SseScTransportServer) HandleConnect(w http.ResponseWriter, r *http.Request) {

	//An active session is required before accepting the request. This is created on
	//authentication/login. Until then SSE connections are blocked.
	clientID, err := httpserver.GetClientIdFromContext(r)

	if err != nil {
		slog.Warn("SSESC HandleConnect. No session available yet, telling client to delay retry to 10 seconds",
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
	// SSE-SC clients include a connection-ID header to link subscriptions to this
	// connection. This is prefixed with "{clientID}-" to ensure uniqueness and
	// prevent connection hijacking.
	cid := r.Header.Get(httpserver.ConnectionIDHeader)

	// add the new sse connection
	sseFallback := false // TODO
	c := NewSSEConnection(clientID, cid, r.RemoteAddr, sseFallback)

	err = svc.cm.AddConnection(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	// don't return until the connection is closed
	c.Serve(w, r)

	// finally cleanup the connection
	svc.cm.RemoveConnection(c.GetConnectionID())
}

//// HandlePing responds with a pong reply message on the SSE return channel
//func (svc *SseScTransportServer) HandlePing(w http.ResponseWriter, r *http.Request) {
//	rp, _ := httpserver.GetRequestParams(r)
//
//	c := svc.GetSseConnection(rp.ConnectionID)
//	if c == nil {
//		http.Error(w, "Missing or unknown connection ID", http.StatusBadRequest)
//		return
//	}
//	resp := transports.NewResponseMessage(wot.HTOpPong, "", "", "pon", nil, rp.CorrelationID)
//	_ = c._send(transports.MessageTypeResponse, resp)
//}

// HandleSubscriptions (un)subscribe events or (un)observe properties
func (svc *SseScTransportServer) HandleSubscriptions(w http.ResponseWriter, r *http.Request) {
	rp, err := httpserver.GetRequestParams(r)
	if err != nil {
		slog.Warn("HandleSubscriptions", "err", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	c := svc.GetSseConnection(rp.ConnectionID)
	if c == nil {
		slog.Error("HandleSubscribeEvent: no matching connection found",
			"clientID", rp.ClientID, "connID", rp.ConnectionID)
	}
	switch rp.Op {
	case wot.OpSubscribeEvent, wot.OpSubscribeAllEvents:
		c.SubscribeEvent(rp.ThingID, rp.Name)
	case wot.OpUnsubscribeEvent, wot.OpUnsubscribeAllEvents:
		c.UnsubscribeEvent(rp.ThingID, rp.Name)
	case wot.OpObserveProperty, wot.OpObserveAllProperties:
		c.ObserveProperty(rp.ThingID, rp.Name)
	case wot.OpUnobserveProperty, wot.OpUnobserveAllProperties:
		c.UnobserveProperty(rp.ThingID, rp.Name)
	}
}

// SendNotification broadcast an event or property change to subscribers clients
func (svc *SseScTransportServer) SendNotification(notification transports.NotificationMessage) {
	cList := svc.cm.GetConnectionByProtocol(transports.ProtocolTypeSSESC)
	for _, c := range cList {
		c.SendNotification(notification)
	}
}

func (svc *SseScTransportServer) Stop() {
	//Close all incoming SSE connections
	svc.cm.CloseAll()
}

// StartSseScTransportServer returns a new SSE-SC sub-protocol binding.
// This is only a 1-way binding that adds an SSE based return channel to the http binding.
//
// This adds http methods for (un)subscribing to events and properties and
// adds new connections to the connection manager for callbacks.
//
// If no ssePath is provided, the default DefaultSSESCPath (/ssesc) is used
func StartSseScTransportServer(
	ssePath string,
	cm *connections.ConnectionManager,
	httpTransport *httpserver.HttpTransportServer,
) *SseScTransportServer {
	if ssePath == "" {
		ssePath = httpserver.DefaultSSESCPath
	}
	b := &SseScTransportServer{
		cm:            cm,
		httpTransport: httpTransport,
	}
	httpTransport.AddOps(nil, []string{SSEOpConnect},
		http.MethodGet, ssePath, b.HandleConnect)
	httpTransport.AddOps(nil, []string{
		wot.OpObserveProperty, wot.OpObserveAllProperties,
		wot.OpSubscribeEvent, wot.OpSubscribeAllEvents,
		wot.OpUnobserveProperty, wot.OpUnobserveAllProperties,
		wot.OpUnsubscribeEvent, wot.OpUnsubscribeAllEvents},
		http.MethodPost, ssePath+"/{operation}/{thingID}", b.HandleSubscriptions)

	return b
}
