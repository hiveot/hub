package httpstransport

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/tlsserver"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/transports/httpstransport/sessions"
	"github.com/hiveot/hub/runtime/transports/httpstransport/subprotocols"
	"io"
	"log/slog"
	"net/http"
)

type HttpsOperation struct {
	op           string
	method       string
	subprotocol  string
	url          string
	handler      http.HandlerFunc
	isThingLevel bool
}

// HttpsTransport is the Hub transport binding for HTTPS
// This wraps the library's https server and add routes and middleware for use in the binding
type HttpsTransport struct {
	// port and path configuration
	config *HttpsTransportConfig

	// server cert/keys
	privKey    keys.IHiveKey
	serverCert *tls.Certificate
	caCert     *x509.Certificate

	// TLS server and router
	httpServer *tlsserver.TLSServer
	router     *chi.Mux

	// subprotocol bindings
	sse   *subprotocols.SseBinding
	ssesc *subprotocols.SseScBinding
	ws    *subprotocols.WsBinding

	// callback handler for incoming events,actions and rpc messages
	handler hubclient.MessageHandler

	// authenticator for logging in and validating session tokens
	authenticator api.IAuthenticator

	// Thing level operations
	operations []HttpsOperation

	// digitwin service
	dtwService *service.DigitwinService
}

// AddGetOp adds protocol binding operation with a URL and handler
//
// This is used to add Forms to the digitwin TDs
func (svc *HttpsTransport) AddGetOp(r chi.Router,
	op string, thingLevel bool, opURL string, handler http.HandlerFunc) {

	svc.operations = append(svc.operations, HttpsOperation{
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
func (svc *HttpsTransport) AddPostOp(r chi.Router,
	op string, isThingLevel bool, opURL string, handler http.HandlerFunc) {
	svc.operations = append(svc.operations, HttpsOperation{
		op:           op,
		method:       http.MethodPost,
		url:          opURL,
		handler:      handler,
		isThingLevel: isThingLevel,
	})
	r.Post(opURL, handler)
}

// setup the chain of routes used by the service and return the router
func (svc *HttpsTransport) createRoutes(router chi.Router) http.Handler {

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

		//- properties methods
		svc.AddGetOp(r, vocab.WotOpReadAllProperties, true,
			"/digitwin/properties/{thingID}", svc.HandleReadAllProperties)
		svc.AddGetOp(r, vocab.WoTOpReadProperty, true,
			"/digitwin/properties/{thingID}/{name}", svc.HandleReadProperty)

		svc.AddPostOp(r, vocab.WotOpObserveAllProperties, true,
			"/ssesc/digitwin/observe/{thingID}", svc.ssesc.HandleObserveAllProperties)
		svc.AddPostOp(r, vocab.WotOpObserveAllProperties, true,
			"/sse/digitwin/observe/{thingID}", svc.sse.HandleObserveAllProperties)
		svc.AddPostOp(r, vocab.WotOpObserveAllProperties, true,
			"/ws/digitwin/observe/{thingID}", svc.ws.HandleObserveAllProperties)

		svc.AddPostOp(r, vocab.WoTOpObserveProperty, true,
			"/ssesc/digitwin/observe/{thingID}/{name}", svc.ssesc.HandleObserveProperty)
		svc.AddPostOp(r, vocab.WoTOpObserveProperty, true,
			"/sse/digitwin/observe/{thingID}/{name}", svc.sse.HandleObserveProperty)
		svc.AddPostOp(r, vocab.WoTOpObserveProperty, true,
			"/ws/digitwin/observe/{thingID}/{name}", svc.ws.HandleObserveProperty)

		svc.AddPostOp(r, vocab.WoTOpUnobserveProperty, true,
			"/ssesc/digitwin/unobserve/{thingID}/{name}", svc.ssesc.HandleUnobserveProperty)
		svc.AddPostOp(r, vocab.WotOpUnobserveAllProperties, true,
			"/ssesc/digitwin/unobserve/{thingID}", svc.ssesc.HandleUnobserveAllProperties)

		svc.AddPostOp(r, vocab.WoTOpWriteProperty, true,
			"/digitwin/properties/{thingID}/{name}", svc.HandleWriteProperty)

		//- events methods
		svc.AddGetOp(r, "readallevents", true,
			"/digitwin/events/{thingID}", svc.HandleReadAllEvents)
		svc.AddGetOp(r, "readevent", true,
			"/digitwin/events/{thingID}/{eventID}", svc.HandleReadEvent)
		svc.AddPostOp(r, vocab.WotOpSubscribeEvent, true,
			"/digitwin/subscribe/{thingID}/{name}", svc.ssesc.HandleSubscribeEvent)
		svc.AddPostOp(r, vocab.WotOpSubscribeAllEvents, true,
			"/digitwin/subscribe/{thingID}", svc.ssesc.HandleSubscribeEvent)
		svc.AddPostOp(r, vocab.WotOpUnsubscribeAllEvents, true,
			"/digitwin/unsubscribe/{thingID}", svc.ssesc.HandleUnsubscribeEvent)
		svc.AddPostOp(r, vocab.WotOpUnsubscribeEvent, true,
			"/digitwin/unsubscribe/{thingID}/{name}", svc.ssesc.HandleUnsubscribeEvent)

		// actions methods
		svc.AddGetOp(r, vocab.WotOpQueryAllActions, true,
			"/digitwin/actions/{thingID}", svc.HandleQueryAllActions)
		svc.AddGetOp(r, vocab.WotOpQueryAction, true,
			"/digitwin/actions/{thingID}/{name}", svc.HandleQueryAction)
		svc.AddPostOp(r, vocab.WotOpInvokeAction, true,
			"/digitwin/actions/{thingID}/{name}", svc.HandleInvokeAction)

		// digitwin directory actions. Are these operations?
		svc.AddGetOp(r, vocab.HTOpReadThing, false,
			"/digitwin/things/{thingID}", svc.HandleReadThing)
		svc.AddGetOp(r, vocab.HTOpReadAllThings, false,
			"/digitwin/things", svc.HandleReadAllThings) // query params: offset,limit

		// handlers for other services. Operations to invoke actions.
		// TODO: these probably belong with the digitwin service TD

		// authn/authz service actions
		svc.AddPostOp(r, vocab.HTOpRefresh, false,
			"/authn/refresh", svc.HandleRefresh)
		svc.AddPostOp(r, vocab.HTOpLogout, false,
			"/authn/logout", svc.HandleLogout)

		// handlers for requests by agents
		// These are included in the digitwin TD forms
		svc.AddPostOp(r, vocab.HTOpUpdateThing, false,
			"/agent/tdd/{thingID}", svc.HandleUpdateThing)
		svc.AddPostOp(r, vocab.HTOpPublishEvent, false,
			"/agent/event/{thingID}/{name}", svc.HandleAgentPublishEvent)
		svc.AddPostOp(r, vocab.HTOpUpdateProperty, false,
			"/agent/property/{thingID}/{name}", svc.HandleAgentPublishEvent)

	})

	return router
}

// getRequestParams reads the client session, URL parameters and body payload from the request.
//
// The session context is set by the http middleware. If the session is not available then
// this returns an error. Note that the session middleware handler will block any request
// that requires a session.
//
// This protocol binding reads two variables, {thingID} and {name} in the path.
//
//	{thingID} is the agent or digital twin thing ID
//	{name} is the property, event or action name. '+' means 'all'
//	{messageType} is a legacy variable that is phased out
func (svc *HttpsTransport) getRequestParams(r *http.Request) (
	session *sessions.ClientSession, messageType string, thingID string, name string, body []byte, err error) {

	// get the required client session of this agent
	ctxSession := r.Context().Value(sessions.SessionContextID)
	if ctxSession == nil {
		// This is an internal error. The middleware session handler would have blocked
		// a request that required a session before getting here.
		err = fmt.Errorf("Missing session for request '%s' from '%s'",
			r.RequestURI, r.RemoteAddr)
		slog.Error(err.Error())
		return nil, messageType, "", "", nil, err
	}
	cs := ctxSession.(*sessions.ClientSession)

	// build a message from the URL and payload
	// URLParam names are defined by the path variables set in the router.
	thingID = chi.URLParam(r, "thingID")
	name = chi.URLParam(r, "name")
	messageType = chi.URLParam(r, "messageType")
	body, _ = io.ReadAll(r.Body)

	return cs, messageType, thingID, name, body, err
}

// receive a message from a client and ensure it has a message ID
// https transport apply a 'h-' messageID prefix for troubleshooting
//func (svc *HttpsTransport) handleMessage(msg *hubclient.ThingMessage) hubclient.DeliveryStatus {
//	if msg.MessageID == "" {
//		msg.MessageID = "h-" + shortid.MustGenerate()
//	}
//	return svc.handler(msg)
//}

// GetProtocolInfo returns info on the protocol supported by this binding
func (svc *HttpsTransport) GetProtocolInfo() api.ProtocolInfo {
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

// InvokeActionToAgent sends a thing action request to the agent
// This passes it to SSE/WS sub-protocol handlers of active sessions
func (svc *HttpsTransport) InvokeActionToAgent(dThingID string, name string, input any) {
	sm := sessions.GetSessionManager()
	sm.InvokeAction(dThingID, name, input)
}

// PublishEventToSubscribers sends an event message to subscribers of this event.
// This passes it to SSE/WS sub-protocol handlers of active sessions
func (svc *HttpsTransport) PublishEventToSubscribers(dThingID string, name string, value any) {
	sm := sessions.GetSessionManager()
	sm.PublishEvent(dThingID, name, value)
}

// PublishPropertyToObservers sends a property value message to observers of this property.
// This passes it to SSE/WS sub-protocol handlers of active sessions
func (svc *HttpsTransport) PublishPropertyToObservers(dThingID string, name string, value any) {
	sm := sessions.GetSessionManager()
	sm.PublishProperty(dThingID, name, value)
}

// Start the https server and listen for incoming connection requests
func (svc *HttpsTransport) Start(handler hubclient.MessageHandler) error {
	slog.Info("Starting HttpsTransport")
	svc.httpServer, svc.router = tlsserver.NewTLSServer(
		svc.config.Host, svc.config.Port, svc.serverCert, svc.caCert)

	svc.handler = handler
	svc.createRoutes(svc.router)
	err := svc.httpServer.Start()
	return err
}

// Stop the https server
func (svc *HttpsTransport) Stop() {
	slog.Info("Stopping HttpsTransport")

	// Shutdown remaining sessions to avoid hanging.
	// (closing the TLS server does not shut down active connections)
	sm := sessions.GetSessionManager()
	sm.CloseAll()
	svc.httpServer.Stop()
}

// writeError is a convenience function that logs and writes an error
// If the reply has an error then write a bad request with the error as payload
func (svc *HttpsTransport) writeError(w http.ResponseWriter, err error, code int) {
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
func (svc *HttpsTransport) writeReply(w http.ResponseWriter, data any) {
	if data != nil {
		// If no header is written then w.Write writes a StatusOK
		payload, _ := json.Marshal(data)
		_, _ = w.Write(payload)
	} else {
		// Only write header if no data is written
		w.WriteHeader(http.StatusOK)
	}
}

// NewHttpSSETransport creates a new instance of the HTTPS Server with JWT authentication
// and endpoints for bindings.
//
//	config
//	privKey
//	caCert
//	sessionAuth for creating and validating authentication tokens
//	dtwService that handles requests
func NewHttpSSETransport(config *HttpsTransportConfig,
	privKey keys.IHiveKey,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	authenticator api.IAuthenticator,
	dtwService *service.DigitwinService,
) *HttpsTransport {

	svc := HttpsTransport{
		authenticator: authenticator,
		config:        config,
		serverCert:    serverCert,
		caCert:        caCert,
		privKey:       privKey,
		dtwService:    dtwService,
		//httpServer:  httpServer,
		//router:      r,
	}
	return &svc
}
