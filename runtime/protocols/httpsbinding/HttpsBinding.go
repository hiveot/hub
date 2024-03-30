package httpsbinding

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/tlsserver"
	"io"
	"log/slog"
	"net/http"
)

// HttpsBinding for handling messages over HTTPS
// THis wraps the library's https server and add routes and middleware for use in the binding
type HttpsBinding struct {

	// server key
	privKey keys.IHiveKey

	// TLS server and router
	httpServer *tlsserver.TLSServer
	router     *chi.Mux

	// callback handler for incoming events,actions and rpc messages
	handleMessage func(tv *things.ThingValue) ([]byte, error)
}

// setup the chain of routes used by the service and return the router
func (svc *HttpsBinding) createRoutes(router *chi.Mux) http.Handler {

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
	router.Use(middleware.Recoverer)
	router.Use(middleware.Logger)
	//router.Use(csrfMiddleware)
	router.Use(middleware.Compress(5,
		"text/html", "text/css", "text/javascript", "image/svg+xml"))

	//--- public routes do not require a Hub connection
	router.Group(func(r chi.Router) {

		// full page routes
		//r.Get("/static/*", staticFileServer.ServeHTTP)
		//r.Post("/login", svc.handleLogin)
		//r.Post("/logout", svc.handleLogout)

	})

	//--- private routes that requires authentication
	router.Group(func(r chi.Router) {
		// client sessions
		r.Use(AddSessionFromToken(svc.privKey.PublicKey()))

		// agents
		r.Post("/event/{agentID}/{thingID}/{key}", svc.handlePostEvent)
		//r.Get("/action/{agentID}/{thingID}", svc.handleGetQueuedActions)

		// consumers
		//r.Get("/event/{agentID}/{thingID}/{key}", svc.handleGetEvents)
		//r.Post("/action/{agentID}/{thingID}/{key}", svc.handlePostAction)

		// both agents and consumers
		//r.Get("/rpc/{serviceID}/{interfaceID}/{method}", svc.handleInvokeRPC)
		// sse has its own validation instead of using session context (which reconnects or redirects to /login)
		//r.Get("/sse", SseHandler)
	})

	return router
}

// onPostEvent handles a posted event by agent
func (svc *HttpsBinding) handlePostEvent(w http.ResponseWriter, r *http.Request) {
	// get the client session of this agent
	ctxSession := r.Context().Value(SessionContextID)
	if ctxSession == nil {
		slog.Error("handlePostEvent. Missing session.")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	cs := ctxSession.(*ClientSession)

	agentID := chi.URLParam(r, "agentID")
	thingID := chi.URLParam(r, "thingID")
	key := chi.URLParam(r, "key")
	data, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Warn("Error reading event body", "err", err)
	}
	// pass event to the protocol handler
	tv := things.NewThingValue(transports.MessageTypeEvent, agentID, thingID, key, data, cs.clientID)
	reply, err := svc.handleMessage(tv)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// return the reply
	if reply != nil {
		_, _ = w.Write(reply)
	}
	w.WriteHeader(http.StatusOK)
	return
}

// Start the https server and listen for incoming connection requests
func (svc *HttpsBinding) Start() error {
	slog.Info("HttpsBinding Start")
	svc.createRoutes(svc.router)
	err := svc.httpServer.Start()
	return err
}

// Stop the https server
func (svc *HttpsBinding) Stop() {
	svc.httpServer.Stop()
}

// NewHttpsBinding creates a new instance of the HTTPS Server with JWT authentication
// and endpoints for bindings.
func NewHttpsBinding(
	port uint,
	privKey keys.IHiveKey,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	handler func(tv *things.ThingValue) ([]byte, error),
) *HttpsBinding {
	httpServer, router := tlsserver.NewTLSServer("", port, serverCert, caCert)
	svc := HttpsBinding{
		privKey:       privKey,
		httpServer:    httpServer,
		handleMessage: handler,
		router:        router,
	}
	return &svc
}
