package service

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hub/bindings/hiveoview/assets"
	"github.com/hiveot/hub/bindings/hiveoview/assets/views/about"
	"github.com/hiveot/hub/bindings/hiveoview/assets/views/app"
	"github.com/hiveot/hub/bindings/hiveoview/assets/views/dashboard"
	"github.com/hiveot/hub/bindings/hiveoview/assets/views/login"
	"github.com/hiveot/hub/bindings/hiveoview/assets/views/thingsview"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"io/fs"
	"log/slog"
	"net/http"
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

	// run in debug mode, extra logging and reload templates render
	debug bool
}

// setup the chain of routes used by the service and return the router
// fs is the filesystem containing /static and template files
func (svc *HiveovService) createRoutes(staticFS fs.FS) chi.Router {

	router := chi.NewRouter()

	//-- add the routes and middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	//--- public routes
	router.Group(func(r chi.Router) {
		staticFileServer := http.FileServer(
			&StaticFSWrapper{
				FileSystem:   http.FS(staticFS),
				FixedModTime: time.Now(),
			})

		// full page routes
		r.Get("/static/*", staticFileServer.ServeHTTP)
		r.Get("/login", login.RenderLogin)
		r.Post("/login", login.PostLogin)
		r.Get("/about", about.RenderAbout)
		//r.Get("/app/about", app.RenderApp(t, "about.html"))
		r.Get("/app/{pageName}", app.RenderApp)
		r.Get("/", app.RenderApp)

		// fragment routes
		r.Get("/htmx/connectStatus.html", login.RenderConnectStatus)
		r.Get("/htmx/about.html", about.RenderAbout)
		r.Get("/htmx/counter.html", app.RenderCounter)
	})

	//--- private routes that requires a valid session
	router.Group(func(r chi.Router) {
		sm := session.GetSessionManager()
		r.Use(session.AuthSession(sm))

		r.Get("/dashboard", dashboard.RenderDashboard)
		r.Get("/things", thingsview.RenderThings)

	})
	return router
}

func (svc *HiveovService) Start() {

	// parse all templates for use in the routes
	// Would like to do ParseFSRecursive("*.html") ... but it doesn't exist
	//templates, err := template.ParseFS(views.EmbeddedViews,
	//	"*.html", "login/*.html", "app/*.html", "about/*.html")
	//templates, err := assets.ParseTemplates()
	//if err != nil {
	//	slog.Error("Parsing templates failed", "err", err)
	//	panic("failed parsing templates")
	//}

	// setup static resources
	// add the routes
	router := svc.createRoutes(assets.EmbeddedStatic)

	// TODO: change into TLS using a signed server certificate
	// TODO: set Cache-Control header
	//err = router.Run(fmt.Sprintf(":%d", svc.port))
	addr := fmt.Sprintf(":%d", svc.port)
	err := http.ListenAndServe(addr, router)
	if err != nil {
		slog.Error("Failed starting server", "err", err)
		panic("failed starting server")

	}
}

func (svc *HiveovService) Stop() {
	//err := svc.ginRouter.Shutdown(time.Second * 3)
	//if err != nil {
	//	slog.Error("Stop error", "err", err)
	//}
}

// NewHiveovService creates a new service instance that serves the
// content from a http.FileSystem.
//
// For a live filesystem use: http.Dir("path/to/files")
//
// For an embedded filesystem use one of:
//
//	embed:       //go:embed path/to/folder
//	(go 1.16+)   var contentFS embed.FS
//	pkger: pkger.Dir("/views")
//	packr: packr.New("Templates","/views")
//	rice:  rice.MustFindBox("views").HTTPBox()
//
// serverPort is the port of the web server will listen on
// debug to enable debugging output
func NewHiveovService(serverPort int, debug bool) *HiveovService {
	svc := HiveovService{
		port:         serverPort,
		shouldUpdate: true,
		debug:        debug,
	}
	return &svc
}
