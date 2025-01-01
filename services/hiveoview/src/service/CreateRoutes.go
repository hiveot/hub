package service

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/about"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/services/hiveoview/src/views/dashboard"
	"github.com/hiveot/hub/services/hiveoview/src/views/directory"
	"github.com/hiveot/hub/services/hiveoview/src/views/history"
	"github.com/hiveot/hub/services/hiveoview/src/views/login"
	"github.com/hiveot/hub/services/hiveoview/src/views/status"
	"github.com/hiveot/hub/services/hiveoview/src/views/thing"
	"github.com/hiveot/hub/services/hiveoview/src/views/thingdetails"
	"github.com/hiveot/hub/services/hiveoview/src/views/tile"
	"net/http"
	"time"
)

const WebSsePath = "/websse"

// CreateRoutes sets-up the chain of routes used by the service and return the router
// rootPath points to the filesystem containing /static and template files
func (svc *HiveovService) CreateRoutes(router *chi.Mux, rootPath string) http.Handler {
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
		r.Post(src.UIPostLoginPath, login.PostLoginHandler(svc.sm))
		r.Post(src.UIPostFormLoginPath, login.PostLoginFormHandler(svc.sm))
		r.Get("/logout", session.SessionLogout)
	})

	//--- private routes that requires a valid session
	// NOTE: these paths must match those defined in the render functions
	router.Group(func(r chi.Router) {
		// these routes must be authenticated otherwise redirect to /login
		r.Use(session.AddSessionToContext(svc.sm))

		r.Get(WebSsePath, SseServe)

		// see also:https://medium.com/gravel-engineering/i-find-it-hard-to-reuse-root-template-in-go-htmx-so-i-made-my-own-little-tools-to-solve-it-df881eed7e4d
		// these render full page or fragments for non hx-boost hx-requests
		//r.Get("/", app.RenderApp)
		r.Handle("/", http.RedirectHandler(src.RenderDashboardRootPath, http.StatusPermanentRedirect))
		r.Get(src.RenderAppHeadPath, app.RenderAppHead)
		r.Get(src.RenderAboutPath, about.RenderAboutPage)

		// dashboard endpoints
		r.Get(src.RenderDashboardRootPath, dashboard.RenderDashboardPage)
		r.Get(src.RenderDashboardAddPath, dashboard.RenderConfigDashboard)
		r.Get(src.RenderDashboardPath, dashboard.RenderDashboardPage)
		r.Get(src.RenderDashboardConfirmDeletePath, dashboard.RenderConfirmDeleteDashboard)
		r.Get(src.RenderDashboardEditPath, dashboard.RenderConfigDashboard)
		r.Post(src.PostDashboardLayoutPath, dashboard.SubmitDashboardLayout)
		r.Post(src.PostDashboardConfigPath, dashboard.SubmitConfigDashboard)
		r.Delete(src.DeleteDashboardPath, dashboard.SubmitDeleteDashboard)

		// Directory endpoints
		r.Get(src.RenderThingDirectoryPath, directory.RenderDirectory)
		r.Get(src.RenderThingConfirmDeletePath, directory.RenderConfirmDeleteTD)
		r.Delete(src.DeleteThingPath, directory.SubmitDeleteTD)

		// Thing details view
		r.Get(src.RenderThingDetailsPath, thingdetails.RenderThingDetails)
		r.Get(src.RenderThingRawPath, thing.RenderThingRaw)

		// Performing Thing Actions and Configuration
		r.Get(src.RenderActionRequestPath, thing.RenderActionRequest)
		r.Get(src.RenderActionStatusPath, thing.RenderActionStatus)
		r.Get(src.RenderThingPropertyEditPath, thing.RenderEditProperty)
		r.Post(src.PostActionRequestPath, thing.SubmitActionRequest)
		r.Post(src.PostThingPropertyEditPath, thing.SubmitProperty)

		// Dashboard tiles
		r.Get(src.RenderTileAddPath, tile.RenderEditTile)
		r.Get(src.RenderTilePath, tile.RenderTile)
		r.Get(src.RenderTileConfirmDeletePath, tile.RenderConfirmDeleteTile)
		r.Get(src.RenderTileEditPath, tile.RenderEditTile)
		r.Get(src.RenderTileSelectSourcesPath, tile.RenderSelectSources)
		r.Get("/tile/{thingID}/{name}/sourceRow", tile.RenderTileSourceRow)
		r.Post(src.PostTileEditPath, tile.SubmitEditTile)
		r.Delete(src.PostTileDeletePath, tile.SubmitDeleteTile)

		// history. Optional query params 'timestamp' and 'duration'
		r.Get(src.RenderHistoryPagePath, history.RenderHistoryPage)
		r.Get(src.RenderHistoryLatestValueRowPath, history.RenderLatestValueRow)

		// Status components
		r.Get(src.RenderStatusPath, status.RenderStatus)
		r.Get(src.RenderConnectionStatusPath, app.RenderConnectStatus)
	})

	return router
}
