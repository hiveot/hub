// Package main with the provisioning service
package main

import (
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/runtime"
	"github.com/hiveot/hub/services/idprov/service"
	"log/slog"
	"os"
	"path"
)

// TODO: merge the server with a web server that hosts the admin ui server
// TODO: option to enable/disable the request server

// DefaultIDProvPort is the default listening port for https requests
const DefaultIDProvPort = 8445

// Start the service.
// Preconditions:
//  1. A loginID and keys for this service must already have been added.
//     This can be done manually using the hubcli or simply be starting it using the launcher.
//  2. The hub core config hub.yaml must be available to load the server cert.
func main() {
	var err error

	// Determine the folder layout and handle commandline options
	env := plugin.GetAppEnvironment("", true)
	logging.SetLogging(env.LogLevel, "")
	slog.Warn("Starting idprov service", "clientID", env.ClientID, "loglevel", env.LogLevel)

	// load the server cert
	// TODO: get server cert info from idprov config
	serverCertPath := path.Join(env.CertsDir, runtime.DefaultServerCertFile)
	serverKeyPath := path.Join(env.CertsDir, runtime.DefaultServerKeyFile)
	serverCert, err := certs.LoadTLSCertFromPEM(serverCertPath, serverKeyPath)
	if err != nil {
		slog.Error("idprov: Failed loading server certificate", "err", err)
		os.Exit(1)
	}

	// start the service using the connection and hub server certificate
	svc := service.NewIdProvService(DefaultIDProvPort, serverCert, env.CaCert)
	plugin.StartPlugin(svc, env.ClientID, env.CertsDir)
}
