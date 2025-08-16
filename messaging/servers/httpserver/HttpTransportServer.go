package httpserver

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/tputils/net"
	"github.com/hiveot/hub/messaging/tputils/tlsserver"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/exp/slices"
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
	connectAddr string
	port        int

	router *chi.Mux
	// The routes that require authentication. These can be added to
	// for sub-protocol bindings such as sse and wss.
	protectedRoutes chi.Router

	// authenticator for logging in and validating session tokens
	authenticator messaging.IAuthenticator

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

// AddSecurityScheme adds the security scheme that this http protocol supports.
// This is also applicable to any sub-protocol such as websocket.
//
// http supports bearer tokens for request authentication, basic and digest authentication
// for logging in.
func (svc *HttpTransportServer) AddSecurityScheme(tdoc *td.TD) {

	// FIXME: this should be set by the authenticator used

	// bearer security scheme for authenticating http and subprotocol connections
	format, alg := svc.authenticator.GetAlg()

	tdoc.AddSecurityScheme("bearer_sc", td.SecurityScheme{
		//AtType:        nil,
		Description: "Bearer token authentication",
		//Descriptions:  nil,
		//Proxy:         "",
		Scheme: "bearer", // nosec, basic, digest, bearer, psk, oauth2, apikey or auto
		//Authorization: authServerURI,// n/a as the token is the authorization
		Name:   "authorization",
		Alg:    alg,
		Format: format,   // jwe, cwt, jws, jwt, paseto
		In:     "header", // query, body, cookie, uri, auto
	})
	// bearer security scheme for authenticating http digest connections
	// tbd. clients should login and use bearer tokens.
	//tdoc.AddSecurityScheme("digest_sc", td.SecurityScheme{
	//	Description: "Digest authentication",
	//	Scheme:      "digest", // nosec, basic, digest, bearer, psk, oauth2, apikey or auto
	//	In:          "body",   // query, header, body, cookie, uri, auto
	//})
}

// AddTDForms adds forms for use of the HTTP requests with the given TD
// 'includeAffordances' adds forms to all affordances to be compliant with the specifications.
// Warning this increases the TD size significantly.
func (svc *HttpTransportServer) AddTDForms(tdoc *td.TD, includeAffordances bool) {
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

	baseURL := fmt.Sprintf("https://%s:%d", svc.connectAddr, svc.port)
	return baseURL
}

// GetAuthURL returns the url of the http basic authentication service
func (svc *HttpTransportServer) GetAuthURL() string {
	authURL := fmt.Sprintf("https://%s:%d%s", svc.connectAddr, svc.port, HttpPostLoginPath)
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
//
// Body contains {"login":name, "password":pass} format
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
		svc.WriteError(w, err, http.StatusUnauthorized)
		return
	}
	// TODO: set client session cookie for browser clients
	//svc.sessionManager.SetSessionCookie(cs.sessionID,token)
	svc.WriteReply(w, true, reply, nil)
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
	svc.WriteReply(w, true, newToken, nil)
}

// HandleLogout ends the session and closes all client connections
func (svc *HttpTransportServer) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// use the authenticator
	rp, err := GetRequestParams(r, nil)
	if err == nil {
		slog.Info("HandleLogout", "clientID", rp.ClientID)
		svc.authenticator.Logout(rp.ClientID)
	}
	svc.WriteReply(w, true, nil, err)
}

// HandlePing with http handler returns a pong response
func (svc *HttpTransportServer) HandlePing(w http.ResponseWriter, r *http.Request) {
	// simply return a pong message
	svc.WriteReply(w, true, "pong", nil)
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
// when handled, this returns a 200 status code if no error is returned.
// handled is false means the request is in progress. This returns a 201.
// if an err is returned this returns a 400 bad request or 403 unauthorized error code
// the data can contain error details.
func (svc *HttpTransportServer) WriteReply(
	w http.ResponseWriter, handled bool, data any, err error) {
	var payloadJSON string

	if data != nil {
		payloadJSON, _ = jsoniter.MarshalToString(data)
	}
	if err != nil {
		if errors.Is(err, messaging.UnauthorizedError) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	} else if handled {
		if payloadJSON != "" {
			_, _ = w.Write([]byte(payloadJSON))
		}
		// Code 200: https://w3c.github.io/wot-profile/#example-17
		w.WriteHeader(http.StatusOK)
	} else {
		// not handled no error. response will be async
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
//	host and port with the server listening address (or "") and port
//	serverCert: the TLS certificate of this server
//	caCert: the CA public certificate that signed the server cert
//	authenticator: plugin to authenticate requests
func StartHttpTransportServer(host string, port int,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	authenticator messaging.IAuthenticator,
) (*HttpTransportServer, error) {

	// if host is empty then listen on all interfaces
	httpServer, httpRouter := tlsserver.NewTLSServer(
		host, port, serverCert, caCert)

	// if no listening address is provided then use the address in the server cert
	// or fall back to the outbound address for discovery.
	// The test server must provide a host to avoid a problem with connecting as
	// the test cert might not use outboundIP.
	connectAddr := host
	//if connectAddr == "" {
	//	b := pem.Block{Type: "CERTIFICATE", Bytes: serverCert.Certificate[0]}
	//	x509Cert, err := x509.ParseCertificate(b.Bytes)
	//	if err == nil {
	//		if len(x509Cert.IPAddresses) > 0 {
	//			connectAddr = x509Cert.IPAddresses[0].String()
	//		} else if len(x509Cert.DNSNames) > 0 {
	//			connectAddr = x509Cert.DNSNames[0]
	//		}
	//	}
	//}
	if connectAddr == "" {
		connectIP := net.GetOutboundIP("")
		connectAddr = connectIP.String()
	}

	//wssURL := fmt.Sprintf("wss://%s:%d", config.Host, config.Port)
	svc := HttpTransportServer{
		authenticator: authenticator,
		connectAddr:   connectAddr,
		port:          port,
		httpServer:    httpServer,
		router:        httpRouter,
		//cm:         cm,
	}

	svc.setupRouting(svc.router)
	err := svc.httpServer.Start()
	return &svc, err
}
