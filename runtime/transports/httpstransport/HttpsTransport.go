package httpstransport

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/tlsserver"
	"github.com/hiveot/hub/runtime/transports/httpstransport/sessions"
	"github.com/hiveot/hub/runtime/transports/httpstransport/sse"
	"io"
	"log/slog"
	"net/http"
)

// HttpsTransport for transporting messages over HTTPS
// THis wraps the library's https server and add routes and middleware for use in the binding
type HttpsTransport struct {
	// port and path configuration
	config *HttpsTransportConfig

	// server key
	privKey keys.IHiveKey

	// TLS server and router
	httpServer *tlsserver.TLSServer
	router     *chi.Mux

	// callback handler for incoming events,actions and rpc messages
	handleMessage api.MessageHandler

	// sessionAuth for logging in and validating session tokens
	sessionAuth api.IAuthenticator

	// handlers for SSE server push connections
	sseHandler *sse.SSEHandler
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

	//-- add the routes and middleware
	//router.Use(middleware.Logger) // todo: proper logging strategy
	router.Use(middleware.Recoverer)
	//router.Use(csrfMiddleware)
	router.Use(middleware.Compress(5,
		"text/html", "text/css", "text/javascript", "image/svg+xml"))

	//--- public routes do not require a Hub connection
	router.Group(func(r chi.Router) {

		//r.Get("/static/*", staticFileServer.ServeHTTP)
		// build-in REST API for easy login to obtain a token
		r.Post(vocab.PostLoginPath, svc.HandlePostLogin)
	})

	//--- private routes that requires authentication
	router.Group(func(r chi.Router) {
		// client sessions authenticate the sender
		r.Use(sessions.AddSessionFromToken(svc.sessionAuth))

		// register the general purpose event and action message transport
		// these allows the binding to work as a transport for agents and consumers
		r.Post(vocab.PostActionPath, svc.HandlePostAction)
		r.Post(vocab.PostEventPath, svc.HandlePostEvent)

		// register rest api for built-in easy auth refresh and logout
		r.Post(vocab.PostRefreshPath, svc.HandlePostRefresh)
		r.Post(vocab.PostLogoutPath, svc.HandlePostLogout)

		// register rest api for built-in services
		//svc.authnHandler.RegisterMethods(r)
		//svc.dtDirectoryHandler.RegisterMethods(r)
		//svc.dtValuesHandler.RegisterMethods(r)
		//svc.dtHistoryHandler.RegisterMethods(r)
		r.Get(vocab.GetEventsPath, svc.HandleGetEvents)
		r.Get(vocab.GetThingsPath, svc.HandleGetThings)

		// sse return channels
		svc.sseHandler.RegisterMethods(r)
		//r.Get(vocab.ConnectWSPath, svc.handleWSConnect)
	})

	return router
}

// getRequestParams reads the client session, URL parameters and body payload from the request.
//
// The session context is set by the http middleware. If the session is not available then
// this returns an error. Note that the session middleware handler will block any request
// that requires a session.
func (svc *HttpsTransport) getRequestParams(r *http.Request) (
	session *sessions.ClientSession, thingID string, key string, body []byte, err error) {
	// get the required client session of this agent
	ctxSession := r.Context().Value(sessions.SessionContextID)
	if ctxSession == nil {
		// This is an internal error. The middleware session handler would have blocked
		// a request that required a session before getting here.
		err = fmt.Errorf("Missing session for request '%s' from '%s'",
			r.RequestURI, r.RemoteAddr)
		slog.Error(err.Error())
		return nil, "", "", nil, err
	}
	cs := ctxSession.(*sessions.ClientSession)

	// build a message from the URL and payload
	// URLParam names are defined by the path variables set in the router.
	thingID = chi.URLParam(r, "thingID")
	key = chi.URLParam(r, "key")
	body, _ = io.ReadAll(r.Body)

	return cs, thingID, key, body, err
}

// writeReply is a convenience function that writes a reply to a request.
// If the reply has an error then write a bad request with the error as payload
func (svc *HttpsTransport) writeReply(w http.ResponseWriter, payload []byte, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if payload != nil {
		_, _ = w.Write(payload)
	}
	w.WriteHeader(http.StatusOK)
}

// GetProtocolInfo returns info on the protocol supported by this binding
// TODO: this is a placeholder and will change to include all information needed for TD forms.
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

// HandlePostAction passes a posted action to the router
// this contains optional query parameter for messageID
func (svc *HttpsTransport) HandlePostAction(w http.ResponseWriter, r *http.Request) {
	cs, thingID, key, body, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	messageID := r.URL.Query().Get("messageID")
	if messageID == "" {
		messageID = uuid.NewString()
	}
	// this request can simply be turned into an action message.
	msg := things.NewThingMessage(
		vocab.MessageTypeAction, thingID, key, body, cs.GetClientID())
	msg.MessageID = messageID

	stat := svc.handleMessage(msg)
	reply, err := json.Marshal(&stat)
	svc.writeReply(w, reply, err)
}

// HandlePostEvent passes a posted event to the router
func (svc *HttpsTransport) HandlePostEvent(w http.ResponseWriter, r *http.Request) {
	cs, thingID, key, body, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	messageID := r.URL.Query().Get("messageID")
	if messageID == "" {
		messageID = uuid.NewString()
	}
	// this request can simply be turned into an event message.
	msg := things.NewThingMessage(
		vocab.MessageTypeEvent, thingID, key, body, cs.GetClientID())
	msg.MessageID = messageID

	stat := svc.handleMessage(msg)
	reply, err := json.Marshal(&stat)
	svc.writeReply(w, reply, err)
}

// SendEvent an event message to subscribers.
// This passes it to SSE handlers of active sessions
func (svc *HttpsTransport) SendEvent(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	sm := sessions.GetSessionManager()
	return sm.SendEvent(msg)
}

// SendToClient sends a message to a connected agent or consumer.
func (svc *HttpsTransport) SendToClient(
	clientID string, msg *things.ThingMessage) (stat api.DeliveryStatus, found bool) {

	stat.MessageID = msg.MessageID
	sm := sessions.GetSessionManager()
	cs, err := sm.GetSessionByClientID(clientID)
	if err == nil {
		payload, _ := json.Marshal(msg)
		count := cs.SendSSE(msg.MessageType, string(payload))
		if count == 0 {
			err = fmt.Errorf("client '%s' is not reachable", clientID)
			found = false
		} else {
			// completion status is send asynchroneously by the agent
			stat.Status = api.DeliveryDelivered
			found = true
		}
	}
	if err != nil {
		stat.Error = err.Error()
		stat.Status = api.DeliveryFailed
	}
	return stat, found
}

// Start the https server and listen for incoming connection requests
func (svc *HttpsTransport) Start(handler api.MessageHandler) error {
	slog.Info("Starting HttpsTransport")
	svc.handleMessage = handler
	svc.sseHandler = sse.NewSSEHandler(svc.sessionAuth)
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
	svc.sseHandler.Stop()

	svc.httpServer.Stop()

}

// NewHttpsBinding creates a new instance of the HTTPS Server with JWT authentication
// and endpoints for bindings.
//
//	config
//	privKey
//	caCert
//	sessionAuth for creating and validating authentication tokens
//	handler
func NewHttpsBinding(config *HttpsTransportConfig,
	privKey keys.IHiveKey,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	sessionAuth api.IAuthenticator,
) *HttpsTransport {

	httpServer, r := tlsserver.NewTLSServer(
		config.Host, uint(config.Port), serverCert, caCert)

	svc := HttpsTransport{
		sessionAuth: sessionAuth,
		config:      config,
		privKey:     privKey,
		httpServer:  httpServer,
		router:      r,
	}
	return &svc
}
