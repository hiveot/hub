package ssesc

import (
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/connections"
	"github.com/hiveot/hub/runtime/transports/httptransport/httpcontext"
	"log/slog"
	"net/http"
	"sync"
)

// SseScBinding is a subprotocol binding
type SseScBinding struct {
	// connection manager to add/remove connections
	cm *connections.ConnectionManager

	// mutex for updating connections
	mux sync.RWMutex
}

// HandleConnect handles a new sse-sc connection.
// This doesn't return until the connection is closed by either client or server.
func (b *SseScBinding) HandleConnect(w http.ResponseWriter, r *http.Request) {

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
	cid := r.Header.Get(hubclient.ConnectionIDHeader)

	// add the new sse connection
	c := NewSSEConnection(clientID, cid, r.RemoteAddr)

	err = b.cm.AddConnection(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	// don't return until the connection is closed
	c.Serve(w, r)

	// finally cleanup the connection
	b.cm.RemoveConnection(c.GetCLCID())
}

// HandleObserveProperty handles a property observe request for one or all properties
func (b *SseScBinding) HandleObserveProperty(w http.ResponseWriter, r *http.Request) {
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

	c := b.cm.GetConnectionByCLCID(rp.CLCID)
	if c != nil {
		c.ObserveProperty(rp.ThingID, rp.Name)
	} else {
		slog.Error("HandleObserveProperty: no matching connection found",
			"clientID", rp.ClientID, "clcid", rp.CLCID)
	}
}

// HandleObserveAllProperties adds a property subscription
func (b *SseScBinding) HandleObserveAllProperties(w http.ResponseWriter, r *http.Request) {
	b.HandleObserveProperty(w, r)
}

// HandleSubscribeEvent handles a subscription request for one or all events
func (b *SseScBinding) HandleSubscribeEvent(w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		slog.Warn("HandleSubscribe", "err", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Info("HandleSubscribe",
		slog.String("clientID", rp.ClientID),
		slog.String("clcid", rp.CLCID),
		slog.String("thingID", rp.ThingID),
		slog.String("name", rp.Name))

	c := b.cm.GetConnectionByCLCID(rp.CLCID)
	if c != nil {
		c.SubscribeEvent(rp.ThingID, rp.Name)
	} else {
		slog.Error("HandleSubscribeEvent: no matching connection found",
			"clientID", rp.ClientID, "connID", rp.CLCID)
	}
}

// HandleSubscribeAllEvents adds a subscription to all events
func (b *SseScBinding) HandleSubscribeAllEvents(w http.ResponseWriter, r *http.Request) {
	b.HandleSubscribeEvent(w, r)
}

// HandleUnobserveAllProperties handles removal of all property observe subscriptions
func (b *SseScBinding) HandleUnobserveAllProperties(w http.ResponseWriter, r *http.Request) {
	b.HandleUnobserveProperty(w, r)
}

// HandleUnobserveProperty handles removal of one property observe subscriptions
func (b *SseScBinding) HandleUnobserveProperty(w http.ResponseWriter, r *http.Request) {
	slog.Info("HandleUnobserveProperty")
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	c := b.cm.GetConnectionByCLCID(rp.CLCID)
	if err == nil {
		c.UnobserveProperty(rp.ThingID, rp.Name)
	}
}

// HandleUnsubscribeAllEvents removes the subscription
func (b *SseScBinding) HandleUnsubscribeAllEvents(w http.ResponseWriter, r *http.Request) {
	b.HandleUnsubscribeEvent(w, r)
}

// HandleUnsubscribeEvent handles removal of one or all event subscriptions
func (b *SseScBinding) HandleUnsubscribeEvent(w http.ResponseWriter, r *http.Request) {
	slog.Info("HandleUnsubscribeEvent")
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	c := b.cm.GetConnectionByCLCID(rp.CLCID)
	if err == nil {
		c.UnsubscribeEvent(rp.ThingID, rp.Name)
	}
}

// NewSseScBinding returns a new SSE-SC sub-protocol binding
func NewSseScBinding(cm *connections.ConnectionManager) *SseScBinding {
	b := &SseScBinding{
		cm: cm,
	}
	return b
}
