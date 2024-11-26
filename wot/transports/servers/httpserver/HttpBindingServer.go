package httpserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/lib/tlsserver"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/connections"
	"github.com/hiveot/hub/wot/transports/servers/ssescserver"
	"github.com/hiveot/hub/wot/transports/servers/wssserver"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"net/http"
)

type HttpOperation struct {
	op           string
	method       string
	subprotocol  string
	url          string
	handler      http.HandlerFunc
	isThingLevel bool
}

// HttpBindingServer is the transport binding server for HTTPS
// This wraps the library's https server and add routes and middleware for use in the binding
type HttpBindingServer struct {
	// port and path configuration
	config *HttpBindingConfig

	// registered handler of received events or requests (which return a reply)
	messageHandler transports.ServerMessageHandler

	// TLS server and router
	httpServer *tlsserver.TLSServer
	router     *chi.Mux

	// subprotocol bindings
	//sse   *sse.SseBindingServer
	ssesc *ssescserver.SseScBindingServer
	ws    *wssserver.WssBindingServer

	// authenticator for logging in and validating session tokens
	authenticator transports.IAuthenticator

	// Thing level operations
	operations []HttpOperation

	// connection manager for adding/removing binding connections
	cm *connections.ConnectionManager
}

// AddGetOp adds protocol binding operation with a URL and handler
//
// This is used to add Forms to the digitwin TDs
func (svc *HttpBindingServer) AddGetOp(r chi.Router,
	op string, thingLevel bool, opURL string, handler http.HandlerFunc) {

	svc.operations = append(svc.operations, HttpOperation{
		op:           op,
		method:       http.MethodGet,
		url:          opURL,
		handler:      handler,
		isThingLevel: thingLevel,
	})
	r.Get(opURL, handler)
}

// AddPostOp adds protocol binding operation with a URL and handler
//
// This is used to add Forms to the digitwin TDs
func (svc *HttpBindingServer) AddPostOp(r chi.Router,
	op string, isThingLevel bool, opURL string, handler http.HandlerFunc) {
	svc.operations = append(svc.operations, HttpOperation{
		op:           op,
		method:       http.MethodPost,
		url:          opURL,
		handler:      handler,
		isThingLevel: isThingLevel,
	})
	r.Post(opURL, handler)
}

// GetConnectionByConnectionID returns the client connection for sending messages to a client
func (svc *HttpBindingServer) GetConnectionByConnectionID(connectionID string) transports.IServerConnection {
	return svc.cm.GetConnectionByConnectionID(connectionID)
}

// GetProtocolInfo returns info on the protocol supported by this binding
func (svc *HttpBindingServer) GetProtocolInfo() transports.ProtocolInfo {
	hostName := svc.config.Host
	if hostName == "" {
		hostName = "localhost"
	}
	baseURL := fmt.Sprintf("https://%s:%d", hostName, svc.config.Port)
	inf := transports.ProtocolInfo{
		BaseURL:   baseURL,
		Schema:    "https",
		Transport: "https",
	}
	return inf
}

// GetHttpServer returns the running tls server. Intended for testing
func (svc *HttpBindingServer) GetHttpServer() *tlsserver.TLSServer {
	return svc.httpServer
}

// Stop the https server
func (svc *HttpBindingServer) Stop() {
	slog.Info("Stopping HttpBindingServer")

	// Shutdown remaining sessions to avoid hanging.
	// (closing the TLS server does not shut down active connections)
	//sm := sessions.GetSessionManager()
	//sm.RemoveAll()
	svc.httpServer.Stop()
}

// writeError is a convenience function that logs and writes an error
// If the reply has an error then write a bad request with the error as payload
func (svc *HttpBindingServer) writeError(w http.ResponseWriter, err error, code int) {
	if code == 0 {
		code = http.StatusBadRequest
	}
	if err != nil {
		slog.Warn("Request error: ", "err", err.Error())
		http.Error(w, err.Error(), code)
	} else {
		w.WriteHeader(code)
	}
}

// writeReply is a convenience function that serializes the data and writes it as a response,
// optionally reporting an error with code BadRequest.
func (svc *HttpBindingServer) writeReply(w http.ResponseWriter, data any, err error) {
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

// StartHttpBindingServer creates and starts a new instance of the HTTPS Server
// with JWT authentication and SSE/SSE-SC/WS sub-protocol bindings.
//
// Call stop to end the transport server.
//
//	config
//	privKey
//	caCert
//	sessionAuth for creating and validating authentication tokens
//	dtwService that handles digital thing requests
func StartHttpBindingServer(config *HttpBindingConfig,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	authenticator transports.IAuthenticator,
	messageHandler transports.ServerMessageHandler,
	cm *connections.ConnectionManager,
) (*HttpBindingServer, error) {

	httpServer, httpRouter := tlsserver.NewTLSServer(
		config.Host, config.Port, serverCert, caCert)

	svc := HttpBindingServer{
		authenticator: authenticator,
		config:        config,

		ws: wssserver.NewWssBindingServer(cm, messageHandler),
		//sse:   ssescserver.NewSseScBindingServer(cm),
		ssesc: ssescserver.NewSseScBindingServer(cm),

		httpServer: httpServer,
		router:     httpRouter,
		cm:         cm,
	}

	svc.createRoutes(svc.router)
	err := svc.httpServer.Start()
	return &svc, err
}
