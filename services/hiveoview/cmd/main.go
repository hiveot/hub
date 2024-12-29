package main

import (
	"crypto/ed25519"
	"flag"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/runtime"
	"github.com/hiveot/hub/services/hiveoview/src/service"
	"log/slog"
	"os"
	"path"
	"time"
)

const port = 8443 // default webserver TLS port
const serverCertFile = runtime.DefaultServerCertFile

// FYI, not all browsers support certificates with ed25519 keys, so this file contains a ecdsa key
const serverKeyFile = runtime.DefaultServerKeyFile
const TemplateRootPath = "services/hiveoview/src"

// During development, run with 'air' and set home to a working hiveot directory
// that has certs.
// hiveoview will use the 'hubKey.pem' key and hubCert.pem for the server cert
// and caCert.pem for connecting to the hub message bus.
//
// The easiest testing env is to run 'air', which automatically rebuilds the
// application on changes. It is configured to use a running hiveot setup at
// ~/bin/hiveot. (~ is expanded to the home directory)
// Eg, tmp/hiveoview --home ~/bin/hiveot --clientID __hiveoview
// See air.toml for the commandline.
//
// Note that the service requires a server cert, server CA and a client auth token.
// For server TLS it uses the existing hubCert/hubKey.pem certificate and key.
//
// To generate a token the hubcli can be used:
// "hubcli gentoken __hiveoview", which generates token in: certs/__hiveoview.token
// If the test user __hiveoview doesn't exist it will be added and a private key
// generated.
func main() {
	var signingKey ed25519.PrivateKey
	serverPort := port
	extfs := false

	flag.IntVar(&serverPort, "port", serverPort, "Webserver port")
	flag.BoolVar(&extfs, "extfs", extfs, "Use external gohtml filesystem")
	env := plugin.GetAppEnvironment("", true)
	env.LogLevel = "info"
	logging.SetLogging(env.LogLevel, "")

	storeDir := path.Join(env.StoresDir, "hiveoview")
	err := os.MkdirAll(storeDir, 0700)
	if err != nil {
		slog.Error("Unable to create session store", "err", err.Error())
		panic(err.Error())
	}
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
	svc := service.NewHiveovService(
		serverPort, false, signingKey, rootPath, serverCert, env.CaCert, false)

	// StartPlugin will connect to the hub and wait for signal to end
	plugin.StartPlugin(svc, env.ClientID, env.CertsDir)
}
