package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/bindings/hiveoview/src"
	"github.com/hiveot/hub/bindings/hiveoview/src/hiveoviewapi"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/comps"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/dashboard"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/directory"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/history"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/login"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/status"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/thing"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/tlsserver"
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

	// filesystem location of the ./static, webcomp, and ./views template root folder
	rootPath string
	tm       *views.TemplateManager

	// tls server
	serverCert *tls.Certificate
	caCert     *x509.Certificate
	tlsServer  *tlsserver.TLSServer

	// hc hub client of this service.
	// This client's CA and URL is also used to establish client sessions.
	hc hubclient.IHubClient

	// cookie signing
	signingKey *ecdsa.PrivateKey

	// run in debug mode, extra logging and reload templates render
	debug bool
}

// setup the chain of routes used by the service and return the router
// rootPath points to the filesystem containing /static and template files
func (svc *HiveovService) createRoutes(router *chi.Mux, rootPath string) http.Handler {
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

		// sse has its own validation instead of using session context (which reconnects or redirects to /login)
		r.Get("/sse", session.SseHandler)

	})

	//--- private routes that requires a valid session
	router.Group(func(r chi.Router) {
		// these routes must be authenticated otherwise redirect to /login
		r.Use(session.AddSessionToContext())

		// see also:https://medium.com/gravel-engineering/i-find-it-hard-to-reuse-root-template-in-go-htmx-so-i-made-my-own-little-tools-to-solve-it-df881eed7e4d
		// these renderer full page or fragments for non hx-boost hx-requests
		r.Get("/", app.RenderApp)
		r.Get("/app/appHead", app.RenderAppHead)
		r.Get("/app/about", app.RenderAbout)

		// dashboard
		r.Get("/dashboard", dashboard.RenderDashboard)
		r.Get("/dashboard/{page}", dashboard.RenderDashboard) // TODO: support multiple pages

		// directory view and thing information
		r.Get("/directory", directory.RenderDirectory)
		r.Get("/thing/confirmDeleteTDDialog/{thingID}", thing.RenderConfirmDeleteTDDialog)
		r.Post("/thing/deleteTD/{thingID}", thing.PostDeleteTD)

		// Thing details view
		r.Get("/thing/details/{thingID}", thing.RenderThingDetails)
		r.Get("/thing/raw/{thingID}", thing.RenderTDRaw)

		// Thing configuration
		r.Get("/thing/configEdit/{thingID}/{key}", thing.RenderConfigEditDialog)
		r.Post("/thing/configEdit/{thingID}/{key}", thing.PostThingConfig)

		// Thing action views
		r.Get("/thing/actionDialog/{thingID}/{key}", thing.RenderActionDialog)
		r.Post("/thing/actionStart/{thingID}/{key}", thing.PostStartAction)

		// History view. Optional query params 'timestamp' and 'duration'
		r.Get("/history/{thingID}/{key}", history.RenderHistoryPage)
		r.Get("/latestValue/{thingID}/{key}", comps.RenderHistoryLatest)

		// Status components
		r.Get("/status", status.RenderStatus)
		r.Get("/app/connectStatus", app.RenderConnectStatus)
		r.Get("/app/progress/{messageID}", thing.RenderProgress)
	})

	return router
}

// Start the web server and publish the service's own TD.
func (svc *HiveovService) Start(hc hubclient.IHubClient) error {
	slog.Info("Starting HiveovService", "clientID", hc.ClientID())
	svc.hc = hc

	// publish a TD for each service capability and set allowable roles
	// in this case only a management capability is published
	err := authz.UserSetPermissions(hc, authz.ThingPermissions{
		AgentID: hc.ClientID(),
		ThingID: hiveoviewapi.HiveoviewServiceID,
		Allow:   []string{authn.ClientRoleAdmin, authn.ClientRoleService},
	})
	if err != nil {
		slog.Error("failed to set the hiveoview service permissions", "err", err.Error())
	}

	// Setup the handling of incoming web sessions
	sm := session.GetSessionManager()
	connStat := hc.GetStatus()
	sm.Init(connStat.HubURL, svc.signingKey, connStat.CaCert)

	// parse the templates
	svc.tm.ParseAllTemplates()

	// Start the TLS server for serving the UI
	if svc.serverCert != nil {
		tlsServer, router := tlsserver.NewTLSServer(
			"", svc.port, svc.serverCert, svc.caCert)

		svc.createRoutes(router, svc.rootPath)
		svc.tlsServer = tlsServer
		err = tlsServer.Start()
	} else {
		// add the routes
		router := chi.NewRouter()
		svc.createRoutes(router, svc.rootPath)

		// For testing and debugging without certificate
		addr := fmt.Sprintf(":%d", svc.port)
		go func() {
			err = http.ListenAndServe(addr, router)
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("Failed starting server", "err", err)
				time.Sleep(time.Second)
				os.Exit(0)
			}
		}()
	}
	// last, publish this service's TD
	_ = svc.PublishServiceTD()

	return nil
}

func (svc *HiveovService) Stop() {
	slog.Info("Stopping HiveovService")
	// TODO: send event the service has stopped
	svc.hc.Disconnect()
	if svc.tlsServer != nil {
		svc.tlsServer.Stop()
	}
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
//	serverPort is the port of the web server will listen on
//	debug to enable debugging output
//	signingKey used to sign cookies. Using nil means that a server restart will invalidate the cookies
//	rootPath containing the templates in the given folder or "" to use the embedded templates
//	serverCert server TLS certificate
//	caCert server CA certificate
func NewHiveovService(serverPort int, debug bool,
	signingKey *ecdsa.PrivateKey, rootPath string,
	serverCert *tls.Certificate, caCert *x509.Certificate,
) *HiveovService {
	templatePath := rootPath
	if rootPath != "" {
		templatePath = path.Join(rootPath, "views")
	}
	if signingKey == nil {
		signingKey, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	}
	tm := views.InitTemplateManager(templatePath)
	svc := HiveovService{
		port:         serverPort,
		shouldUpdate: true,
		debug:        debug,
		signingKey:   signingKey,
		rootPath:     rootPath,
		tm:           tm,
		serverCert:   serverCert,
		caCert:       caCert,
	}
	return &svc
}
