package httpserver

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/connections"
	"github.com/hiveot/hub/wot/transports/utils/tlsserver"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"net/http"
)

const DefaultHttpsPort = 8444

// HttpTransportServer is the transport binding server for HTTPS
// This wraps the library's https server and add routes and middleware for use in the binding
type HttpTransportServer struct {

	// registered handler of received events or requests (which return a reply)
	messageHandler transports.ServerMessageHandler

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

// AddGetOp adds a protocol binding operation with a path and handler
// This will be added as a protected route that requires authentication.
// Intended for adding operations for http routes and for sub-protocol bindings.
//
// This is used to add Forms to the digitwin TDs
func (svc *HttpTransportServer) AddGetOp(
	op string, opURL string, handler http.HandlerFunc) {

	svc.operations = append(svc.operations, HttpOperation{
		op:      op,
		method:  http.MethodGet,
		url:     opURL,
		handler: handler,
		//isThingLevel: thingLevel,
	})
	svc.protectedRoutes.Get(opURL, handler)
}

// AddPostOp adds protocol binding operation with a URL and handler
// This will be added as a protected route that requires authentication.
// Intended for adding operations for http routes and for sub-protocol bindings.
//
// This is used to add Forms to the digitwin TDs
func (svc *HttpTransportServer) AddPostOp(
	op string, opURL string, handler http.HandlerFunc) {
	svc.operations = append(svc.operations, HttpOperation{
		op:      op,
		method:  http.MethodPost,
		url:     opURL,
		handler: handler,
		//isThingLevel: isThingLevel,
	})
	svc.protectedRoutes.Post(opURL, handler)
}

// GetConnectionByConnectionID returns the client connection for sending messages to a client
func (svc *HttpTransportServer) GetConnectionByConnectionID(connectionID string) transports.IServerConnection {
	return svc.cm.GetConnectionByConnectionID(connectionID)
}

// GetProtocolInfo returns info on the protocol supported by this binding
//func (svc *HttpTransportServer) GetProtocolInfo() transports.ProtocolInfo {
//	//hostName := svc.config.Host
//	//if hostName == "" {
//	//	hostName = "localhost"
//	//}
//	baseURL := fmt.Sprintf("https://%s:%d", svc.hostName, svc.port)
//	inf := transports.ProtocolInfo{
//		BaseURL:   baseURL,
//		Schema:    "https",
//		Transport: "https",
//	}
//	return inf
//}

// SendNotification broadcast an event or property change to subscribers clients
func (svc *HttpTransportServer) SendNotification(operation string, dThingID, name string, data any) {
	// this is needed so mqtt can broadcast once via the message bus instead all individual connections
	// tbd. An embedded mqtt server can still send per connection?
	slog.Error("not supported in http")
}

// Stop the https server
func (svc *HttpTransportServer) Stop() {
	slog.Info("Stopping HttpTransportServer")

	// Shutdown remaining sessions to avoid hanging.
	// (closing the TLS server does not shut down active connections)
	//sm := sessions.GetSessionManager()
	//sm.RemoveAll()
	svc.httpServer.Stop()
}

// writeError is a convenience function that logs and writes an error
// If the reply has an error then write a bad request with the error as payload
func (svc *HttpTransportServer) writeError(w http.ResponseWriter, err error, code int) {
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
func (svc *HttpTransportServer) writeReply(w http.ResponseWriter, data any, err error) {
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
//	config
//	privKey
//	caCert
//	sessionAuth for creating and validating authentication tokens
//	dtwService that handles digital thing requests
func StartHttpTransportServer(host string, port int,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	authenticator transports.IAuthenticator,
	messageHandler transports.ServerMessageHandler,
	cm *connections.ConnectionManager,
) (*HttpTransportServer, error) {

	httpServer, httpRouter := tlsserver.NewTLSServer(
		host, port, serverCert, caCert)

	//wssURL := fmt.Sprintf("wss://%s:%d", config.Host, config.Port)
	svc := HttpTransportServer{
		authenticator:  authenticator,
		messageHandler: messageHandler,

		//ws: wssserver.NewWssTransportServer(cm, requestHandler, wssURL),
		//sse:   ssescserver.NewSseScTransportServer(cm),
		//ssesc: ssescserver.NewSseScTransportServer(cm),

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
