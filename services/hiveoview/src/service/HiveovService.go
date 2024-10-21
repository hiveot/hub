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
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/tlsserver"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views"
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
	hc hubclient.IAgentClient

	// cookie signing
	signingKey *ecdsa.PrivateKey

	// run in debug mode, extra logging and reload templates render
	debug bool
}

// Start the web server and publish the service's own TD.
func (svc *HiveovService) Start(hc hubclient.IAgentClient) error {
	slog.Info("Starting HiveovService", "clientID", hc.GetClientID())
	svc.hc = hc

	// publish a TD for each service capability and set allowable roles
	// in this case only a management capability is published
	err := authz.UserSetPermissions(hc, authz.ThingPermissions{
		AgentID: hc.GetClientID(),
		ThingID: src.HiveoviewServiceID,
		Allow:   []authz.ClientRole{authz.ClientRoleAdmin, authz.ClientRoleService},
	})
	if err != nil {
		slog.Error("failed to set the hiveoview service permissions", "err", err.Error())
	}

	// Setup the handling of incoming web sessions
	sm := session.GetSessionManager()
	connStat := hc.GetStatus()
	sm.Init(connStat.HubURL, svc.signingKey, connStat.CaCert, hc)

	// parse the templates
	svc.tm.ParseAllTemplates()

	// Start the TLS server for serving the UI
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
