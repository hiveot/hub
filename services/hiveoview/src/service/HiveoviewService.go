package service

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/agent"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	bucketstorepkg "github.com/hiveot/hivekit/go/modules/bucketstore/pkg"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/tlsserver"
	tlsserverpkg "github.com/hiveot/hivekit/go/modules/transport/tlsserver/pkg"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views"
)

const HiveoviewStoreName = "hiveoview.kvbtree"

// HiveoviewService operates the html web server.
// It utilizes gin, htmx and TempL for serving html.
// credits go to: https://github.com/marco-souza/gx/blob/main/cmd/server/server.go
type HiveoviewService struct {
	modules.HiveModuleBase

	//serverAddr   string // listening address
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
	tlsServer  transport.IHttpServer

	// a web session per connections
	sm *session.WebSessionManager

	// ag agent hub client of this service.
	// This client's CA and URL is also used to establish client sessions.
	ag *agent.Agent

	// cookie signing
	signingKey ed25519.PrivateKey

	// run in debug mode, extra logging and reload templates render
	debug bool

	// timeout of hub client connections
	timeout time.Duration

	// backend storage of UI configuration for clients by clientID
	storeDir    string
	configStore bucketstore.IBucketStore
}

//func (svc *HiveoviewService) GetServerURL() string {
//	return fmt.Sprintf("https://%s:%d%s", svc.serverAddr, svc.port, WebSsePath)
//}

// GetSM returns the web session manager
// Intended for testing.
func (svc *HiveoviewService) GetSM() *session.WebSessionManager {
	return svc.sm
}

// Start the web server and publish the service's own TD.
//
// This is invoked by the plugin library.
//
//	ag is the service agent connection to the hub for publishing notifications
func (svc *HiveoviewService) Start() error {
	slog.Info("Starting HiveoviewService", "clientID", svc.ag.GetThingID())

	storePath := path.Join(svc.storeDir, HiveoviewStoreName)
	// svc.configStore = kvbtree.NewKVStore(storePath)
	svc.configStore = bucketstorepkg.NewKVBTreeStore(storePath)
	err := svc.configStore.Open()
	if err != nil {
		return err
	}

	// publish a TD for the service and set allowable roles in this case only a management capability is published
	// authzClient := authzpkg.NewAuthzClient(ag)
	// err = authz.UserSetPermissions(ag, authz.ThingPermissions{
	// 	AgentID: ag.GetClientID(),
	// 	ThingID: src.HiveoviewServiceID,
	// 	Allow:   []authz.ClientRole{authz.ClientRoleAdmin, authz.ClientRoleService, authz.ClientRoleManager},
	// })
	// if err != nil {
	// 	slog.Error("failed to set the hiveoview service permissions", "err", err.Error())
	// }

	// Setup the handling of incoming web sessions
	// re-use the runtime connection manager
	svc.sm = session.NewWebSessionManager(
		svc.ag, svc.signingKey, svc.caCert, svc.configStore, svc.timeout)

	// parse the templates
	svc.tm.ParseAllTemplates()

	// TODO: hostname configurable as the server can live elsewhere
	// This is an SSE server
	//outbound := net.GetOutboundIP("")
	//hostName := outbound.String()
	//svc.serverURL = fmt.Sprintf("https://%s:%d%s", hostName, svc.port, WebSsePath)

	// Start the TLS server for serving the UI
	// The server certificate must match the domain name used here, so just
	// use the hub url.
	if svc.serverCert != nil {

		httpCfg := tlsserver.NewTLSServerConfig("", svc.port, svc.serverCert, svc.caCert, true)
		tlsServer := tlsserverpkg.NewTLSServer(httpCfg, authenticator)

		router := tlsServer.GetProtectedRoute()
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
	svc.PublishServiceProps()

	return nil
}

func (svc *HiveoviewService) Stop() {
	slog.Info("Stopping HiveoviewService")
	// TODO: send event the service has stopped
	svc.ag.Stop()
	svc.sm.CloseAllWebSessions()
	if svc.tlsServer != nil {
		svc.tlsServer.Stop()
	}
	_ = svc.configStore.Close()
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
//	ag is the agent used to connect to the hub
//	serverPort is the port of the web server will listen on
//	debug to enable debugging output
//	signingKey used to sign cookies. Using nil means that a server restart will invalidate the cookies
//	rootPath containing the templates in the given folder or "" to use the embedded templates
//	serverCert server TLS certificate
//	caCert server CA certificate
//	timeout of client hub connections
//	storeDir path to directory holding client session and dashboard data
func NewHiveovService(
	ag *agent.Agent, serverPort int, debug bool,
	signingKey ed25519.PrivateKey, rootPath string,
	serverCert *tls.Certificate, caCert *x509.Certificate,
	timeout time.Duration,
	storeDir string,
) *HiveoviewService {
	templatePath := rootPath
	if rootPath != "" {
		templatePath = path.Join(rootPath, "views")
	}
	if signingKey == nil {
		_, signingKey, _ = ed25519.GenerateKey(rand.Reader)
	}
	tm := views.InitTemplateManager(templatePath)
	svc := &HiveoviewService{
		HiveModuleBase: *modules.NewHiveModuleBase(src.HiveoviewServiceID, 0),
		ag:             ag,
		port:           serverPort,
		shouldUpdate:   true,
		debug:          debug,
		signingKey:     signingKey,
		rootPath:       rootPath,
		tm:             tm,
		serverCert:     serverCert,
		caCert:         caCert,
		storeDir:       storeDir,
		timeout:        timeout,
	}
	var _ modules.IHiveModule = svc // interface check
	return svc
}
