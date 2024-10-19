package httptransport

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hub/api/go/vocab"
	"net/http"
)

// HttpRouter contains the method to setup the HTTP binding routes

// setup the chain of routes used by the service and return the router
func (svc *HttpBinding) createRoutes(router chi.Router) http.Handler {

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
		//r.Post(PostLoginPath, svc.HandleLogin)
		svc.AddPostOp(r, vocab.HTOpLogin, false,
			"/authn/login", svc.HandleLogin)
	})

	//--- private routes that requires authentication (as published in the TD)
	router.Group(func(r chi.Router) {
		r.Use(middleware.Compress(5,
			"text/html", "text/css", "text/javascript", "image/svg+xml"))

		// client sessions authenticate the sender
		r.Use(svc.AddSessionFromToken(svc.authenticator))

		//- direct methods for digital twins
		svc.AddGetOp(r, vocab.WotOpReadAllProperties, true,
			"/digitwin/properties/{thingID}", svc.HandleReadAllProperties)
		svc.AddGetOp(r, vocab.WoTOpReadProperty, true,
			"/digitwin/properties/{thingID}/{name}", svc.HandleReadProperty)
		svc.AddPostOp(r, vocab.WoTOpWriteProperty, true,
			"/digitwin/properties/{thingID}/{name}", svc.HandleWriteProperty)

		svc.AddGetOp(r, "readallevents", true,
			"/digitwin/events/{thingID}", svc.HandleReadAllEvents)
		svc.AddGetOp(r, "readevent", true,
			"/digitwin/events/{thingID}/{eventID}", svc.HandleReadEvent)

		svc.AddGetOp(r, vocab.WotOpQueryAllActions, true,
			"/digitwin/actions/{thingID}", svc.HandleQueryAllActions)
		svc.AddGetOp(r, vocab.WotOpQueryAction, true,
			"/digitwin/actions/{thingID}/{name}", svc.HandleQueryAction)
		svc.AddPostOp(r, vocab.WotOpInvokeAction, true,
			"/digitwin/actions/{thingID}/{name}", svc.HandleActionRequest)

		// sse-sc subprotocol routes
		svc.AddGetOp(r, "connect", true,
			"/ssesc", svc.ssesc.HandleConnect)
		svc.AddPostOp(r, vocab.WotOpObserveAllProperties, true,
			"/ssesc/digitwin/observe/{thingID}", svc.ssesc.HandleObserveAllProperties)
		svc.AddPostOp(r, vocab.WotOpSubscribeAllEvents, true,
			"/ssesc/digitwin/subscribe/{thingID}", svc.ssesc.HandleSubscribeAllEvents)
		svc.AddPostOp(r, vocab.WotOpSubscribeEvent, true,
			"/ssesc/digitwin/subscribe/{thingID}/{name}", svc.ssesc.HandleSubscribeEvent)
		svc.AddPostOp(r, vocab.WoTOpObserveProperty, true,
			"/ssesc/digitwin/observe/{thingID}/{name}", svc.ssesc.HandleObserveProperty)
		svc.AddPostOp(r, vocab.WotOpUnobserveAllProperties, true,
			"/ssesc/digitwin/observe/{thingID}", svc.ssesc.HandleUnobserveAllProperties)
		svc.AddPostOp(r, vocab.WoTOpUnobserveProperty, true,
			"/ssesc/digitwin/unobserve/{thingID}/{name}", svc.ssesc.HandleUnobserveProperty)
		svc.AddPostOp(r, vocab.WotOpUnsubscribeAllEvents, true,
			"/ssesc/digitwin/unsubscribe/{thingID}", svc.ssesc.HandleUnsubscribeAllEvents)
		svc.AddPostOp(r, vocab.WotOpUnsubscribeEvent, true,
			"/ssesc/digitwin/unsubscribe/{thingID}/{name}", svc.ssesc.HandleUnsubscribeEvent)

		// ws subprotocol routes
		//svc.AddPostOp(r, vocab.WotOpObserveAllProperties, true,
		//	"/ws/digitwin/observe/{thingID}", svc.ws.HandleObserveAllProperties)
		//svc.AddPostOp(r, vocab.WoTOpObserveProperty, true,
		//	"/ws/digitwin/observe/{thingID}/{name}", svc.ws.HandleObserveProperty)
		//
		//// sse subprotocol routes
		//svc.AddPostOp(r, vocab.WotOpSubscribeAllEvents, true,
		//	"/sse/digitwin/subscribe/{thingID}", svc.sse.HandleSubscribeAllEvents)
		//svc.AddPostOp(r, vocab.WotOpObserveAllProperties, true,
		//	"/sse/digitwin/observe/{thingID}", svc.sse.HandleObserveAllProperties)

		// digitwin directory actions. Are these operations or actions?
		svc.AddGetOp(r, vocab.HTOpReadThing, false,
			"/digitwin/directory/{thingID}", svc.HandleReadThing)
		svc.AddGetOp(r, vocab.HTOpReadAllThings, false,
			"/digitwin/directory", svc.HandleReadAllThings) // query params: offset,limit

		// handlers for other services. Operations to invoke actions.
		// TODO: these probably belong with the digitwin service TD

		// authn/authz service actions
		svc.AddPostOp(r, vocab.HTOpRefresh, false,
			"/authn/refresh", svc.HandleRefresh)
		svc.AddPostOp(r, vocab.HTOpLogout, false,
			"/authn/logout", svc.HandleLogout)

		// handlers for requests by agents
		// TODO: These should be included in the digitwin TD forms
		svc.AddPostOp(r, vocab.HTOpUpdateThing, false,
			"/agent/tdd/{thingID}", svc.HandleUpdateThing)
		svc.AddPostOp(r, vocab.HTOpPublishEvent, false,
			"/agent/event/{thingID}/{name}", svc.HandlePublishEvent)
		svc.AddPostOp(r, vocab.HTOpUpdateProperty, false,
			"/agent/property/{thingID}/{name}", svc.HandleUpdateProperty)
		svc.AddPostOp(r, "updateMultipleProperties", false,
			"/agent/properties/{thingID}", svc.HandleUpdateProperty)
		svc.AddPostOp(r, vocab.HTOpDelivery, false,
			"/agent/progress", svc.HandleActionProgress)

	})

	return router
}
