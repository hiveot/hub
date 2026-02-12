package httpbasic

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/hiveot/hub/lib/messaging"
)

// HTTP-basic profile constants
const (
	// ConnectionIDHeader is intended for linking return channels to requests.
	// intended for separated return channel like sse.
	ConnectionIDHeader = "cid"
	// CorrelationIDHeader is the header to be able to link requests to out of band responses
	// tentative as it isn't part of the wot spec
	CorrelationIDHeader = "correlationID"

	// HttpPostLoginPath is the fixed authentication endpoint of the hub
	HttpPostLoginPath   = "/authn/login"
	HttpPostLogoutPath  = "/authn/logout"
	HttpPostRefreshPath = "/authn/refresh"
	HttpGetPingPath     = "/ping"

	// The generic path for thing operations over http using URI variables
	HttpBaseFormOp                   = "/things"
	HttpBasicAffordanceOperationPath = "/things/{operation}/{thingID}/{name}"
	HttpBasicThingOperationPath      = "/things/{operation}/{thingID}"
	HttpBasicOperationURIVar         = "operation"
	HttpBasicThingIDURIVar           = "thingID"
	HttpBasicNameURIVar              = "name"

	// static file server routes
	DefaultHttpStaticBase      = "/static"
	DefaultHttpStaticDirectory = "stores/httpstatic" // relative to home
)

// HttpWoTBasicServer provides the http-basic protocol binding using the provided http server.
// This is the simplest protocol binding supported by hiveot.
// Features:
// - security bootstrapping as per https://w3c.github.io/wot-discovery/#exploration-secboot
//   - login to obtain a bearer token:  POST {base}/authn/login
//   - refresh bearer token:            POST {base}/authn/refresh
//
// - post/get thing operations:       POST {base}/things/{op}/{thingID}
// - post/get affordance operations:  POST {base}/things/{op}/{thingID}/{name}
//
// This uses the provided httpserver instance.
type HttpWoTBasicServer struct {
	// authenticator for logging in and validating session tokens
	authenticator messaging.IAuthenticator

	// connection host and port the server can be reached at
	connectAddr string

	// Thing level operations added by the http router
	//operations []HttpOperation

	// the root http router
	router *chi.Mux

	// The routes that require authentication. These can be used by sub-protocol bindings.
	protectedRoutes chi.Router
	// The routes that do not require authentication. These can be used by sub-protocol bindings.
	publicRoutes chi.Router

	// notification handler to allow devices to send notifications over http
	// intended for use by integration with 3rd party libraries
	serverNotificationHandler messaging.NotificationHandler

	// handler for incoming request messages
	// (after converting requests to the standard internal format)
	serverRequestHandler messaging.RequestHandler

	// response handler to allow devices to send responses over http
	// intended for use by integration with 3rd party libraries
	serverResponseHandler messaging.ResponseHandler
}

// CloseAll does nothing as http is connectionless.
func (srv *HttpWoTBasicServer) CloseAll() {
}

// CloseAllClientConnections does nothing as http is connectionless.
func (srv *HttpWoTBasicServer) CloseAllClientConnections(clientID string) {
	_ = clientID
}

// EnableStatic adds a path to read files from the static directory. Auth required.
//
//	base is the base path on which to serve the static files, eg: "/static"
//	staticRoot is the root directory where static files are kept. This must be a full path.
func (srv *HttpWoTBasicServer) EnableStatic(base string, staticRoot string) error {
	if srv.protectedRoutes == nil || base == "" {
		return fmt.Errorf("no protected route or invalid parameters")
	}
	var staticFileServer http.Handler
	// if staticRoot == "" {
	// 	staticFileServer = http.FileServer(
	// 		&StaticFSWrapper{
	// 			FileSystem:   http.FS(src.EmbeddedStatic),
	// 			FixedModTime: time.Now(),
	// 		})
	// } else {
	// during development when run from the 'hub' project directory
	staticFileServer = http.FileServer(http.Dir(staticRoot))
	// }
	staticPath := base + "/*"
	srv.protectedRoutes.Get(staticPath, staticFileServer.ServeHTTP)
	return nil
}

// GetAuthServerURI returns the URI of the authentication server to include in the TD security scheme
// FIXME: Should this be some kind of authorization flow with a web page?
// This is currently just the login endpoint (post /authn/login).
// The http server might need to include a web page where users can enter their login
// name and password, although that won't work for machines... tbd
//
// Note that web browsers do not directly access the runtime endpoints.
// Instead a web server (hiveoview or other) provides the user interface.
// Including the auth endpoint here is currently just a hint. How to integrate this?
func (srv *HttpWoTBasicServer) GetAuthServerURI() string {
	return HttpPostLoginPath
}

// GetConnectionByConnectionID returns nil as http-basic is connectionless
func (srv *HttpWoTBasicServer) GetConnectionByConnectionID(clientID, cid string) messaging.IConnection {
	_ = clientID
	_ = cid
	return nil
}

// GetConnectionByClientID returns returns nil as http-basic is connectionless
func (srv *HttpWoTBasicServer) GetConnectionByClientID(agentID string) messaging.IConnection {
	_ = agentID
	return nil
}

// GetConnectURL returns connection url of the http server
func (srv *HttpWoTBasicServer) GetConnectURL() string {

	baseURL := fmt.Sprintf("https://%s", srv.connectAddr)
	return baseURL
}

// GetForm returns a form for the given operation
func (srv *HttpWoTBasicServer) GetForm(operation string, thingID string, name string) *td.Form {
	// FIXME: why does a server need this? - its in the TD for the client ...???
	return nil
}

// GetProtectedRouter return the router for adding protected paths.
// Protected means the client is authenticated.
func (srv *HttpWoTBasicServer) GetProtectedRouter() chi.Router {
	return srv.protectedRoutes
}

func (srv *HttpWoTBasicServer) GetProtocolType() string {
	return messaging.ProtocolTypeHTTPBasic
}

// GetPublicRouter return the router for adding public paths.
func (srv *HttpWoTBasicServer) GetPublicRouter() chi.Router {
	return srv.publicRoutes
}

// SendNotification does nothing as http-basic is connectionless
func (srv *HttpWoTBasicServer) SendNotification(msg *messaging.NotificationMessage) {
}

// Start listening on the routes
func (srv *HttpWoTBasicServer) Start() error {
	slog.Info("Starting http-basic server, Listening on: " + srv.GetConnectURL())

	//srv.setupRouting(srv.router)
	return nil
}
func (srv *HttpWoTBasicServer) Stop() {
}

// NewWoTHttpBasicServer creates a new WoT http-basic protocol binding.
// Intended for use as server for sub-protocols such as sse and wss.
//
//	connectAddr is the host:port the server can be reached at.
//	router is the router to register paths at.
//
// On startup this creates a public and protected route. Protected routes can be
// registered by sub-protocols. This http-basic handles the connection authentication.
func NewWoTHttpBasicServer(
	connectAddr string,
	router *chi.Mux,
	authenticator messaging.IAuthenticator,

	handleNotification messaging.NotificationHandler,
	handleRequest messaging.RequestHandler,
	handleResponse messaging.ResponseHandler,
) *HttpWoTBasicServer {

	srv := &HttpWoTBasicServer{
		authenticator:             authenticator,
		connectAddr:               connectAddr,
		serverNotificationHandler: handleNotification,
		serverRequestHandler:      handleRequest,
		serverResponseHandler:     handleResponse,
		router:                    router,
	}
	// TODO: I'd rather not setup routes until start
	srv.setupRouting(srv.router)

	return srv
}
