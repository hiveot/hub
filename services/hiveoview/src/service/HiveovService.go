package service

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils/tlsserver"
	"log/slog"
	"net/http"
	"net/url"
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
	serverURL  string
	// a web session per connections
	sm *session.WebSessionManager

	// hc hub client of this service.
	// This client's CA and URL is also used to establish client sessions.
	hc transports.IClientConnection

	// cookie signing
	signingKey ed25519.PrivateKey

	// run in debug mode, extra logging and reload templates render
	debug bool
	// don't use the state store for persistence
	noState bool
}

func (svc *HiveovService) GetServerURL() string {
	return svc.serverURL
}

// GetSM returns the web session manager
// Intended for testing.
func (svc *HiveovService) GetSM() *session.WebSessionManager {
	return svc.sm
}

// Start the web server and publish the service's own TD.
// domainName is the listening domain name that matches the server certificate.
func (svc *HiveovService) Start(hc transports.IClientConnection) error {
	slog.Info("Starting HiveovService", "clientID", hc.GetClientID())
	svc.hc = hc

	// publish a TD for the service and set allowable roles in this case only a management capability is published
	err := authz.UserSetPermissions(hc, authz.ThingPermissions{
		AgentID: hc.GetClientID(),
		ThingID: src.HiveoviewServiceID,
		Allow:   []authz.ClientRole{authz.ClientRoleAdmin, authz.ClientRoleService, authz.ClientRoleManager},
	})
	if err != nil {
		slog.Error("failed to set the hiveoview service permissions", "err", err.Error())
	}

	// Setup the handling of incoming web sessions
	// re-use the runtime connection manager
	hubURL := hc.GetServerURL()
	svc.sm = session.NewWebSessionManager(hubURL, svc.signingKey, svc.caCert, hc, svc.noState)

	// parse the templates
	svc.tm.ParseAllTemplates()

	// TODO: hostname configurable as the server can live elsewhere
	// This is an SSE server
	urlParts, _ := url.Parse(hc.GetServerURL())
	svc.serverURL = fmt.Sprintf("https://%s:%d%s", urlParts.Hostname(), svc.port, WebSsePath)

	// Start the TLS server for serving the UI
	// The server certificate must match the domain name used here, so just
	// use the hub url.
	if svc.serverCert != nil {

		tlsServer, router := tlsserver.NewTLSServer(
			"", svc.port, svc.serverCert, svc.caCert)

		svc.CreateRoutes(router, svc.rootPath)
		svc.tlsServer = tlsServer
		err = tlsServer.Start()

	} else {
		// add the routes
		router := chi.NewRouter()
		svc.CreateRoutes(router, svc.rootPath)

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
	// last, publish this service's TD and properties
	_ = svc.PublishServiceTD()
	_ = svc.PublishServiceProps()

	return nil
}

func (svc *HiveovService) Stop() {
	slog.Info("Stopping HiveovService")
	// TODO: send event the service has stopped
	svc.hc.Disconnect()
	svc.sm.CloseAllWebSessions()
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
//	noState flag to not use the state service for persistance. Intended for testing.
func NewHiveovService(serverPort int, debug bool,
	signingKey ed25519.PrivateKey, rootPath string,
	serverCert *tls.Certificate, caCert *x509.Certificate,
	noState bool,
) *HiveovService {
	templatePath := rootPath
	if rootPath != "" {
		templatePath = path.Join(rootPath, "views")
	}
	if signingKey == nil {
		_, signingKey, _ = ed25519.GenerateKey(rand.Reader)
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
		noState:      noState,
	}
	return &svc
}
