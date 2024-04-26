package httpsbinding

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/protocols/httpsbinding/rest"
	"github.com/hiveot/hub/runtime/protocols/httpsbinding/sessions"
	"github.com/hiveot/hub/runtime/protocols/httpsbinding/sse"
	"github.com/hiveot/hub/runtime/router"
	"github.com/hiveot/hub/runtime/tlsserver"
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
	sessionAuth api.IAuthenticator

	// handlers for the REST APIs
	dtDirectoryHandler *rest.DigiTwinDirectory
	dtValuesHandler    *rest.DigiTwinValues
	dtHistoryHandler   *rest.DigiTwinHistory
	authnRestHandler   *rest.AuthnRest

	// handlers for SSE server push connections
	sseHandler *sse.SSEHandler
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

		//r.Get("/static/*", staticFileServer.ServeHTTP)
		r.Post(vocab.PostLoginPath, svc.authnRestHandler.HandlePostLogin)
	})

	//--- private routes that requires authentication
	router.Group(func(r chi.Router) {
		// client sessions authenticate the sender
		r.Use(sessions.AddSessionFromToken(svc.sessionAuth))

		// register rest api for built-in services
		svc.authnRestHandler.RegisterMethods(r)
		svc.dtDirectoryHandler.RegisterMethods(r)
		svc.dtValuesHandler.RegisterMethods(r)
		svc.dtHistoryHandler.RegisterMethods(r)

		// sse has its own validation instead of using session context (which reconnects or redirects to /login)
		svc.sseHandler.RegisterMethods(r)
		//r.Get(vocab.ConnectSSEPath, svc.handleSseConnect)
		//r.Get(vocab.ConnectWSPath, svc.handleWSConnect)
	})

	return router
}

// Start the https server and listen for incoming connection requests
func (svc *HttpsBinding) Start(handler router.MessageHandler) error {
	slog.Info("Starting HttpsBinding")
	svc.handleMessage = handler

	svc.dtDirectoryHandler = rest.NewDigiTwinDirectory(svc.handleMessage)
	svc.dtHistoryHandler = rest.NewDigiTwinHistory(svc.handleMessage)
	svc.dtValuesHandler = rest.NewDigiTwinValues(svc.handleMessage)
	svc.authnRestHandler = rest.NewAuthnRest(svc.handleMessage, svc.sessionAuth)
	svc.sseHandler = sse.NewSSEHandler(svc.handleMessage, svc.sessionAuth)

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
	sessionAuth api.IAuthenticator,
) *HttpsBinding {

	httpServer, r := tlsserver.NewTLSServer(
		config.Host, uint(config.Port), serverCert, caCert)

	svc := HttpsBinding{
		sessionAuth: sessionAuth,
		config:      config,
		privKey:     privKey,
		httpServer:  httpServer,
		router:      r,
	}
	return &svc
}
