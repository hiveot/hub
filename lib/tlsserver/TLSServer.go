// Package tlsserver with TLS server for use by plugins and testing
package tlsserver

import (
	"context"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"golang.org/x/exp/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/cors"

	"github.com/gorilla/mux"
)

// TLSServer is a simple TLS MsgServer supporting BASIC, Jwt and client certificate authentication
type TLSServer struct {
	address           string
	port              uint
	caCert            *x509.Certificate
	serverCert        *tls.Certificate
	httpServer        *http.Server
	router            *mux.Router
	httpAuthenticator *HttpAuthenticator

	//jwtIssuer *JWTIssuer
}

// AddHandler adds a new handler for a path.
//
// The server authenticates the request before passing it to this handler.
// The handler's userID is that of the authenticated user, and is intended for authorization of the request.
// If authentication is not enabled then the userID is empty.
//
// apply .Method(http.MethodXyz) to restrict the accepted HTTP methods
//
//	path to listen on. See https://github.com/gorilla/mux
//	handler to invoke with the request. The userID is only provided when an authenticator is used
//
// Returns the route. Apply '.Method(http.MethodPut|Post|Get)' to restrict the accepted HTTP methods
func (srv *TLSServer) AddHandler(path string,
	handler func(userID string, resp http.ResponseWriter, req *http.Request)) *mux.Route {

	// do we need a local copy of handler? not sure
	local_handler := handler

	// the internal authenticator performs certificate based, basic or jwt token authentication if needed
	route := srv.router.HandleFunc(path, func(resp http.ResponseWriter, req *http.Request) {
		// test, allow CORS if enabled.
		if req.Method == http.MethodOptions {
			// don't return a payload with the cors options request
			return
		}
		msg := fmt.Sprintf("TLSServer.HandleFunc %s: Method=%s from %s. Vars=%s",
			path, req.Method, req.RemoteAddr, mux.Vars(req))
		slog.Info(msg)

		// valid authentication without userID means a plugin certificate was used which is always authorized
		userID, match := srv.httpAuthenticator.AuthenticateRequest(resp, req)
		if !match {
			msg := fmt.Sprintf("TLSServer.HandleFunc %s: User '%s' from %s is unauthorized",
				path, userID, req.RemoteAddr)
			slog.Warn(msg)
			srv.WriteForbidden(resp, msg)
		} else {
			local_handler(userID, resp, req)
		}
	})
	return route
}

// Authenticator returns the authenticator used for this server
func (srv *TLSServer) Authenticator() *HttpAuthenticator {
	return srv.httpAuthenticator
}

// AddHandlerNoAuth adds a new handler for a path that does not require authentication
// The server passes the request directly to the handler
//
//	path to listen on. This supports wildcards
//	handler to invoke with the request. The userID is only provided when an authenticator is used
//
// Returns the route. Apply '.Method(http.MethodPut|Post|Get)' to restrict the accepted HTTP methods
func (srv *TLSServer) AddHandlerNoAuth(path string,
	handler func(resp http.ResponseWriter, req *http.Request)) *mux.Route {

	route := srv.router.HandleFunc(path, func(resp http.ResponseWriter, req *http.Request) {
		handler(resp, req)
	})
	return route

}

// EnableBasicAuth enables BASIC authentication on this server
// Basic auth is a legacy authentication scheme and not recommended as it requires each service to
// have access to the credentials store. Use of JwtAuth is preferred.
//
// validateCredentials is the function that verifies the given credentials
func (srv *TLSServer) EnableBasicAuth(validateCredentials func(loginName string, password string) bool) {
	srv.httpAuthenticator.EnableBasicAuth(validateCredentials)
}

// EnableJwtAuth enables JWT authentication using asymmetric keys.
//
// JWT access token is expected to be included in the http header authorization field, and signed by an issuing
// authentication server using the server's private key.
//
// verificationKey is the public key used to verify tokens. Use nil to use the TLS server own public key
func (srv *TLSServer) EnableJwtAuth(verificationKey *ecdsa.PublicKey) {
	if verificationKey == nil {
		issuerKey := srv.serverCert.PrivateKey.(*ecdsa.PrivateKey)
		verificationKey = &issuerKey.PublicKey
	}
	srv.httpAuthenticator.EnableJwtAuth(verificationKey)
}

// Start the TLS server using the provided CA and MsgServer certificates.
// If a client certificate is provided it must be valid.
// This configures handling of CORS requests to allow:
//   - any origin by returning the requested origin (not using wildcard '*').
//   - any method, eg PUT, POST, GET, PATCH,
//   - headers "Origin", "Accept", "Content-Type", "X-Requested-With"
func (srv *TLSServer) Start() error {
	var err error
	var mutex = sync.Mutex{}

	slog.Info("Starting TLS server", "address", srv.address, "port", srv.port)
	if srv.caCert == nil || srv.serverCert == nil {
		err := fmt.Errorf("missing CA or server certificate")
		slog.Error(err.Error())
		return err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(srv.caCert)

	serverTLSConf := &tls.Config{
		Certificates:       []tls.Certificate{*srv.serverCert},
		ClientAuth:         tls.VerifyClientCertIfGiven,
		ClientCAs:          caCertPool,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
	}

	// handle CORS using the cors plugin
	// see also: https://stackoverflow.com/questions/43871637/no-access-control-allow-origin-header-is-present-on-the-requested-resource-whe
	// TODO: add configuration for CORS origin: allowed, sameaddress, exact
	c := cors.New(cors.Options{
		// return the origin as allowed origin
		AllowOriginFunc: func(orig string) bool {
			// local requests are always allowed, even over http (for testing) - todo: disable in production
			if strings.HasPrefix(orig, "https://127.0.0.1") || strings.HasPrefix(orig, "https://localhost") ||
				strings.HasPrefix(orig, "http://127.0.0.1") || strings.HasPrefix(orig, "http://localhost") {
				slog.Debug("TLSServer.AllowOriginFunc: Cors origin Is True", "origin", orig)
				return true
			} else if strings.HasPrefix(orig, "https://"+srv.address) {
				slog.Debug("TLSServer.AllowOriginFunc: Cors origin Is True", "origin", orig)
				return true
			}
			slog.Warn("TLSServer.AllowOriginFunc: Cors: origin Is False", "orig", orig)
			return false
		},
		// default allowed headers is "Origin", "Accept", "Content-Type", "X-Requested-With" (missing authorization)
		AllowedHeaders: []string{"Origin", "Accept", "Content-Type", "Authorization", "Headers"},
		// default is get/put/patch/post/delete/head
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		Debug:            true,
		AllowCredentials: true,
	})
	handler := c.Handler(srv.router)

	srv.httpServer = &http.Server{
		Addr: fmt.Sprintf("%s:%d", srv.address, srv.port),
		// ReadTimeout:  5 * time.Minute, // 5 min to allow for delays when 'curl' on OSx prompts for username/password
		// WriteTimeout: 10 * time.Second,
		Handler:   handler,
		TLSConfig: serverTLSConf,
	}
	// mutex to capture error result in case startup in the background failed
	go func() {
		// serverTLSConf contains certificate and key
		err2 := srv.httpServer.ListenAndServeTLS("", "")
		if err2 != nil && err2 != http.ErrServerClosed {
			mutex.Lock()
			//t := err2.Error()
			err = fmt.Errorf("ListenAndServeTLS: %s", err2.Error())
			slog.Error(err.Error())
			mutex.Unlock()
		}
	}()
	// Make sure the server is listening before continuing
	time.Sleep(time.Second)
	mutex.Lock()
	defer mutex.Unlock()
	return err
}

// Stop the TLS server and close all connections
func (srv *TLSServer) Stop() {
	slog.Info("Stopping TLS server")

	if srv.httpServer != nil {
		srv.httpServer.Shutdown(context.Background())
	}
}

// NewTLSServer creates a new TLS MsgServer instance with authentication support.
// Use AddHandler to handle incoming requests for the given route and indicate if authentication is required.
//
// The following authentication methods are supported:
//
//	Certificate based auth using the caCert to verify client certificates
//	Basic authentication if 'EnableBasicAuth' is used.
//	JWT asymmetric token authentication if EnableJwtAuth is used.
//
//	address        server listening address
//	port           listening port
//	serverCert     MsgServer TLS certificate
//	caCert         CA certificate to verify client certificates
//
// returns TLS server for handling requests
func NewTLSServer(address string, port uint,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
) *TLSServer {

	srv := &TLSServer{
		caCert:     caCert,
		serverCert: serverCert,
		router:     mux.NewRouter(),
	}
	//// support for CORS response headers
	//srv.router.Use(mux.CORSMethodMiddleware(srv.router))

	//issuerKey := serverCert.PrivateKey.(*ecdsa.PrivateKey)
	//serverX509, _ := x509.ParseCertificate(serverCert.Certificate[0])
	//pubKey := certsclient.PublicKeyFromCert(serverX509)

	// Authenticate incoming https requests
	srv.httpAuthenticator = NewHttpAuthenticator()

	srv.address = address
	srv.port = port
	return srv
}
