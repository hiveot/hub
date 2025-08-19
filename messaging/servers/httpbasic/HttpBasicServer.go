package httpbasic

import (
	"fmt"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/wot/td"
)

// HTTP-basic protocol headers constants
const (
	ConnectionIDHeader  = "cid"
	HttpPostLoginPath   = "/authn/login"
	HttpPostLogoutPath  = "/authn/logout"
	HttpPostRefreshPath = "/authn/refresh"
	HttpGetPingPath     = "/ping"

	HttpBasicAffordanceOperationPath = "/things/{op}/{thingID}/{name}"
	HttpBasicThingOperationPath      = "/things/{op}/{thingID}"

	// HttpBasicRequestHRef is the generic HTTP path for sending requests to the server.
	// this can be used with http-basic profile when building forms
	//HttpBasicRequestHRef = "/httpbasic/{operation}/{thingID}/{name}"
)

// HttpBasicServer provides the http-basic protocol binding using the provided http server.
// This is the simplest protocol binding supported by hiveot.
// Features:
// - login to obtain a bearer token:  POST {base}/authn/login
// - refresh bearer token:            POST {base}/authn/refresh
// - post/get thing operations:       POST {base}/things/{op}/{thingID}
// - post/get affordance operations:  POST {base}/things/{op}/{thingID}/{name}
//
// This uses the provided httpserver instance.
type HttpBasicServer struct {
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
func (srv *HttpBasicServer) CloseAll() {
}

// CloseAllClientConnections does nothing as http is connectionless.
func (srv *HttpBasicServer) CloseAllClientConnections(clientID string) {
	_ = clientID
}

// GetConnectionByConnectionID returns nil as http-basic is connectionless
func (srv *HttpBasicServer) GetConnectionByConnectionID(clientID, cid string) messaging.IConnection {
	_ = clientID
	_ = cid
	return nil
}

// GetConnectionByClientID returns returns nil as http-basic is connectionless
func (srv *HttpBasicServer) GetConnectionByClientID(agentID string) messaging.IConnection {
	_ = agentID
	return nil
}

// GetConnectURL returns connection url of the http server
func (srv *HttpBasicServer) GetConnectURL() string {

	baseURL := fmt.Sprintf("https://%s", srv.connectAddr)
	return baseURL
}

// GetForm returns a form for the given operation
func (srv *HttpBasicServer) GetForm(operation string, thingID string, name string) *td.Form {
	// FIXME: why does a server need this? - its in the TD for the client ...???
	return nil
}

// GetProtectedRouter return the router for adding protected paths.
// Protected means the client is authenticated.
func (srv *HttpBasicServer) GetProtectedRouter() chi.Router {
	return srv.protectedRoutes
}

func (srv *HttpBasicServer) GetProtocolType() string {
	return messaging.ProtocolTypeHTTPBasic
}

// GetPublicRouter return the router for adding public paths.
func (srv *HttpBasicServer) GetPublicRouter() chi.Router {
	return srv.publicRoutes
}

// SendNotification does nothing as http-basic is connectionless
func (srv *HttpBasicServer) SendNotification(msg *messaging.NotificationMessage) {
}

// Start listening on the routes
func (srv *HttpBasicServer) Start() error {
	slog.Info("Starting http-basic server, Listening on: " + srv.GetConnectURL())

	// Add the routes used in SSE connection and subscription requests
	// hmm, needed by sub-protocols before starting
	//srv.setupRouting(srv.router)
	return nil
}
func (srv *HttpBasicServer) Stop() {
}

// NewHttpBasicServer creates a new http-basic protocol binding.
// Intended for use as server for sub-protocols such as sse and wss.
//
//	connectAddr is the host:port the server can be reached at.
//	router is the router to register paths at.
//
// On startup this creates a public and protected route. Protected routes can be
// registered by sub-protocols. This http-basic handles the connection authentication.
func NewHttpBasicServer(
	connectAddr string,
	router *chi.Mux,
	authenticator messaging.IAuthenticator,

	handleNotification messaging.NotificationHandler,
	handleRequest messaging.RequestHandler,
	handleResponse messaging.ResponseHandler,
) *HttpBasicServer {

	srv := &HttpBasicServer{
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
