package httpserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/connections"
	"github.com/hiveot/hub/transports/tputils/tlsserver"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"net/http"
)

// HttpTransportServer is the transport binding server for HTTPS
// This wraps the library's https server and add routes and middleware for use in the binding
type HttpTransportServer struct {

	// registered handler of received notifications (sent by agents)
	serverNotificationHandler transports.ServerNotificationHandler
	// registered handler of requests (which return a reply)
	serverRequestHandler transports.ServerRequestHandler
	// registered handler of responses (which sends a reply to the request sender)
	serverResponseHandler transports.ServerResponseHandler

	// TLS server and router
	httpServer *tlsserver.TLSServer
	// host and https port the server listens on
	hostName string
	port     int

	router *chi.Mux
	// The routes that require authentication. These can be added to
	// for sub-protocol bindings such as sse and wss.
	protectedRoutes chi.Router

	// subprotocol bindings
	//sse   *sse.SseBindingServer
	//ssesc *ssescserver.SseScTransportServer
	//ws    *wssserver.WssTransportServer

	// authenticator for logging in and validating session tokens
	authenticator transports.IAuthenticator

	// Thing level operations added by the http router
	operations []HttpOperation

	// connection manager for adding/removing binding connections
	cm *connections.ConnectionManager
}

// AddOps adds one or more protocol binding operations with a path and handler
// This will be added as a protected route that requires authentication.
// Intended for adding operations for http routes and for sub-protocol bindings.
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

// GetConnectionByConnectionID returns the client connection for sending messages to a client
func (svc *HttpTransportServer) GetConnectionByConnectionID(connectionID string) transports.IServerConnection {
	return svc.cm.GetConnectionByConnectionID(connectionID)
}

// GetConnectURL returns connection url of the http server
func (svc *HttpTransportServer) GetConnectURL() string {
	baseURL := fmt.Sprintf("https://%s:%d", svc.hostName, svc.port)
	return baseURL
}

// SendNotification broadcast an event or property change to subscribers clients
func (svc *HttpTransportServer) SendNotification(msg transports.NotificationMessage) {
	cList := svc.cm.GetConnectionByProtocol(transports.ProtocolTypeHTTPS)
	for _, c := range cList {
		c.SendNotification(msg)
	}
}

// Stop the https server
func (svc *HttpTransportServer) Stop() {
	slog.Info("Stopping HttpTransportServer")

	// Note: closing the TLS server does not shut down active connections
	svc.httpServer.Stop()
}

// writeError is a convenience function that logs and writes an error
// If the reply has an error then write a bad request with the error as payload
// This also writes the StatusHeader containing StatusFailed.
func (svc *HttpTransportServer) writeError(w http.ResponseWriter, err error, code int) {
	if code == 0 {
		code = http.StatusBadRequest
	}
	if err != nil {
		slog.Warn("Request error: ", "err", err.Error())
		http.Error(w, err.Error(), code)
	} else {
		replyHeader := w.Header()
		replyHeader.Set(StatusHeader, transports.StatusCompleted)
		w.WriteHeader(code)
	}
}

// writeReply is a convenience function that serializes the data and writes it as a response,
// optionally reporting an error with code BadRequest.
//
// status is completed,failed,... set in the 'StatusHeader' reply header if provided.
// only used by hiveot.
func (svc *HttpTransportServer) writeReply(
	w http.ResponseWriter, data any, status string, err error) {

	if status != "" {
		replyHeader := w.Header()
		replyHeader.Set(StatusHeader, status)
	}
	if err != nil {
		slog.Warn("Request error: ", "err", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else if data != nil {
		// If no header is written then w.Write writes a StatusOK
		payload, _ := jsoniter.Marshal(data)
		_, _ = w.Write(payload)
	} else {
		// Only write header if no data is written
		w.WriteHeader(http.StatusOK)
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
//	cm: handler of new incoming connections
//	serverRequestHandler: handler of incoming requests from clients
//	serverResponseHandler: handler of incoming response from agents
//	serverNotificationHandler: handler if incoming notifications from agents
func StartHttpTransportServer(host string, port int,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	authenticator transports.IAuthenticator,
	cm *connections.ConnectionManager,
	handleNotification transports.ServerNotificationHandler,
	handleRequest transports.ServerRequestHandler,
	handleResponse transports.ServerResponseHandler,
) (*HttpTransportServer, error) {

	httpServer, httpRouter := tlsserver.NewTLSServer(
		host, port, serverCert, caCert)

	//wssURL := fmt.Sprintf("wss://%s:%d", config.Host, config.Port)
	svc := HttpTransportServer{
		authenticator: authenticator,

		serverRequestHandler:      handleRequest,
		serverResponseHandler:     handleResponse,
		serverNotificationHandler: handleNotification,

		hostName:   host,
		port:       port,
		httpServer: httpServer,
		router:     httpRouter,
		cm:         cm,
	}

	svc.createRoutes(svc.router)
	err := svc.httpServer.Start()
	return &svc, err
}
