package main

import (
	"flag"
	"github.com/hiveot/hub/bindings/hiveoview/src/service"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"log/slog"
	"os"
	"path"
)

const port = 8080 // default webserver port

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
	sessionFile := ""
	serverPort := port

	flag.IntVar(&serverPort, "port", serverPort, "Webserver port")
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
	sessionFile = path.Join(storeDir, "sessions.json")
	svc := service.NewHiveovService(serverPort, false, sessionFile)
	// StartPlugin will connect to the hub and wait for signal to end
	plugin.StartPlugin(svc, &env)
}
