package service

import (
	"embed"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hub/bindings/hiveoview/src/hovmw"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/views"
	"github.com/hiveot/hub/bindings/hiveoview/views/about"
	"github.com/hiveot/hub/bindings/hiveoview/views/app"
	"github.com/hiveot/hub/bindings/hiveoview/views/dashboard"
	"github.com/hiveot/hub/bindings/hiveoview/views/login"
	"github.com/hiveot/hub/bindings/hiveoview/views/thingsview"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
)

// HiveovService operates the html web server.
// It utilizes gin, htmx and TempL for serving html.
// credits go to: https://github.com/marco-souza/gx/blob/main/cmd/server/server.go
type HiveovService struct {
	port         int  // listening port
	dev          bool // development configuration
	shouldUpdate bool
	router       chi.Router

	// session manager for tracking client sessions
	sm *session.SessionManager

	// run in debug mode, extra logging and reload templates render
	debug bool
}

// setup the chain of routes used by the service and return the router
// fs is the filesystem containing /static and template files
func (svc *HiveovService) createRoutes(t *template.Template, staticFS fs.FS) chi.Router {

	router := chi.NewRouter()

	//-- add the routes and middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	//--- public routes
	router.Group(func(r chi.Router) {
		staticFileServer := http.FileServer(http.FS(staticFS))
		r.Handle("/static/*", staticFileServer)
		r.Get("/htmx/connectStatus.html", login.GetConnectStatus(t, svc.sm))
		r.Get("/login", login.GetLogin(t, svc.sm))
		r.Post("/login", login.PostLogin(t, svc.sm))
		r.Get("/about", about.GetAbout(t))
		r.Get("/", app.GetApp(t))
		// initialize the application components
		app.InitCounterComp(t, r)
	})

	//--- private routes that requires a valid session
	router.Group(func(r chi.Router) {
		r.Use(hovmw.AuthSession(svc.sm))

		r.Get("/dashboard", dashboard.GetDashboard(t))
		r.Get("/things", thingsview.GetThings(t))

	})
	return router
}

// ParseTemplates the html templates in the given embedded filesystem
func (svc *HiveovService) ParseTemplates(files embed.FS) (*template.Template, error) {
	t := template.New("")
	err := fs.WalkDir(files, ".", func(parent string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			_, err = t.ParseFS(files, filepath.Join(parent, "*.html"))
			// todo, difference between no files and parse error?
			if err != nil {
				slog.Error("error parsing template", "err", err)
			}
			err = nil
		}
		return err
	})
	return t, err
}

func (svc *HiveovService) Start() {

	// parse all templates for use in the routes
	// Would like to do ParseFSRecursive("*.html") ... but it doesn't exist
	//templates, err := template.ParseFS(views.EmbeddedTemplates,
	//	"*.html", "login/*.html", "app/*.html", "about/*.html")
	templates, err := svc.ParseTemplates(views.EmbeddedTemplates)
	if err != nil {
		slog.Error("Parsing templates failed", "err", err)
		panic("failed parsing templates")
	}

	// setup static resources
	// add the routes
	router := svc.createRoutes(templates, views.EmbeddedStatic)

	// FIXME: change into TLS using a signed server certificate
	//err = router.Run(fmt.Sprintf(":%d", svc.port))
	addr := fmt.Sprintf(":%d", svc.port)
	err = http.ListenAndServe(addr, router)
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
	sm := session.NewSessionManager()

	svc := HiveovService{
		port:         serverPort,
		shouldUpdate: true,
		debug:        debug,
		sm:           sm,
	}
	return &svc
}
