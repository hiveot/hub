package httpserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils/tlsserver"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/exp/slices"
	"io"
	"log/slog"
	"net/http"
)

// HTTP protoocol headers constants
const (
	ConnectionIDHeader  = "cid"
	DefaultHttpsPort    = 8444
	HttpPostLoginPath   = "/authn/login"
	HttpPostLogoutPath  = "/authn/logout"
	HttpPostRefreshPath = "/authn/refresh"
	HttpGetPingPath     = "/ping"

	// HttpBasicRequestHRef is the generic HTTP path for sending requests to the server.
	// this can be used with http-basic profile when building forms
	//HttpBasicRequestHRef = "/httpbasic/{operation}/{thingID}/{name}"
)

type HttpOperation struct {
	ops         []string
	method      string
	subprotocol string
	url         string
	handler     http.HandlerFunc
	//isThingLevel bool
}

// HttpTransportServer is the transport binding server for HTTPS.
//
// This wraps the library's https server and add routes, middleware, forms,
// and authentication.
// Intended for use with the SSE and WSS sub-protocols which inject their own routes.
type HttpTransportServer struct {

	// TLS server and router
	httpServer *tlsserver.TLSServer
	// host and https port the server listens on
	hostName string
	port     int

	router *chi.Mux
	// The routes that require authentication. These can be added to
	// for sub-protocol bindings such as sse and wss.
	protectedRoutes chi.Router

	// authenticator for logging in and validating session tokens
	authenticator transports.IAuthenticator

	// Thing level operations added by the http router
	operations []HttpOperation
}

// AddOps adds one or more protocol binding operations with a path and handler
// This will be added as a protected route that requires authentication.
//
// Intended for adding operations for http routes and for sub-protocol bindings.
//
//	r is the router to use or nil for the default protected route.
//	ops is a list of operations to register with this URL.
//	method is the HTTP method to use
//	opURL is the URL for the operation(s). Can contain URI variables.
//	handler is the server handler for the operation
func (svc *HttpTransportServer) AddOps(
	r chi.Router, ops []string, method string, opURL string, handler http.HandlerFunc) {

	if r == nil {
		r = svc.protectedRoutes
	}
	svc.operations = append(svc.operations, HttpOperation{
		ops:     ops,
		method:  method,
		url:     opURL,
		handler: handler,
		//isThingLevel: thingLevel,
	})
	r.Method(method, opURL, handler)
}

// setupRouting creates the middleware chain for handling requests, including
// recoverer, compression and token verification for protected routes.
//
// This includes the unprotected routes for login and ping (for now)
// This includes the protected routes for refresh and logout. (for now)
// Everything else should be added by the sub-protocols.
//
// Routes are added by (sub)protocols such as http-basic, sse and wss.
func (svc *HttpTransportServer) setupRouting(router chi.Router) http.Handler {

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
	//router.Use(csrfMiddleware)
	//router.Use(middleware.Compress(5,
	//	"text/html", "text/css", "text/javascript", "image/svg+xml"))

	//--- public routes do not require an authenticated session
	router.Group(func(r chi.Router) {
		r.Use(middleware.Compress(5,
			"text/html", "text/css", "text/javascript", "image/svg+xml"))

		//r.Get("/static/*", staticFileServer.ServeHTTP)
		// build-in REST API for easy login to obtain a token
		svc.AddOps(r, []string{""},
			http.MethodPost, HttpPostLoginPath, svc.HandleLogin)

		svc.AddOps(r, []string{wot.HTOpPing},
			http.MethodGet, HttpGetPingPath, svc.HandlePing)
	})

	//--- private routes that requires authentication (as published in the TD)
	router.Group(func(r chi.Router) {
		r.Use(middleware.Compress(5,
			"text/html", "text/css", "text/javascript", "image/svg+xml"))

		// client sessions authenticate the sender
		r.Use(AddSessionFromToken(svc.authenticator))

		// Using AddOps without provider router will add paths to this route.
		svc.protectedRoutes = r

		// authn service actions needed for all http (sub)protocols
		svc.AddOps(r, []string{},
			http.MethodPost, HttpPostRefreshPath, svc.HandleAuthRefresh)
		svc.AddOps(r, []string{},
			http.MethodPost, HttpPostLogoutPath, svc.HandleLogout)
	})
	return router
}

//// GetConnectionByConnectionID returns the client connection for sending messages to a client
//func (svc *HttpTransportServer) GetConnectionByConnectionID(connectionID string) transports.IServerConnection {
//	return svc.cm.GetConnectionByConnectionID(connectionID)
//}

// GetConnectURL returns connection url of the http server
func (svc *HttpTransportServer) GetConnectURL() string {
	baseURL := fmt.Sprintf("https://%s:%d", svc.hostName, svc.port)
	return baseURL
}

// GetAuthURL returns the url of the http basic authentication service
func (svc *HttpTransportServer) GetAuthURL() string {
	authURL := fmt.Sprintf("https://%s:%d%s", svc.hostName, svc.port, HttpPostLoginPath)
	return authURL
}

// GetForm returns a new HTTP form for the given operation
// Intended for Thing level operations
func (svc *HttpTransportServer) GetForm(op, thingID, name string) *td.Form {

	// Operations use URI variables in URLs for selecting things.
	for _, httpOp := range svc.operations {
		if slices.Contains(httpOp.ops, op) {
			form := td.NewForm(op, httpOp.url)
			form["htv:methodName"] = httpOp.method
			return &form
		}
	}

	slog.Info("GetForm. No form found for operation", "op", op)
	return nil
}

// HandleLogin handles a login request, posted by a consumer.
// This is the only unprotected route supported.
// This uses the configured session authenticator.
func (svc *HttpTransportServer) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var reply any
	var args map[string]string

	payload, err := io.ReadAll(r.Body)
	if err == nil {
		err = jsoniter.Unmarshal(payload, &args)
	}
	if err == nil {
		// the login is handled in-house and has an immediate return
		// TODO: use-case for 3rd party login? oauth2 process support? tbd
		// FIXME: hard-coded keys!? ugh
		clientID := args["login"]
		password := args["password"]
		reply, err = svc.authenticator.Login(clientID, password)
		slog.Info("HandleLogin", "clientID", clientID)
	}
	if err != nil {
		slog.Warn("HandleLogin failed:", "err", err.Error())
		svc.WriteError(w, err, http.StatusBadRequest)
		return
	}
	// TODO: set client session cookie for browser clients
	//svc.sessionManager.SetSessionCookie(cs.sessionID,token)
	svc.WriteReply(w, reply, transports.StatusCompleted, nil)
}

// HandleAuthRefresh refreshes the auth token using the session authenticator.
// The session authenticator is that of the authn service. This allows testing with a dummy
// authenticator without having to run the authn service.
func (svc *HttpTransportServer) HandleAuthRefresh(w http.ResponseWriter, r *http.Request) {
	var newToken string
	var oldToken string
	rp, err := GetRequestParams(r, &oldToken)

	slog.Info("HandleAuthRefresh", "clientID", rp.ClientID)

	if err == nil {
		newToken, err = svc.authenticator.RefreshToken(rp.ClientID, oldToken)
	}
	if err != nil {
		slog.Warn("HandleAuthRefresh failed:", "err", err.Error())
		svc.WriteError(w, err, 0)
		return
	}
	svc.WriteReply(w, newToken, transports.StatusCompleted, nil)
}

// HandleLogout ends the session and closes all client connections
func (svc *HttpTransportServer) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// use the authenticator
	rp, err := GetRequestParams(r, nil)
	if err == nil {
		slog.Info("HandleLogout", "clientID", rp.ClientID)
		svc.authenticator.Logout(rp.ClientID)
	}
	svc.WriteReply(w, nil, transports.StatusCompleted, err)
}

// HandlePing with http handler returns a pong response
func (svc *HttpTransportServer) HandlePing(w http.ResponseWriter, r *http.Request) {
	// simply return a pong message
	svc.WriteReply(w, "pong", transports.StatusCompleted, nil)
}

// Stop the https server
func (svc *HttpTransportServer) Stop() {
	slog.Info("Stopping HttpTransportServer")

	// Note: closing the TLS server does not shut down active connections
	svc.httpServer.Stop()
}

// WriteError is a convenience function that logs and writes an error
// If the reply has an error then write a bad request with the error as payload
// This also writes the StatusHeader containing StatusFailed.
func (svc *HttpTransportServer) WriteError(w http.ResponseWriter, err error, code int) {
	if code == 0 {
		code = http.StatusBadRequest
	}
	if err != nil {
		slog.Warn("Request error: ", "err", err.Error())
		http.Error(w, err.Error(), code)
	} else {
		//replyHeader := w.Header()
		//replyHeader.Set(StatusHeader, transports.StatusCompleted)
		w.WriteHeader(code)
	}
}

// WriteReply is a convenience function that serializes the data and writes it as a response,
// optionally reporting an error with code BadRequest.
//
// status is completed,failed,... set in the 'StatusHeader' reply header if provided.
// only used by hiveot.
func (svc *HttpTransportServer) WriteReply(
	w http.ResponseWriter, data any, status string, err error) {
	var payloadJSON string

	if status != "" {
		//replyHeader := w.Header()
		//replyHeader.Set(StatusHeader, status)
	}
	if data != nil {
		payloadJSON, _ = jsoniter.MarshalToString(data)
		tmp, _ := jsoniter.Marshal(data)
		payloadJSON = string(tmp)
	}
	if status == transports.StatusFailed {
		var payload string
		if err != nil {
			payload = err.Error()
		}
		http.Error(w, payload, http.StatusBadRequest)

	} else if status == transports.StatusCompleted {
		if payloadJSON != "" {
			_, _ = w.Write([]byte(payloadJSON))
		}
		w.WriteHeader(http.StatusOK)
	} else if status == transports.StatusRunning {
		// Code 200: https://w3c.github.io/wot-profile/#example-17
		w.WriteHeader(http.StatusOK)
		if payloadJSON != "" {
			_, _ = w.Write([]byte(payloadJSON))
		} else {
			slog.Error("Expected a ActionStatus or ResponseMessage payload. Got nothing")
		}
	} else {
		// status is pending, possibly an ActionStatus payload
		// Code 201: https://w3c.github.io/wot-profile/#sec-http-sse-profile
		w.WriteHeader(201)
		if payloadJSON != "" {
			_, _ = w.Write([]byte(payloadJSON))
		}
	}
}

// StartHttpTransportServer creates and starts a new instance of the HTTPS Server
// with JWT authentication and SSE/SSE-SC/WS sub-protocol bindings.
//
// Call stop to end the transport server.
//
//	host, port with the server listening address (or "") and port
//	serverCert: the TLS certificate of this server
//	caCert: the CA public certificate that signed the server cert
//	authenticator: plugin to authenticate requests
func StartHttpTransportServer(host string, port int,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	authenticator transports.IAuthenticator,
) (*HttpTransportServer, error) {

	httpServer, httpRouter := tlsserver.NewTLSServer(
		host, port, serverCert, caCert)

	//wssURL := fmt.Sprintf("wss://%s:%d", config.Host, config.Port)
	svc := HttpTransportServer{
		authenticator: authenticator,
		hostName:      host,
		port:          port,
		httpServer:    httpServer,
		router:        httpRouter,
		//cm:         cm,
	}

	svc.setupRouting(svc.router)
	err := svc.httpServer.Start()
	return &svc, err
}
