package service

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/template/html/v2"
	"log/slog"
	"net/http"
	"time"
)

// Embed the static and views directories
//
////go:embed static/* views/*
//var embedContent embed.FS

// HiveovService operates the html web server.
// It utilizes fiber, htmx and TempL for serving html.
// credits go to: https://github.com/marco-souza/gx/blob/main/cmd/server/server.go
type HiveovService struct {
	port         int  // listening port
	dev          bool // development configuration
	shouldUpdate bool
	fiberApp     *fiber.App
	contentFS    http.FileSystem
	// run in debug mode, extra logging and reload templates render
	debug bool
}

func (svc *HiveovService) hotReloadHandler(c *fiber.Ctx) error {
	defer func() { svc.shouldUpdate = false }()
	return c.SendString(fmt.Sprintf("%t", svc.shouldUpdate))
}

//func (svc *HiveovService) setupRoutes() {
//	if svc.dev {
//		log.Println("setup hot reload")
//		svc.fiberApp.Get("/hot-reload", svc.hotReloadHandler)
//	}
//
//	log.Println("setup static resources")
//	staticDir := path.Join(svc.templateDir, "static")
//	svc.fiberApp.Static(staticDir, "./static", fiber.Static{
//		Compress:      true,
//		ByteRange:     true,
//		Browse:        true,
//		CacheDuration: 10 * time.Second,
//		MaxAge:        3600,
//	})
//
//	//routes.SetupRoutes(svc.fiberApp)
//}

func (svc *HiveovService) Start() {

	engine := html.NewFileSystem(svc.contentFS, ".html")

	// reload the templates on each render, for debugging.
	engine.Reload(svc.debug)
	engine.Debug(svc.debug)

	svc.fiberApp = fiber.New(fiber.Config{
		Views: engine,
		//ViewsLayout: "layouts/main",
	})

	// service static files
	svc.fiberApp.Use("/static", filesystem.New(filesystem.Config{
		Root:       svc.contentFS,
		PathPrefix: "static", // location of static in contentFS
		Browse:     true,
	}))

	addr := fmt.Sprintf("%s:%d", "localhost", svc.port)

	svc.fiberApp.Get("/", func(c *fiber.Ctx) error {
		err := c.Render("app/app",
			fiber.Map{
				"Title":       "Hello world",
				"theme":       "dark",
				"theme_icon":  "dark_mode",
				"pages":       []string{"page1", "page2"},
				"conn_icon":   "link", // "link_off"
				"conn_status": "Connected",
			}, "index")
		return err
	})

	// FIXME: change into TLS using a signed server certificate
	err := svc.fiberApp.Listen(addr)
	if err != nil {
		slog.Error("Failed starting server", "err", err)
		panic("failed starting server")

	}
}

func (svc *HiveovService) Stop() {
	err := svc.fiberApp.ShutdownWithTimeout(time.Second * 3)
	if err != nil {
		slog.Error("Stop error", "err", err)
	}
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
func NewHiveovService(port int, contentFS http.FileSystem, debug bool) *HiveovService {
	svc := HiveovService{
		port:         port,
		shouldUpdate: true,
		contentFS:    contentFS,
		debug:        debug,
	}
	return &svc
}
