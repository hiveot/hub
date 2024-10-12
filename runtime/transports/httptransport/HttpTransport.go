package httptransport

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/tlsserver"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/hubrouter"
	"github.com/hiveot/hub/runtime/transports/httptransport/sessions"
	"github.com/hiveot/hub/runtime/transports/httptransport/subprotocols"
	"log/slog"
	"net/http"
)

type HttpOperation struct {
	op           string
	method       string
	subprotocol  string
	url          string
	handler      http.HandlerFunc
	isThingLevel bool
}

// HttpTransport is the Hub transport binding for HTTPS
// This wraps the library's https server and add routes and middleware for use in the binding
type HttpTransport struct {
	// port and path configuration
	config *HttpTransportConfig

	// TLS server and router
	httpServer *tlsserver.TLSServer
	router     *chi.Mux
	//sm         *sessions.SessionManager

	// subprotocol bindings
	sse   *subprotocols.SseBinding
	ssesc *subprotocols.SseScBinding
	ws    *subprotocols.WsBinding

	// authenticator for logging in and validating session tokens
	authenticator api.IAuthenticator

	// Thing level operations
	operations []HttpOperation

	// routing of action, event and property requests
	hubRouter hubrouter.IHubRouter

	// reading of digital twin info
	dtwService *service.DigitwinService
}

// AddGetOp adds protocol binding operation with a URL and handler
//
// This is used to add Forms to the digitwin TDs
func (svc *HttpTransport) AddGetOp(r chi.Router,
	op string, thingLevel bool, opURL string, handler http.HandlerFunc) {

	svc.operations = append(svc.operations, HttpOperation{
		op:           op,
		method:       http.MethodGet,
		url:          opURL,
		handler:      handler,
		isThingLevel: thingLevel,
	})
	r.Get(opURL, handler)
}

// AddPostOp adds protocol binding operation with a URL and handler
//
// This is used to add Forms to the digitwin TDs
func (svc *HttpTransport) AddPostOp(r chi.Router,
	op string, isThingLevel bool, opURL string, handler http.HandlerFunc) {
	svc.operations = append(svc.operations, HttpOperation{
		op:           op,
		method:       http.MethodPost,
		url:          opURL,
		handler:      handler,
		isThingLevel: isThingLevel,
	})
	r.Post(opURL, handler)
}

// setup the chain of routes used by the service and return the router
func (svc *HttpTransport) createRoutes(router chi.Router) http.Handler {

	// TODO: is there a use for a static file server?
	//var staticFileServer http.Handler
	//if rootPath == "" {
	//	staticFileServer = http.FileServer(
	//		&StaticFSWrapper{
	//			FileSystem:   http.FS(src.EmbeddedStatic),
	//			FixedModTime: time.Now(),
	//		})
	//} else {
	//	// during development when run from the 'hub' project directory
	//	staticFileServer = http.FileServer(http.Dir(rootPath))
	//}

	// TODO: add csrf support in posts
	//csrfMiddleware := csrf.Protect(
	//	[]byte("32-byte-long-auth-key"),
	//	csrf.SameSite(csrf.SameSiteStrictMode))

	//-- add the middleware before routes
	router.Use(middleware.Recoverer)
	//router.Use(middleware.Logger) // todo: proper logging strategy
	//router.Use(middleware.Recoverer)
	//router.Use(csrfMiddleware)
	//router.Use(middleware.Compress(5,
	//	"text/html", "text/css", "text/javascript", "image/svg+xml"))

	//--- public routes do not require an authenticated session
	router.Group(func(r chi.Router) {
		r.Use(middleware.Compress(5,
			"text/html", "text/css", "text/javascript", "image/svg+xml"))

		//r.Get("/static/*", staticFileServer.ServeHTTP)
		// build-in REST API for easy login to obtain a token
		//r.Post(PostLoginPath, svc.HandleLogin)
		svc.AddPostOp(r, vocab.HTOpLogin, false,
			"/authn/login", svc.HandleLogin)
	})

	//--- private routes that requires authentication (as published in the TD)
	router.Group(func(r chi.Router) {
		r.Use(middleware.Compress(5,
			"text/html", "text/css", "text/javascript", "image/svg+xml"))

		// client sessions authenticate the sender
		r.Use(sessions.AddSessionFromToken(svc.authenticator))

		//- direct methods for digital twins
		svc.AddGetOp(r, vocab.WotOpReadAllProperties, true,
			"/digitwin/properties/{thingID}", svc.HandleReadAllProperties)
		svc.AddGetOp(r, vocab.WoTOpReadProperty, true,
			"/digitwin/properties/{thingID}/{name}", svc.HandleReadProperty)
		svc.AddPostOp(r, vocab.WoTOpWriteProperty, true,
			"/digitwin/properties/{thingID}/{name}", svc.HandleWriteProperty)

		svc.AddGetOp(r, "readallevents", true,
			"/digitwin/events/{thingID}", svc.HandleReadAllEvents)
		svc.AddGetOp(r, "readevent", true,
			"/digitwin/events/{thingID}/{eventID}", svc.HandleReadEvent)

		svc.AddGetOp(r, vocab.WotOpQueryAllActions, true,
			"/digitwin/actions/{thingID}", svc.HandleQueryAllActions)
		svc.AddGetOp(r, vocab.WotOpQueryAction, true,
			"/digitwin/actions/{thingID}/{name}", svc.HandleQueryAction)
		svc.AddPostOp(r, vocab.WotOpInvokeAction, true,
			"/digitwin/actions/{thingID}/{name}", svc.HandleActionRequest)

		// sse-sc subprotocol routes
		svc.AddGetOp(r, "connect", true,
			"/ssesc", svc.ssesc.HandleConnect)
		svc.AddPostOp(r, vocab.WotOpObserveAllProperties, true,
			"/ssesc/digitwin/observe/{thingID}", svc.ssesc.HandleObserveAllProperties)
		svc.AddPostOp(r, vocab.WotOpSubscribeAllEvents, true,
			"/ssesc/digitwin/subscribe/{thingID}", svc.ssesc.HandleSubscribeAllEvents)
		svc.AddPostOp(r, vocab.WotOpSubscribeEvent, true,
			"/ssesc/digitwin/subscribe/{thingID}/{name}", svc.ssesc.HandleSubscribeEvent)
		svc.AddPostOp(r, vocab.WoTOpObserveProperty, true,
			"/ssesc/digitwin/observe/{thingID}/{name}", svc.ssesc.HandleObserveProperty)
		svc.AddPostOp(r, vocab.WotOpUnobserveAllProperties, true,
			"/ssesc/digitwin/observe/{thingID}", svc.ssesc.HandleUnobserveAllProperties)
		svc.AddPostOp(r, vocab.WoTOpUnobserveProperty, true,
			"/ssesc/digitwin/unobserve/{thingID}/{name}", svc.ssesc.HandleUnobserveProperty)
		svc.AddPostOp(r, vocab.WotOpUnsubscribeAllEvents, true,
			"/ssesc/digitwin/unsubscribe/{thingID}", svc.ssesc.HandleUnsubscribeAllEvents)
		svc.AddPostOp(r, vocab.WotOpUnsubscribeEvent, true,
			"/ssesc/digitwin/unsubscribe/{thingID}/{name}", svc.ssesc.HandleUnsubscribeEvent)

		// ws subprotocol routes
		svc.AddPostOp(r, vocab.WotOpObserveAllProperties, true,
			"/ws/digitwin/observe/{thingID}", svc.ws.HandleObserveAllProperties)
		svc.AddPostOp(r, vocab.WoTOpObserveProperty, true,
			"/ws/digitwin/observe/{thingID}/{name}", svc.ws.HandleObserveProperty)

		// sse subprotocol routes
		svc.AddPostOp(r, vocab.WotOpSubscribeAllEvents, true,
			"/sse/digitwin/subscribe/{thingID}", svc.sse.HandleSubscribeAllEvents)
		svc.AddPostOp(r, vocab.WotOpObserveAllProperties, true,
			"/sse/digitwin/observe/{thingID}", svc.sse.HandleObserveAllProperties)

		// digitwin directory actions. Are these operations or actions?
		svc.AddGetOp(r, vocab.HTOpReadThing, false,
			"/digitwin/directory/{thingID}", svc.HandleReadThing)
		svc.AddGetOp(r, vocab.HTOpReadAllThings, false,
			"/digitwin/directory", svc.HandleReadAllThings) // query params: offset,limit

		// handlers for other services. Operations to invoke actions.
		// TODO: these probably belong with the digitwin service TD

		// authn/authz service actions
		svc.AddPostOp(r, vocab.HTOpRefresh, false,
			"/authn/refresh", svc.HandleRefresh)
		svc.AddPostOp(r, vocab.HTOpLogout, false,
			"/authn/logout", svc.HandleLogout)

		// handlers for requests by agents
		// TODO: These should be included in the digitwin TD forms
		svc.AddPostOp(r, vocab.HTOpUpdateThing, false,
			"/agent/tdd/{thingID}", svc.HandleUpdateThing)
		svc.AddPostOp(r, vocab.HTOpPublishEvent, false,
			"/agent/event/{thingID}/{name}", svc.HandlePublishEvent)
		svc.AddPostOp(r, vocab.HTOpUpdateProperty, false,
			"/agent/property/{thingID}/{name}", svc.HandleUpdateProperty)
		svc.AddPostOp(r, "updateMultipleProperties", false,
			"/agent/properties/{thingID}", svc.HandleUpdateProperty)
		svc.AddPostOp(r, vocab.HTOpDelivery, false,
			"/agent/progress", svc.HandleProgressUpdate)

	})

	return router
}

// GetProtocolInfo returns info on the protocol supported by this binding
func (svc *HttpTransport) GetProtocolInfo() api.ProtocolInfo {
	hostName := svc.config.Host
	if hostName == "" {
		hostName = "localhost"
	}
	baseURL := fmt.Sprintf("https://%s:%d", hostName, svc.config.Port)
	inf := api.ProtocolInfo{
		BaseURL:   baseURL,
		Schema:    "https",
		Transport: "https",
	}
	return inf
}

// InvokeAction sends a thing action request to the agent
// This passes it to SSE/WS sub-protocol handlers of active sessions
func (svc *HttpTransport) InvokeAction(
	agentID string, thingID string, name string, input any, messageID string, senderID string) (
	status string, output any, err error) {

	if svc.ws != nil {
		status, output, err = svc.ws.InvokeAction(agentID, thingID, name, input, messageID, senderID)
	}
	if svc.ssesc != nil {
		status, output, err = svc.ssesc.InvokeAction(agentID, thingID, name, input, messageID, senderID)
	}
	if err != nil && svc.sse != nil {
		status, output, err = svc.sse.InvokeAction(agentID, thingID, name, input, messageID, senderID)
	}
	if err != nil {
		status = vocab.ProgressStatusFailed
		err = fmt.Errorf("InvokeAction: No sub-protocol bindings")
	}
	return status, output, err
}

// PublishActionProgress sends the action update to the client
func (svc *HttpTransport) PublishProgressUpdate(
	clientID string, stat hubclient.DeliveryStatus, agentID string) (found bool, err error) {

	found = false
	if svc.ws != nil {
		found, err = svc.ws.SendActionResult(clientID, stat, agentID)
	}
	if !found && svc.ssesc != nil {
		found, err = svc.ssesc.SendActionResult(clientID, stat, agentID)
	}
	if !found && svc.sse != nil {
		found, err = svc.sse.SendActionResult(clientID, stat, agentID)
	}
	return found, err
}

// PublishEvent sends an event message to subscribers of this event.
// This passes it to SSE/WS sub-protocol handlers of active sessions
func (svc *HttpTransport) PublishEvent(
	dThingID string, name string, value any, messageID string, agentID string) {
	if svc.ws != nil {
		svc.ws.PublishEvent(dThingID, name, value, messageID, agentID)
	}
	if svc.ssesc != nil {
		svc.ssesc.PublishEvent(dThingID, name, value, messageID, agentID)
	}
	if svc.sse != nil {
		svc.sse.PublishEvent(dThingID, name, value, messageID, agentID)
	}
}

// PublishProperty sends a property value update to observers of this property.
// This passes it to SSE/WS sub-protocol handlers of active sessions
func (svc *HttpTransport) PublishProperty(
	dThingID string, name string, value any, messageID string, agentID string) {

	if svc.ws != nil {
		svc.ws.PublishProperty(dThingID, name, value, messageID, agentID)
	}
	if svc.ssesc != nil {
		svc.ssesc.PublishProperty(dThingID, name, value, messageID, agentID)
	}
	if svc.sse != nil {
		svc.sse.PublishProperty(dThingID, name, value, messageID, agentID)
	}
}

// Stop the https server
func (svc *HttpTransport) Stop() {
	slog.Info("Stopping HttpTransport")

	// Shutdown remaining sessions to avoid hanging.
	// (closing the TLS server does not shut down active connections)
	//sm := sessions.GetSessionManager()
	//sm.CloseAll()
	svc.httpServer.Stop()
}

// writeError is a convenience function that logs and writes an error
// If the reply has an error then write a bad request with the error as payload
func (svc *HttpTransport) writeError(w http.ResponseWriter, err error, code int) {
	if code == 0 {
		code = http.StatusBadRequest
	}
	if err != nil {
		slog.Warn("Request error: ", "err", err.Error())
		http.Error(w, err.Error(), code)
	} else {
		w.WriteHeader(code)
	}
}

// writeReply is a convenience function that serializes the data and writes it as a response.
func (svc *HttpTransport) writeReply(w http.ResponseWriter, data any) {
	if data != nil {
		// If no header is written then w.Write writes a StatusOK
		payload, _ := json.Marshal(data)
		_, _ = w.Write(payload)
	} else {
		// Only write header if no data is written
		w.WriteHeader(http.StatusOK)
	}
}

// WriteProperty sends a request to write a property to the agent with the given ID
func (svc *HttpTransport) WriteProperty(
	agentID string, thingID string, name string, input any, messageID string, senderID string) (
	found bool, status string, err error) {

	found = false
	if svc.ws != nil {
		found, status, err = svc.ws.WriteProperty(agentID, thingID, name, input, messageID, senderID)
	}
	if !found && svc.ssesc != nil {
		found, status, err = svc.ssesc.WriteProperty(agentID, thingID, name, input, messageID, senderID)
	}
	if !found && svc.sse != nil {
		found, status, err = svc.sse.WriteProperty(agentID, thingID, name, input, messageID, senderID)
	}
	return found, status, err
}

// StartHttpTransport creates and starts a new instance of the HTTPS Server
// with JWT authentication and SSE/SSE-SC/WS sub-protocol bindings.
//
// Call stop to end the transport server.
//
//	config
//	privKey
//	caCert
//	sessionAuth for creating and validating authentication tokens
//	dtwService that handles digital thing requests
func StartHttpTransport(config *HttpTransportConfig,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	authenticator api.IAuthenticator,
	hubRouter hubrouter.IHubRouter,
	dtwService *service.DigitwinService,
) (*HttpTransport, error) {
	// FIXME: do not use a global. the sessionmanager belongs to this transport binding.
	//
	sm := sessions.GetSessionManager()

	httpServer, router := tlsserver.NewTLSServer(
		config.Host, config.Port, serverCert, caCert)

	svc := HttpTransport{
		authenticator: authenticator,
		config:        config,
		// subprotocol bindings need session info
		ws:         subprotocols.NewWsBinding(sm),
		sse:        subprotocols.NewSseBinding(sm),
		ssesc:      subprotocols.NewSseScBinding(sm),
		httpServer: httpServer,
		router:     router,
		hubRouter:  hubRouter,
		dtwService: dtwService,
	}

	svc.createRoutes(svc.router)
	err := svc.httpServer.Start()
	return &svc, err
}
