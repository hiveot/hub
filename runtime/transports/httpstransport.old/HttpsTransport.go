package httpstransport_old

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/tlsserver"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/transports/httpstransport/sessions"
	"github.com/hiveot/hub/runtime/transports/httpstransport/sseserver"
	"github.com/teris-io/shortid"
	"io"
	"log/slog"
	"net/http"
)

// HttpsTransport for transporting messages over HTTPS
// THis wraps the library's https server and add routes and middleware for use in the binding
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

	// callback handler for incoming events,actions and rpc messages
	handler hubclient.MessageHandler

	// authenticator for logging in and validating session tokens
	authenticator api.IAuthenticator

	// SSE server push connections
	sseServer *sseserver.SSEServer
}

// setup the chain of routes used by the service and return the router
func (svc *HttpsTransport) createRoutes(router *chi.Mux) http.Handler {

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
		r.Post(PostLoginPath, svc.HandleLogin)
	})
	//--- sse
	router.Group(func(r chi.Router) {
		// compression doesnt work with go-sse server?
		//r.Use(middleware.Compress(5,
		//	"text/html", "text/css", "text/javascript", "image/svg+xml"))

		// client sessions authenticate the sender
		r.Use(sessions.AddSessionFromToken(svc.authenticator))
		r.HandleFunc(ConnectSSEPath, svc.sseServer.ServeHTTP)
	})

	//--- private routes that requires authentication (as published in the TD)
	router.Group(func(r chi.Router) {
		r.Use(middleware.Compress(5,
			"text/html", "text/css", "text/javascript", "image/svg+xml"))

		// client sessions authenticate the sender
		r.Use(sessions.AddSessionFromToken(svc.authenticator))

		//- properties methods
		r.Get(GetReadAllPropertiesPath, svc.HandleReadAllProperties)
		r.Get(GetReadPropertyPath, svc.HandleReadProperty)
		r.Post(PostWritePropertyPath, svc.HandleWriteProperty)
		r.Post(PostObserveAllPropertiesPath, svc.HandleObserve)
		r.Post(PostObservePropertyPath, svc.HandleObserve)

		//- events methods
		r.Get(GetReadAllEventsPath, svc.HandleReadAllEvents)
		r.Post(PostSubscribeEventPath, svc.HandleSubscribe)
		r.Post(PostSubscribeAllEventsPath, svc.HandleSubscribe)
		r.Post(PostUnsubscribeEventPath, svc.HandleUnsubscribe)
		r.Post(PostUnsubscribeAllEventsPath, svc.HandleUnsubscribe)
		r.Post(PostAgentPublishEventPath, svc.HandleAgentPublishEvent)

		// actions methods
		r.Get(GetReadActionPath, svc.HandleReadAction)
		r.Get(GetReadAllActionsPath, svc.HandleReadAction)
		r.Post(PostInvokeActionPath, svc.HandleInvokeAction)

		//- directory methods
		r.Get(GetReadThingPath, svc.HandleReadThing)
		r.Get(GetReadAllThingsPath, svc.HandleReadAllThings) // query params: offset,limit

		// authn service
		r.Post(PostRefreshPath, svc.HandleRefresh)
		r.Post(PostLogoutPath, svc.HandleLogout)
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
func (svc *HttpsTransport) handleMessage(msg *hubclient.ThingMessage) hubclient.DeliveryStatus {
	if msg.MessageID == "" {
		msg.MessageID = "h-" + shortid.MustGenerate()
	}
	return svc.handler(msg)
}

// writeStatReply is a convenience function that writes the reply in a delivery status message
// If stat has an error then write a bad request with the error as payload
func (svc *HttpsTransport) writeStatReply(w http.ResponseWriter, stat hubclient.DeliveryStatus) {
	if stat.Error != "" {
		http.Error(w, stat.Error, http.StatusBadRequest)
		return
	}
	if stat.Reply != nil {
		// this transport uses json encoding
		payload, _ := json.Marshal(stat.Reply)
		// If no header is written then w.Write writes a StatusOK
		_, _ = w.Write(payload)
	} else {
		// Only write header if no data is written
		w.WriteHeader(http.StatusOK)
	}
}

// writeReply is a convenience function that writes a reply to a request.
// If the reply has an error then write a bad request with the error as payload
func (svc *HttpsTransport) writeReply(w http.ResponseWriter, payload []byte, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if payload != nil {
		// If no header is written then w.Write writes a StatusOK
		_, _ = w.Write(payload)
	} else {
		// Only write header if no data is written
		w.WriteHeader(http.StatusOK)
	}
}

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

// SendEvent an event message to subscribers.
// This passes it to SSE handlers of active sessions
func (svc *HttpsTransport) SendEvent(msg *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
	sm := sessions.GetSessionManager()
	return sm.SendEvent(msg)
}

// SendToClient sends a message to a connected agent or consumer.
// This sends the message to all sessions of this client. This might not be what is
// intended ...
func (svc *HttpsTransport) SendToClient(
	clientID string, msg *hubclient.ThingMessage) (stat hubclient.DeliveryStatus, found bool) {

	stat.MessageID = msg.MessageID
	sm := sessions.GetSessionManager()
	csList, err := sm.GetSessionsByClientID(clientID)
	count := 0
	if err == nil {
		for _, cs := range csList {
			count += cs.SendSSE(msg.MessageID, msg.MessageType, msg)
		}
		if count == 0 {
			err = fmt.Errorf("client '%s' is not reachable", clientID)
			found = false
		} else {
			// completion status is sent asynchroneously by the agent
			stat.Progress = hubclient.DeliveredToAgent
			found = true
		}
	}

	if err != nil {
		stat.Failed(msg, err)
	}
	return stat, found
}

// Start the https server and listen for incoming connection requests
func (svc *HttpsTransport) Start(handler hubclient.MessageHandler) error {
	slog.Info("Starting HttpsTransport")
	svc.httpServer, svc.router = tlsserver.NewTLSServer(
		svc.config.Host, svc.config.Port, svc.serverCert, svc.caCert)

	svc.handler = handler
	svc.sseServer = sseserver.NewSSEServer()
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
	svc.sseServer.Stop()
}

// NewHttpSSETransport creates a new instance of the HTTPS Server with JWT authentication
// and endpoints for bindings.
//
//	config
//	privKey
//	caCert
//	sessionAuth for creating and validating authentication tokens
//	handler
func NewHttpSSETransport(config *HttpsTransportConfig,
	privKey keys.IHiveKey,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	authenticator api.IAuthenticator,
) *HttpsTransport {

	svc := HttpsTransport{
		authenticator: authenticator,
		config:        config,
		serverCert:    serverCert,
		caCert:        caCert,
		privKey:       privKey,
		//httpServer:  httpServer,
		//router:      r,
	}
	return &svc
}