package httptransport

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/lib/tlsserver"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/hubrouter"
	"github.com/hiveot/hub/runtime/transports/httptransport/subprotocols"
	sessions2 "github.com/hiveot/hub/runtime/transports/sessions"
	"io"
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
	sm           *sessions2.SessionManager
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

	// reading of digital twin info
	dtwService *service.DigitwinService

	// session manager for adding/removing sessions (login,logout)
	sm *sessions2.SessionManager

	// connection manager for adding/removing binding connections
	cm *sessions2.ConnectionManager
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
func (svc *HttpBinding) GetConnectionByCID(cid string) sessions2.IClientConnection {
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

// GetRequestParams reads the client session, URL parameters and body payload from the request.
//
// The session context is set by the http middleware. If the session is not available then
// this returns an error. Note that the session middleware handler will block any request
// that requires a session.
//
// This protocol binding reads two variables, {thingID} and {name} in the path.
//
//	{thingID} is the agent or digital twin thing ID
//	{name} is the property, event or action name. '+' means 'all'
func (svc *HttpBinding) GetRequestParams(r *http.Request) (clientID string, thingID string, name string, body []byte, err error) {

	// get the required client session of this agent
	sessID, clientID, err := subprotocols.GetSessionIdFromContext(r)
	_ = sessID
	if err != nil {
		// This is an internal error. The middleware session handler would have blocked
		// a request that required a session before getting here.
		slog.Error(err.Error())
		return "", "", "", nil, err
	}

	// build a message from the URL and payload
	// URLParam names are defined by the path variables set in the router.
	thingID = chi.URLParam(r, "thingID")
	name = chi.URLParam(r, "name")
	body, _ = io.ReadAll(r.Body)

	return clientID, thingID, name, body, err
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

// writeReply is a convenience function that serializes the data and writes it as a response.
func (svc *HttpBinding) writeReply(w http.ResponseWriter, data any) {
	if data != nil {
		// If no header is written then w.Write writes a StatusOK
		payload, _ := json.Marshal(data)
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
	dtwService *service.DigitwinService,
	cm *sessions2.ConnectionManager,
	sm *sessions2.SessionManager,
) (*HttpBinding, error) {

	httpServer, router := tlsserver.NewTLSServer(
		config.Host, config.Port, serverCert, caCert)

	svc := HttpBinding{
		authenticator: authenticator,
		config:        config,
		// subprotocol bindings need session info
		ws:         subprotocols.NewWsBinding(cm, sm),
		sse:        subprotocols.NewSseBinding(cm, sm),
		ssesc:      subprotocols.NewSseScBinding(cm, sm),
		httpServer: httpServer,
		router:     router,
		hubRouter:  hubRouter,
		dtwService: dtwService,
		cm:         cm,
		sm:         sm,
	}

	svc.createRoutes(svc.router)
	err := svc.httpServer.Start()
	return &svc, err
}
