package ssescserver

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/clients/httpbinding"
	"github.com/hiveot/hub/wot/transports/connections"
	"github.com/hiveot/hub/wot/transports/servers/httpserver"
	"github.com/hiveot/hub/wot/transports/servers/httpserver/httpcontext"
	"log/slog"
	"net/http"
	"sync"
)

const DefaultSSESCPath = "/ssesc"

// SseScTransportServer is a subprotocol binding server of http
type SseScTransportServer struct {
	// connection manager to add/remove connections
	cm *connections.ConnectionManager

	httpTransport *httpserver.HttpTransportServer

	// mutex for updating connections
	mux sync.RWMutex
}

func (svc *SseScTransportServer) AddTDForms(td *tdd.TD) error {
	// forms are handled through the http binding
	return svc.httpTransport.AddTDForms(td)
}

// GetForm returns a new SSE form for the given operation
// this returns the http form
func (svc *SseScTransportServer) GetForm(op string) tdd.Form {
	// forms are handled through the http binding
	return svc.httpTransport.GetForm(op)
}

// GetProtocolInfo returns info on the protocol supported by this binding
func (svc *SseScTransportServer) GetProtocolInfo() transports.ProtocolInfo {
	return svc.httpTransport.GetProtocolInfo()
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
	clientID, err := httpcontext.GetClientIdFromContext(r)

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
	cid := r.Header.Get(httpbinding.ConnectionIDHeader)

	// add the new sse connection
	c := NewSSEConnection(clientID, cid, r.RemoteAddr)

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

// HandleObserveAllProperties adds a property subscription
func (svc *SseScTransportServer) HandleObserveAllProperties(w http.ResponseWriter, r *http.Request) {
	svc.HandleObserveProperty(w, r)
}

// HandleObserveProperty handles a property observe request for one or all properties
func (svc *SseScTransportServer) HandleObserveProperty(w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		slog.Warn("HandleObserveProperty", "err", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Info("HandleObserveProperty",
		slog.String("clientID", rp.ClientID),
		slog.String("thingID", rp.ThingID),
		slog.String("name", rp.Name))

	c := svc.GetSseConnection(rp.ConnectionID)
	if c != nil {
		c.ObserveProperty(rp.ThingID, rp.Name)
	} else {
		slog.Error("HandleObserveProperty: no matching connection found",
			"clientID", rp.ClientID, "connectionID", rp.ConnectionID)
	}
}

// HandleSubscribeAllEvents adds a subscription to all events
func (svc *SseScTransportServer) HandleSubscribeAllEvents(w http.ResponseWriter, r *http.Request) {
	svc.HandleSubscribeEvent(w, r)
}

// HandleSubscribeEvent handles a subscription request for one or all events
func (svc *SseScTransportServer) HandleSubscribeEvent(w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		slog.Warn("HandleSubscribe", "err", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Info("HandleSubscribe",
		slog.String("clientID", rp.ClientID),
		slog.String("connectionID", rp.ConnectionID),
		slog.String("thingID", rp.ThingID),
		slog.String("name", rp.Name))

	c := svc.GetSseConnection(rp.ConnectionID)
	if c != nil {
		c.SubscribeEvent(rp.ThingID, rp.Name)
	} else {
		slog.Error("HandleSubscribeEvent: no matching connection found",
			"clientID", rp.ClientID, "connID", rp.ConnectionID)
	}
}

// HandleUnobserveAllProperties handles removal of all property observe subscriptions
func (svc *SseScTransportServer) HandleUnobserveAllProperties(w http.ResponseWriter, r *http.Request) {
	svc.HandleUnobserveProperty(w, r)
}

// HandleUnobserveProperty handles removal of one property observe subscriptions
func (svc *SseScTransportServer) HandleUnobserveProperty(w http.ResponseWriter, r *http.Request) {
	slog.Info("HandleUnobserveProperty")
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	c := svc.GetSseConnection(rp.ConnectionID)
	if err == nil {
		c.UnobserveProperty(rp.ThingID, rp.Name)
	}
}

// HandleUnsubscribeAllEvents removes the subscription
func (svc *SseScTransportServer) HandleUnsubscribeAllEvents(w http.ResponseWriter, r *http.Request) {
	svc.HandleUnsubscribeEvent(w, r)
}

// HandleUnsubscribeEvent handles removal of one or all event subscriptions
func (svc *SseScTransportServer) HandleUnsubscribeEvent(w http.ResponseWriter, r *http.Request) {
	slog.Info("HandleUnsubscribeEvent")
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	c := svc.GetSseConnection(rp.ConnectionID)
	if err == nil {
		c.UnsubscribeEvent(rp.ThingID, rp.Name)
	}
}

func (svc *SseScTransportServer) Stop() {
	// nothing to do here as this runs on top of the http server
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
		ssePath = DefaultSSESCPath
	}
	b := &SseScTransportServer{
		cm:            cm,
		httpTransport: httpTransport,
	}
	httpTransport.AddGetOp("connect",
		ssePath, b.HandleConnect)
	httpTransport.AddPostOp(vocab.OpObserveAllProperties,
		ssePath+"/digitwin/observe/{thingID}", b.HandleObserveAllProperties)
	httpTransport.AddPostOp(vocab.OpSubscribeAllEvents,
		ssePath+"/digitwin/subscribe/{thingID}", b.HandleSubscribeAllEvents)
	httpTransport.AddPostOp(vocab.OpSubscribeEvent,
		ssePath+"/digitwin/subscribe/{thingID}/{name}", b.HandleSubscribeEvent)
	httpTransport.AddPostOp(vocab.OpObserveProperty,
		ssePath+"/digitwin/observe/{thingID}/{name}", b.HandleObserveProperty)
	httpTransport.AddPostOp(vocab.OpUnobserveAllProperties,
		ssePath+"/digitwin/unobserve/{thingID}", b.HandleUnobserveAllProperties)
	httpTransport.AddPostOp(vocab.OpUnobserveProperty,
		ssePath+"/digitwin/unobserve/{thingID}/{name}", b.HandleUnobserveProperty)
	httpTransport.AddPostOp(vocab.OpUnsubscribeAllEvents,
		ssePath+"/digitwin/unsubscribe/{thingID}", b.HandleUnsubscribeAllEvents)
	httpTransport.AddPostOp(vocab.OpUnsubscribeEvent,
		ssePath+"/digitwin/unsubscribe/{thingID}/{name}", b.HandleUnsubscribeEvent)
	return b
}
