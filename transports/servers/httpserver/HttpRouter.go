package httpserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hub/wot"
	"net/http"
)

const (

	// HTTP protoocol constants
	// StatusHeader contains the result of the request, eg Pending, Completed or Failed
	StatusHeader = "status"
	// RequestIDHeader for transports that support headers can include a message-ID
	RequestIDHeader = "request-id"
	// ConnectionIDHeader identifies the client's connection in case of multiple
	// connections from the same client.
	ConnectionIDHeader = "connection-id"
	// DataSchemaHeader to indicate which  'additionalresults' dataschema being returned.
	DataSchemaHeader = "dataschema"

	// HTTP Paths for auth. - FIXME MOVE TO HTTP implementation
	// THIS WILL BE REMOVED AFTER THE PROTOCOL BINDING PUBLISHES THESE IN THE TDD.
	// The hub client will need the TD (ConsumedThing) to determine the paths.
	HttpPostLoginPath   = "/authn/login"
	HttpPostLogoutPath  = "/authn/logout"
	HttpPostRefreshPath = "/authn/refresh"
	HttpGetDigitwinPath = "/digitwin/{operation}/{thingID}/{name}"

	// paths for HTTP subprotocols
	DefaultWSSPath   = "/wss"
	DefaultSSEPath   = "/sse"
	DefaultSSESCPath = "/ssesc"

	// Generic form href that maps to all operations for the http client, using URI variables
	// Generic HiveOT HTTP urls when Forms are not available. The payload is a
	// corresponding standardized message.
	HiveOTPostNotificationHRef = "/hiveot/notification"
	HiveOTPostRequestHRef      = "/hiveot/request"
	HiveOTPostResponseHRef     = "/hiveot/response"
)

// HttpRouter contains the method to setup the HTTP binding routes

// setup the chain of routes used by the service and return the router
// this also sets the routes for the sub-protocol handlers (sse-sc and wss)
func (svc *HttpTransportServer) createRoutes(router chi.Router) http.Handler {

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
	//router.Use(middleware.Recoverer)
	//router.Use(csrfMiddleware)
	//router.Use(middleware.Compress(5,
	//	"text/html", "text/css", "text/javascript", "image/svg+xml"))

	//--- public routes do not require an authenticated session
	router.Group(func(r chi.Router) {
		r.Use(middleware.Compress(5,
			"text/html", "text/css", "text/javascript", "image/svg+xml"))

		//r.Get("/static/*", staticFileServer.ServeHTTP)
		// build-in REST API for easy login to obtain a token
		svc.AddOps(r, []string{wot.HTOpLogin}, http.MethodPost, HttpPostLoginPath, svc.HandleLogin)
	})

	//--- private routes that requires authentication (as published in the TD)
	// general format for digital twins: /digitwin/{operation}/{thingID}/{name}
	router.Group(func(r chi.Router) {
		r.Use(middleware.Compress(5,
			"text/html", "text/css", "text/javascript", "image/svg+xml"))

		// client sessions authenticate the sender
		r.Use(AddSessionFromToken(svc.authenticator))

		// the following are protected http routes
		svc.protectedRoutes = r
		svc.AddOps(r, []string{wot.HTOpPing}, http.MethodGet, "/ping", svc.HandlePing)

		//- direct methods for digital twins
		svc.AddOps(r, []string{
			wot.HTOpReadAllEvents,
			wot.HTOpReadAllTDs,
			wot.OpReadAllProperties,
			wot.OpQueryAllActions},
			http.MethodGet, "/digitwin/{operation}/{thingID}", svc.HandleRequestMessage)
		svc.AddOps(r, []string{
			wot.HTOpReadEvent,
			wot.HTOpReadTD,
			wot.OpReadProperty,
			wot.OpQueryAction},
			http.MethodGet, "/digitwin/{operation}/{thingID}/{name}", svc.HandleRequestMessage)
		svc.AddOps(r, []string{
			wot.OpWriteProperty,
			wot.OpInvokeAction},
			http.MethodPost, "/digitwin/{operation}/{thingID}/{name}", svc.HandleRequestMessage)

		// authn service actions
		svc.AddOps(r, []string{wot.HTOpRefresh},
			http.MethodPost, HttpPostRefreshPath, svc.HandleLoginRefresh)
		svc.AddOps(r, []string{wot.HTOpLogout},
			http.MethodPost, HttpPostLogoutPath, svc.HandleLogout)

		//svc.AddGetOp(r, wot.OpReadAllProperties,
		//	"/digitwin/readallproperties/{thingID}", svc.HandleReadAllProperties)
		//svc.AddGetOp(r, wot.OpReadProperty,
		//	"/digitwin/readproperty/{thingID}/{name}", svc.HandleReadProperty)
		//svc.AddPostOp(r, wot.OpWriteProperty,
		//	"/digitwin/writeproperty/{thingID}/{name}", svc.HandleWriteProperty)

		//svc.AddGetOp(r, wot.HTOpReadAllEvents,
		//	"/digitwin/readallevents/{thingID}", svc.HandleReadAllEvents)
		//svc.AddGetOp(r, wot.HTOpReadEvent,
		//	"/digitwin/readevent/{thingID}/{eventID}", svc.HandleReadEvent)

		//svc.AddGetOp(r, wot.OpQueryAllActions,
		//	"/digitwin/queryallactions/{thingID}", svc.HandleQueryAllActions)
		//svc.AddGetOp(r, wot.OpQueryAction,
		//	"/digitwin/queryaction/{thingID}/{name}", svc.HandleQueryAction)
		//svc.AddPostOp(r, wot.OpInvokeAction,
		//	"/digitwin/invokeaction/{thingID}/{name}", svc.HandleInvokeAction)

		//if svc.sse != nil {
		//// sse subprotocol routes
		//svc.AddPostOp(r, vocab.OpSubscribeAllEvents, true,
		//	"/sse/digitwin/subscribe/{thingID}", svc.sse.HandleSubscribeAllEvents)
		//svc.AddPostOp(r, vocab.OpObserveAllProperties, true,
		//	"/sse/digitwin/observe/{thingID}", svc.sse.HandleObserveAllProperties)
		//}
		// digitwin directory actions. These are just for convenience as actions are normally used
		//svc.AddGetOp(r, wot.HTOpReadTD,
		//	"/digitwin/readtd/{thingID}", svc.HandleReadTD)
		//svc.AddGetOp(r, wot.HTOpReadAllTDs,
		//	"/digitwin/readalltds", svc.HandleReadAllTDs) // query params: offset,limit

		// handlers for other services. Operations to invoke actions.
		// TODO: these probably belong with the digitwin service TD

		// HiveOT messaging API using standardized envelopes. This can be used instead
		// of Forms. Only 3 endpoints are needed. Please try me :)
		//svc.AddPostOp(r, "request", HiveOTPostRequestHRef, svc.HandleHiveotRequest)
		//svc.AddPostOp(r, "response", HiveOTPostResponseHRef, svc.HandleHiveotResponse)
		//svc.AddPostOp(r, "notification", HiveOTPostNotificationHRef, svc.HandleHiveotNotification)
	})

	return router
}
