package main

import (
	"crypto/ed25519"
	"flag"
	"log/slog"
	"os"
	"path"
	"time"

	"github.com/hiveot/hivekit/go/certs"
	"github.com/hiveot/hivekit/go/keys"
	"github.com/hiveot/hivekit/go/logging"
	"github.com/hiveot/hivekit/go/plugin"
	"github.com/hiveot/hub/runtime"
	"github.com/hiveot/hub/services/hiveoview/config"
	"github.com/hiveot/hub/services/hiveoview/src/service"
)

const defaultServerPort = 8443
const serverCertFile = runtime.DefaultServerCertFile

// FYI, not all browsers support certificates with ed25519 keys, so this file contains a ecdsa key
const serverKeyFile = runtime.DefaultServerKeyFile
const TemplateRootPath = "services/hiveoview/src"

// hiveoview will use the 'hubKey.pem' key and hubCert.pem for the server cert
// and caCert.pem for connecting to the hub message bus.
//
// Note that the service requires a server cert, server CA and a client auth token.
//
// The launcher will automatically generate an auth token in the certs directory
// with the service name.
// To generate a token manually the hubcli can be used:
// "hubcli gentoken __hiveoview", which generates token in: certs/__hiveoview.token
// If the test user __hiveoview doesn't exist it will be added and a private key
// generated.
//
// By default the templates are embedded. For development it can be useful to reload
// the templates for each request. Use the --extfs flag for this.
func main() {
	var signingKey ed25519.PrivateKey
	serverPort := defaultServerPort
	extfs := false

	flag.IntVar(&serverPort, "defaultServerPort", serverPort, "Webserver listening defaultServerPort")
	flag.BoolVar(&extfs, "extfs", extfs, "Use external gohtml filesystem")
	env := plugin.GetAppEnvironment("", true)
	env.LogLevel = "info"
	logging.SetLogging(env.LogLevel, "")

	// this config will be replaced with hiveoview Thing config
	cfg := config.NewHiveoviewConfig(serverPort)
	_ = env.LoadConfig(&cfg)
	// each app instance has its own storage directory
	storageDir := path.Join(env.StoresDir, env.ClientID)

	//storeDir := path.Join(env.StoresDir, "hiveoview")
	//err := os.MkdirAll(storeDir, 0700)
	//if err != nil {
	//	slog.Error("Unable to create session store", "err", err.Error())
	//	panic(err.Error())
	//}
	// serve the hiveoview web pages
	keyData, err := os.ReadFile(env.KeyFile)
	if err == nil {
		k := keys.NewEd25519Key()
		err = k.ImportPrivate(string(keyData))
		signingKey = k.PrivateKey().(ed25519.PrivateKey)
		//signingKey, err = jwt.ParseECPrivateKeyFromPEM(keyData)
	}
	// development only, serve files and parse templates from filesystem
	rootPath := ""
	if extfs {
		cwd, _ := os.Getwd()
		rootPath = path.Join(cwd, TemplateRootPath)
	}
	// A server certificate is needed in the certs directory
	serverCertPath := path.Join(env.CertsDir, serverCertFile)
	serverKeyPath := path.Join(env.CertsDir, serverKeyFile)
	serverCert, err := certs.LoadTLSCertFromPEM(serverCertPath, serverKeyPath)
	if err != nil {
		slog.Error("Unable to load server certificate: " + err.Error())
		return
	}
	// FIXME: currently timeout is set to 20 sec to allow slow requests to complete.
	// However these requests should handle multiple async updates instead.
	// for example zwavejs refresh device info can take up to 20sec
	svc := service.NewHiveovService(
		cfg.ServerPort, false, signingKey, rootPath, serverCert, env.CaCert,
		time.Second*20, storageDir)

	// StartPlugin will connect to the hub and wait for a signal to end.
	plugin.StartPlugin(svc, env.ClientID, env.CertsDir, env.ServerURL)
}
