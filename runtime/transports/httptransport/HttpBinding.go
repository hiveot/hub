package httptransport

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/lib/tlsserver"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/connections"
	"github.com/hiveot/hub/runtime/hubrouter"
	"github.com/hiveot/hub/runtime/transports/httptransport/subprotocols"
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

// HttpBinding is the Hub transport binding for HTTPS
// This wraps the library's https server and add routes and middleware for use in the binding
type HttpBinding struct {
	// port and path configuration
	config *HttpTransportConfig

	// TLS server and router
	httpServer *tlsserver.TLSServer
	router     *chi.Mux

	// subprotocol bindings
	sse   *subprotocols.SseBinding
	ssesc *subprotocols.SseScBinding
	ws    *subprotocols.WsBinding

	// authenticator for logging in and validating session tokens
	authenticator api.IAuthenticator

	// Thing level operations
	operations []HttpOperation

	// routing of action, event and property requests
	hubRouter hubrouter.IHubRouter

	// connection manager for adding/removing binding connections
	cm *connections.ConnectionManager
}

// AddGetOp adds protocol binding operation with a URL and handler
//
// This is used to add Forms to the digitwin TDs
func (svc *HttpBinding) AddGetOp(r chi.Router,
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
func (svc *HttpBinding) AddPostOp(r chi.Router,
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

// GetConnectionByCID returns the client connection for sending messages to a client
func (svc *HttpBinding) GetConnectionByCID(cid string) connections.IClientConnection {
	return svc.cm.GetConnectionByCID(cid)
}

// GetProtocolInfo returns info on the protocol supported by this binding
func (svc *HttpBinding) GetProtocolInfo() api.ProtocolInfo {
	hostName := svc.config.Host
	if hostName == "" {
		hostName = "localhost"
	}
	baseURL := fmt.Sprintf("https://%s:%d", hostName, svc.config.Port)
	inf := api.ProtocolInfo{
		BaseURL:   baseURL,
		Schema:    "https",
		Transport: "https",
	}
	return inf
}

// Stop the https server
func (svc *HttpBinding) Stop() {
	slog.Info("Stopping HttpBinding")

	// Shutdown remaining sessions to avoid hanging.
	// (closing the TLS server does not shut down active connections)
	//sm := sessions.GetSessionManager()
	//sm.RemoveAll()
	svc.httpServer.Stop()
}

// writeError is a convenience function that logs and writes an error
// If the reply has an error then write a bad request with the error as payload
func (svc *HttpBinding) writeError(w http.ResponseWriter, err error, code int) {
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
func (svc *HttpBinding) writeReply(w http.ResponseWriter, data any, err error) {
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

// StartHttpTransport creates and starts a new instance of the HTTPS Server
// with JWT authentication and SSE/SSE-SC/WS sub-protocol bindings.
//
// Call stop to end the transport server.
//
//	config
//	privKey
//	caCert
//	sessionAuth for creating and validating authentication tokens
//	dtwService that handles digital thing requests
func StartHttpTransport(config *HttpTransportConfig,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	authenticator api.IAuthenticator,
	hubRouter hubrouter.IHubRouter,
	cm *connections.ConnectionManager,
) (*HttpBinding, error) {

	httpServer, router := tlsserver.NewTLSServer(
		config.Host, config.Port, serverCert, caCert)

	svc := HttpBinding{
		authenticator: authenticator,
		config:        config,
		// subprotocol bindings need session info
		ws:         subprotocols.NewWsBinding(cm),
		sse:        subprotocols.NewSseBinding(cm),
		ssesc:      subprotocols.NewSseScBinding(cm),
		httpServer: httpServer,
		router:     router,
		hubRouter:  hubRouter,
		cm:         cm,
	}

	svc.createRoutes(svc.router)
	err := svc.httpServer.Start()
	return &svc, err
}
