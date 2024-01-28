package service

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hub/bindings/hiveoview/src"
	"github.com/hiveot/hub/bindings/hiveoview/src/hiveoviewapi"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/dashboard"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/directory"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/login"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/status"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/thing"
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/auth/authclient"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/vocab"
	"log/slog"
	"net/http"
	"os"
	"path"
	"time"
)

// HiveovService operates the html web server.
// It utilizes gin, htmx and TempL for serving html.
// credits go to: https://github.com/marco-souza/gx/blob/main/cmd/server/server.go
type HiveovService struct {
	port         int  // listening port
	dev          bool // development configuration
	shouldUpdate bool
	router       chi.Router
	rootPath     string
	tm           *views.TemplateManager

	// hc hub client of this service.
	// This client's CA and URL is also used to establish client sessions.
	hc *hubclient.HubClient

	// cookie signing
	signingKey *ecdsa.PrivateKey

	// run in debug mode, extra logging and reload templates render
	debug bool
}

// setup the chain of routes used by the service and return the router
// rootPath points to the filesystem containing /static and template files
func (svc *HiveovService) createRoutes(rootPath string) http.Handler {
	var staticFileServer http.Handler

	if rootPath == "" {
		staticFileServer = http.FileServer(
			&StaticFSWrapper{
				FileSystem:   http.FS(src.EmbeddedStatic),
				FixedModTime: time.Now(),
			})
	} else {
		// during development when run from the 'hub' project directory
		staticFileServer = http.FileServer(http.Dir(rootPath))
	}
	router := chi.NewRouter()

	// TODO: add csrf support in posts
	//csrfMiddleware := csrf.Protect(
	//	[]byte("32-byte-long-auth-key"),
	//	csrf.SameSite(csrf.SameSiteStrictMode))

	//-- add the routes and middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	//router.Use(csrfMiddleware)
	router.Use(middleware.Compress(5,
		"text/html", "text/css", "text/javascript", "image/svg+xml"))

	//--- public routes do not require a Hub connection
	router.Group(func(r chi.Router) {
		// serve static files with the startup timestamp so caching works
		//staticFileServer := http.FileServer(
		//	&StaticFSWrapper{
		//		FileSystem:   http.FS(staticFS),
		//		FixedModTime: time.Now(),
		//	})

		// full page routes
		r.Get("/static/*", staticFileServer.ServeHTTP)
		r.Get("/webcomp/*", staticFileServer.ServeHTTP)
		r.Get("/login", login.RenderLogin)
		r.Post("/login", login.PostLogin)
		r.Get("/logout", session.SessionLogout)

		// SSE has its own validation
		r.Get("/sse", session.SseHandler)

	})

	//--- private routes that requires a valid session
	router.Group(func(r chi.Router) {
		// these routes must be authenticated otherwise redirect to /login
		r.Use(session.AddSessionToContext())

		// see also:https://medium.com/gravel-engineering/i-find-it-hard-to-reuse-root-template-in-go-htmx-so-i-made-my-own-little-tools-to-solve-it-df881eed7e4d
		// option1: render full app and trigger htmx request for htmx/thing/agentID/thingID
		//        eg, substitute # for htmx with target #thing
		// option2: add support for hashroutes in chi
		r.Get("/", app.RenderApp)
		r.Get("/dashboard", dashboard.RenderDashboard)
		r.Get("/dashboard/{page}", dashboard.RenderDashboard) // TODO: support multiple pages
		r.Get("/directory", directory.RenderDirectory)
		r.Get("/directory/thing/{a}/{b}", thing.RenderThingDetails)
		r.Get("/status", status.RenderStatus)
		r.Get("/about", app.RenderApp) //about.RenderAbout)

		// fragment routes for loading data into specific hx-target elements
		r.Get("/htmx/connectStatus.html", app.RenderConnectStatus)
		r.Get("/htmx/dashboard", dashboard.RenderDashboard)        // TODO: support multiple pages
		r.Get("/htmx/dashboard/{page}", dashboard.RenderDashboard) // TODO: support multiple pages
		r.Get("/htmx/directory", directory.RenderDirectory)
	})

	return router
}

// CreateHiveoviewTD creates a new Thing TD document describing the service capability
func (svc *HiveovService) CreateHiveoviewTD() *things.TD {
	title := "Web Server"
	deviceType := vocab.DeviceTypeService
	td := things.NewTD(hiveoviewapi.HiveoviewServiceCap, title, deviceType)
	// TODO: add properties: uptime, max nr clients

	td.AddEvent("activeSessions", "", "Nr Sessions", "Number of currently active sessions",
		&things.DataSchema{
			//AtType: vocab.SessionCount,
			Type: vocab.WoTDataTypeInteger,
		})

	return td
}

// Start the web server and publish the service's own TD.
func (svc *HiveovService) Start(hc *hubclient.HubClient) error {
	slog.Warn("Starting HiveovService", "clientID", hc.ClientID())
	svc.hc = hc

	// publish a TD for each service capability and set allowable roles
	// in this case only a management capability is published
	myProfile := authclient.NewProfileClient(svc.hc)
	err := myProfile.SetServicePermissions(hiveoviewapi.HiveoviewServiceCap, []string{
		authapi.ClientRoleAdmin,
		authapi.ClientRoleService})
	if err != nil {
		slog.Error("failed to set the hiveoview service permissions", "err", err.Error())
	}

	myTD := svc.CreateHiveoviewTD()
	myTDJSON, _ := json.Marshal(myTD)
	err = svc.hc.PubEvent(hiveoviewapi.HiveoviewServiceCap, vocab.EventNameTD, myTDJSON)
	if err != nil {
		slog.Error("failed to publish the hiveoview service TD", "err", err.Error())
	}

	// Setup the handling of incoming web sessions
	sm := session.GetSessionManager()
	connStat := hc.GetStatus()
	sm.Init(connStat.HubURL, connStat.Core, svc.signingKey, connStat.CaCert)

	// parse the templates
	svc.tm.ParseAllTemplates()

	// add the routes
	router := svc.createRoutes(svc.rootPath)

	// TODO: change into TLS using a signed server certificate
	addr := fmt.Sprintf(":%d", svc.port)
	go func() {
		err = http.ListenAndServe(addr, router)
		if err != nil {
			// TODO: close gracefully
			slog.Error("Failed starting server", "err", err)
			// service must exit on close
			time.Sleep(time.Second)
			os.Exit(0)
		}
	}()
	return nil
}

func (svc *HiveovService) Stop() {
	// TODO: send event the service has stopped
	svc.hc.Disconnect()
	//svc.router.Stop()

	//if err != nil {
	//	slog.Error("Stop error", "err", err)
	//}
}

// NewHiveovService creates a new service instance that serves the
// content from a http.FileSystem.
//
// rootPath is the root directory when serving files from the filesystem.
// This must contain static/, views/ and webc/ directories.
// If empty, the embedded filesystem is used.
//
// serverPort is the port of the web server will listen on
// debug to enable debugging output
func NewHiveovService(serverPort int, debug bool, signingKey *ecdsa.PrivateKey, rootPath string) *HiveovService {
	templatePath := rootPath
	if rootPath != "" {
		templatePath = path.Join(rootPath, "views")
	}
	tm := views.InitTemplateManager(templatePath)
	svc := HiveovService{
		port:         serverPort,
		shouldUpdate: true,
		debug:        debug,
		signingKey:   signingKey,
		rootPath:     rootPath,
		tm:           tm,
	}
	return &svc
}
