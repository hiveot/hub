package httpbasic

import (
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hub/messaging"
	_ "github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/tputils/tlsserver"
	jsoniter "github.com/json-iterator/go"

	"io"
	"log/slog"
	"net/http"
)

// handleAffordanceOperation converts the http request to a request message and pass it to the registered request handler
// this read request params for {operation}, {thingID} and {name}
func (srv *HttpBasicServer) handleAffordanceOperation(w http.ResponseWriter, r *http.Request) {
	var output any
	var handled bool

	// 1. Decode the request message
	rp, err := GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// Use the authenticated clientID as the sender
	var input any
	err = rp.Unmarshal(&input)
	req := messaging.NewRequestMessage(rp.Op, rp.ThingID, rp.Name, input, "")
	req.SenderID = rp.ClientID
	req.CorrelationID = rp.CorrelationID

	// filter on allowed operations
	if !slices.Contains(HttpKnownOperations, req.Operation) {
		slog.Warn("Unsupported operation for http-basic",
			"method", r.Method, "URL", r.URL.String(),
			"operation", req.Operation, "thingID", req.ThingID, "name", req.Name, "clientID", req.SenderID)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// http-basic is connectionless so connectionID is not used here
	//connectionID := rp.ConnectionID

	// pass request it on to the application
	resp := srv.serverRequestHandler(req, nil)
	if resp == nil {
		// no response available
	} else {
		output = resp.Value
		if resp.Error != nil {
			err = resp.Error.AsError()
		}
	}

	// 4. Return the response
	tlsserver.WriteReply(w, handled, output, err)
}

// handleThingOperation converts the http request to a request message and pass it to the registered request handler
func (srv *HttpBasicServer) handleThingOperation(w http.ResponseWriter, r *http.Request) {
	// same same
	srv.handleAffordanceOperation(w, r)
}

// HandleLogin handles a login request and returns an auth token.
//
// Body contains {"login":name, "password":pass} format
// This is the only unprotected route supported.
// This uses the configured session authenticator.
func (srv *HttpBasicServer) HandleLogin(w http.ResponseWriter, r *http.Request) {
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
		reply, err = srv.authenticator.Login(clientID, password)
		slog.Info("HandleLogin", "clientID", clientID)
	}
	if err != nil {
		slog.Warn("HandleLogin failed:", "err", err.Error())
		tlsserver.WriteError(w, err, http.StatusUnauthorized)
		return
	}
	// TODO: set client session cookie for browser clients
	//srv.sessionManager.SetSessionCookie(cs.sessionID,token)
	tlsserver.WriteReply(w, true, reply, nil)
}

// HandleAuthRefresh refreshes the auth token using the session authenticator.
// The session authenticator is that of the authn service. This allows testing with a dummy
// authenticator without having to run the authn service.
func (srv *HttpBasicServer) HandleAuthRefresh(w http.ResponseWriter, r *http.Request) {
	var newToken string
	var oldToken string
	rp, err := GetRequestParams(r)

	if err == nil {
		jsoniter.Unmarshal(rp.Payload, &oldToken)
		slog.Info("HandleAuthRefresh", "clientID", rp.ClientID)
		newToken, err = srv.authenticator.RefreshToken(rp.ClientID, oldToken)
	}
	if err != nil {
		slog.Warn("HandleAuthRefresh failed:", "err", err.Error())
		tlsserver.WriteError(w, err, 0)
		return
	}
	tlsserver.WriteReply(w, true, newToken, nil)
}

// HandleLogout ends the session and closes all client connections
func (srv *HttpBasicServer) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// use the authenticator
	rp, err := GetRequestParams(r)
	if err == nil {
		slog.Info("HandleLogout", "clientID", rp.ClientID)
		srv.authenticator.Logout(rp.ClientID)
	}
	tlsserver.WriteReply(w, true, nil, err)
}

// HandlePing with http handler returns a pong response
func (srv *HttpBasicServer) HandlePing(w http.ResponseWriter, r *http.Request) {
	// simply return a pong message
	tlsserver.WriteReply(w, true, "pong", nil)
}

// setupRouting creates the middleware chain for handling requests, including
// recoverer, compression and token verification for protected routes.
//
// This includes the unprotected routes for login and ping (for now)
// This includes the protected routes for refresh and logout. (for now)
// Everything else should be added by the sub-protocols.
//
// Routes are added by (sub)protocols such as http-basic, sse and wss.
func (srv *HttpBasicServer) setupRouting(router chi.Router) http.Handler {

	// TODO: add csrf support in posts
	//csrfMiddleware := csrf.Protect(
	//	[]byte("32-byte-long-auth-key"),
	//	csrf.SameSite(csrf.SameSiteStrictMode))

	//-- add the middleware before routes
	router.Use(middleware.Recoverer)
	//router.Use(middleware.Logger) // todo: proper logging strategy
	//router.Use(csrfMiddleware)
	router.Use(middleware.Compress(5,
		"text/html", "text/css", "text/javascript", "image/svg+xml"))

	//--- public routes do not require an authenticated session
	router.Group(func(r chi.Router) {
		// sub-protocols can add public routes
		srv.publicRoutes = r

		//r.Get("/static/*", staticFileServer.ServeHTTP)
		// build-in REST API for easy login to obtain a token

		// register authentication endpoints
		// FIXME: determine how WoT wants auth endpoints to be published
		r.Post(HttpPostLoginPath, srv.HandleLogin)
		r.Get(HttpGetPingPath, srv.HandlePing)
	})

	//--- private routes that requires authentication (as published in the TD)
	router.Group(func(r chi.Router) {

		// client sessions authenticate the sender
		r.Use(AddSessionFromToken(srv.authenticator))

		// sub-protocols can add protected routes
		srv.protectedRoutes = r

		// register generic handlers for operations on Thing and affordance level
		// these endpoints are published in the forms of each TD. See also AddTDForms.
		r.HandleFunc(HttpBasicAffordanceOperationPath, srv.handleAffordanceOperation)
		r.HandleFunc(HttpBasicThingOperationPath, srv.handleThingOperation)

		// http supported authentication endpoints
		r.Post(HttpPostRefreshPath, srv.HandleAuthRefresh)
		r.Post(HttpPostLogoutPath, srv.HandleLogout)
	})
	return router
}
