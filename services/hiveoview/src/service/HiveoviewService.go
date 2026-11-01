package service

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"path"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/bucketstore"
	"github.com/hiveot/hivekit/go/cells/bucketstore/kvbtreestore"
	"github.com/hiveot/hivekit/go/cells/thing"
	"github.com/hiveot/hivekit/go/cells/transport/tlsserver"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views"
)

const HiveoviewStoreName = "hiveoview.kvbtree"

// HiveoviewService operates the html web server.
// It utilizes gin, htmx and TempL for serving html.
// credits go to: https://github.com/marco-souza/gx/blob/main/cmd/server/server.go
type HiveoviewService struct {
	*thing.ExposedThing

	// URL to connect to the hub with
	// TODO: use a hub TD instead
	hubURL string

	// listening port on which to serve the UI
	port int
	// development configuration
	dev          bool
	shouldUpdate bool
	// router for UI paths
	router chi.Router

	// filesystem location of the ./static, webcomp, and ./views template root folder
	rootPath string
	// SSR templates
	tm *views.TemplateManager

	// The CA's for client auth
	rootCAs *x509.CertPool
	// The server's TLS certificate
	serverCert *tls.Certificate
	// The TLS server does serves the UI
	tlsServer api.IHttpServer

	// Manager for web sessions, one per connection
	sm *session.WebSessionManager

	// key for signing cookies
	signingKey ed25519.PrivateKey

	// run in debug mode, extra logging and render templates from file.
	debug bool

	// timeout to use with of hub client connections
	timeout time.Duration

	// backend storage of UI configuration for clients by clientID
	storeDir    string
	configStore bucketstore.IBucketStore
}

//func (svc *HiveoviewService) GetServerURL() string {
//	return fmt.Sprintf("https://%s:%d%s", svc.serverAddr, svc.port, WebSsePath)
//}

// callback handler when a new session is opened
// This publishes an event notification.
func (svc *HiveoviewService) onSessionOpened(sm *session.WebClientSession, nrSessions int) {
	svc.PubEvent(src.HiveoviewServiceID, src.NrActiveSessionsEvent, nrSessions)
}

// callback handler when a new session is closed
// This publishes an event notification.
func (svc *HiveoviewService) onSessionClosed(sm *session.WebClientSession, nrSessions int) {
	svc.PubEvent(src.HiveoviewServiceID, src.NrActiveSessionsEvent, nrSessions)
}

// GetSM returns the web session manager
// Intended for testing.
func (svc *HiveoviewService) GetSM() *session.WebSessionManager {
	return svc.sm
}

// Start publishes the service's TD.
//
//	ag is the service agent connection to the hub for publishing notifications
func (svc *HiveoviewService) Start() {
	slog.Info("Starting HiveoviewService", "clientID", svc.GetID())

	// var err error
	// storePath := path.Join(svc.storeDir, HiveoviewStoreName)
	// // svc.configStore = kvbtree.NewKVStore(storePath)
	// svc.configStore, err = kvbtreestore.OpenKVBTreeStore(storePath)
	// if err != nil {
	// 	slog.Error("Start: failed", err.Error())
	// }

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

	// // Setup the handling of incoming web sessions
	// // re-use the runtime connection manager
	// svc.sm = session.NewWebSessionManager(
	// 	svc.ag, svc.signingKey, svc.caCert, svc.configStore, svc.timeout)

	// // parse the templates
	// svc.tm.ParseAllTemplates()

	// TODO: hostname configurable as the server can live elsewhere
	// This is an SSE server
	//outbound := net.GetOutboundIP("")
	//hostName := outbound.String()
	//svc.serverURL = fmt.Sprintf("https://%s:%d%s", hostName, svc.port, WebSsePath)

	// // Start the TLS server for serving the UI
	// // The server certificate must match the domain name used here, so just
	// // use the hub url.
	// if svc.serverCert != nil {

	// 	httpCfg := tlsserver.NewTLSServerConfig("", svc.port, svc.serverCert, svc.caCert, true)
	// 	tlsServer, err := tlsserver.NewTLSServer(httpCfg, authenticator)

	// 	router := tlsServer.GetProtectedRoute()
	// 	svc.CreateRoutes(router, svc.rootPath)
	// 	svc.tlsServer = tlsServer
	// 	err = tlsServer.Start()

	// } else {
	// 	// add the routes
	// 	router := chi.NewRouter()
	// 	svc.CreateRoutes(router, svc.rootPath)

	// 	// For testing and debugging without certificate
	// 	addr := fmt.Sprintf(":%d", svc.port)
	// 	go func() {
	// 		err = http.ListenAndServe(addr, router)
	// 		if err != nil && !errors.Is(err, http.ErrServerClosed) {
	// 			slog.Error("Failed starting server", "err", err)
	// 			time.Sleep(time.Second)
	// 			os.Exit(0)
	// 		}
	// 	}()
	// }
	// last, publish this service's TD and properties
	_ = svc.PublishServiceTD()
	svc.PublishServiceProps()

	return
}

// Start the web server
func (svc *HiveoviewService) startWebServer(serverPort int, debug bool,
	rootPath string, serverCert *tls.Certificate, rootCAs *x509.CertPool,
) (api.IHttpServer, error) {

	// start the TLS server
	tlsConfig := tlsserver.NewTLSServerConfig("", serverPort, serverCert, rootCAs, true)
	tlsServer, err := tlsserver.NewTLSServer(tlsConfig, nil)
	// rootRouter := chi.NewRouter()
	router := tlsServer.GetPublicRoute()
	svc.CreateRoutes(router, svc.rootPath)
	return tlsServer, err
}

func (svc *HiveoviewService) Stop() {
	slog.Info("Stopping HiveoviewService")
	// TODO: send event the service has stopped
	// svc.ag.Stop()
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

// NewHiveovService creates a new HiveOT service instance that serves the
// content from a http.FileSystem.
//
// rootPath is the root directory when serving files from the filesystem.
// This must contain static/, views/ and webc/ directories.
// If empty, the embedded filesystem is used.
//
//	hubURL is the URL for creating session connections to the hub.
//	serverPort is the port of the web server will listen on
//	debug to enable debugging output
//	rootPath containing the templates in the given folder or "" to use the embedded templates
//	serverCert server TLS certificate
//	caCert server CA certificate
//	timeout of client hub connections
//	storeDir path to directory holding client session and dashboard data
func NewHiveovService(
	hubURL string,
	serverPort int, debug bool,
	rootPath string,
	serverCert *tls.Certificate, rootCAs *x509.CertPool,
	timeout time.Duration,
	storeDir string,
) (*HiveoviewService, error) {

	templatePath := rootPath
	if rootPath != "" {
		templatePath = path.Join(rootPath, "views")
	}
	// use the server cert for signing
	signingKey, _ := serverCert.PrivateKey.(ed25519.PrivateKey)
	if signingKey == nil {
		_, signingKey, _ = ed25519.GenerateKey(rand.Reader)
	}
	tm := views.InitTemplateManager(templatePath)
	svc := &HiveoviewService{
		ExposedThing: thing.NewExposedThing(src.HiveoviewServiceID, nil),
		// ExposedThing: thing.NewExposedThing(env.AppID, nil),
		// env:          env,
		port:         serverPort,
		shouldUpdate: true,
		debug:        debug,
		hubURL:       hubURL,
		signingKey:   signingKey,
		rootPath:     rootPath,
		tm:           tm,
		serverCert:   serverCert,
		rootCAs:      rootCAs,
		storeDir:     storeDir,
		timeout:      timeout,
	}

	var err error
	storePath := path.Join(svc.storeDir, HiveoviewStoreName)
	// svc.configStore = kvbtree.NewKVStore(storePath)
	svc.configStore, err = kvbtreestore.OpenKVBTreeStore(storePath)
	if err != nil {
		slog.Error("Start: failed", err.Error())
		return nil, err
	}

	// Setup the handling of incoming web sessions
	// re-use the runtime connection manager
	svc.sm = session.NewWebSessionManager(
		svc.onSessionOpened, svc.onSessionClosed, hubURL,
		svc.signingKey, svc.rootCAs, svc.configStore, svc.timeout)

	// parse the templates
	svc.tm.ParseAllTemplates()

	svc.tlsServer, err = svc.startWebServer(
		serverPort, debug, rootPath, serverCert, rootCAs)

	var _ api.IHiveCell = svc // interface check
	return svc, err
}
