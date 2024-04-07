package httpsbinding

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/authn"
	"github.com/hiveot/hub/runtime/tlsserver"
	"io"
	"log/slog"
	"net/http"
)

// HttpsBinding for handling messages over HTTPS
// THis wraps the library's https server and add routes and middleware for use in the binding
type HttpsBinding struct {
	// port and path configuration
	config *HttpsBindingConfig

	// server key
	privKey keys.IHiveKey

	// TLS server and router
	httpServer *tlsserver.TLSServer
	router     *chi.Mux

	// callback handler for incoming events,actions and rpc messages
	handleMessage func(tv *things.ThingMessage) ([]byte, error)

	// sessionAuth for logging in and validating session tokens
	sessionAuth authn.IAuthenticator
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
		r.Post(svc.config.PostLoginPath, svc.handleLogin)
		//r.Post("/logout", svc.handleLogout)

	})

	//--- private routes that requires authentication
	router.Group(func(r chi.Router) {
		// client sessions
		r.Use(svc.AddSessionFromToken())

		// handlers for receiving requests by type
		r.Post(svc.config.PostActionPath, svc.handlePostAction)
		r.Post(svc.config.PostEventPath, svc.handlePostEvent)
		r.Post(svc.config.PostRPCPath, svc.handlePostRPC)

		//r.Get("/action/{agentID}/{thingID}", svc.handleGetQueuedActions)

		// consumers
		//r.Get("/event/{agentID}/{thingID}/{key}", svc.handleGetEvents)
		//r.Post("/action/{agentID}/{thingID}/{key}", svc.handlePostAction)

		// both agents and consumers
		//r.Get("/rpc/{serviceID}/{interfaceID}/{method}", svc.handleInvokeRPC)
		// sse has its own validation instead of using session context (which reconnects or redirects to /login)
		r.Get("/sse", SseHandler)
	})

	return router
}

// Handle login and return an auth token with a new session id for the client
func (svc *HttpsBinding) handleLogin(w http.ResponseWriter, req *http.Request) {
	// credentials are in a json payload
	authMsg := make(map[string]string)
	data, _ := io.ReadAll(req.Body)
	err := json.Unmarshal(data, &authMsg)
	if err != nil {
		slog.Warn("handleLogin failed getting credentials", "err", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	loginID := authMsg["login"]
	password := authMsg["password"]
	authToken, err := svc.sessionAuth.Login(loginID, password, "")
	if err != nil {
		// missing bearer token
		slog.Warn("handleLogin bad login", "clientID", loginID)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, _ = w.Write([]byte(authToken))
	w.WriteHeader(http.StatusOK)
}

// Start the https server and listen for incoming connection requests
func (svc *HttpsBinding) Start() error {
	slog.Info("Starting HttpsBinding")
	svc.createRoutes(svc.router)
	err := svc.httpServer.Start()
	return err
}

// Stop the https server
func (svc *HttpsBinding) Stop() {
	slog.Info("Stopping HttpsBinding")
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
func NewHttpsBinding(config *HttpsBindingConfig,
	privKey keys.IHiveKey,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	sessionAuth authn.IAuthenticator,
	handler func(tv *things.ThingMessage) ([]byte, error),
) *HttpsBinding {

	httpServer, router := tlsserver.NewTLSServer(
		config.Host, uint(config.Port), serverCert, caCert)

	svc := HttpsBinding{
		sessionAuth:   sessionAuth,
		config:        config,
		privKey:       privKey,
		httpServer:    httpServer,
		handleMessage: handler,
		router:        router,
	}
	return &svc
}
