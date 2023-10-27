// Package main with the provisioning service
package main

import (
	"context"
	"github.com/hiveot/hub/core/config"
	"github.com/hiveot/hub/core/idprov/service"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"os"
	"path"
)

// TODO: merge the server with a web server that hosts the admin ui server
// TODO: option to enable/disable the request server

// DefaultIDProvPort is the default listening port for https requests
const DefaultIDProvPort = 9444

// Start the service.
// Preconditions:
//  1. A loginID and keys for this service must already have been added.
//     This can be done manually using the hubcli or simply be starting it using the launcher.
//  2. The hub core config hub.yaml must be available to load the server cert.
func main() {
	var err error

	// Determine the folder layout and handle commandline options
	env := utils.GetAppEnvironment("", true)
	logging.SetLogging(env.LogLevel, "")
	slog.Warn("Starting idprov service", "clientID", env.ClientID, "loglevel", env.LogLevel)

	// load the server cert
	// TODO: get server cert info from idprov config
	serverCertPath := path.Join(env.CertsDir, config.DefaultServerCertFile)
	serverKeyPath := path.Join(env.CertsDir, config.DefaultServerKeyFile)
	serverCert, err := certs.LoadTLSCertFromPEM(serverCertPath, serverKeyPath)
	if err != nil {
		slog.Error("idprov: Failed loading server certificate", "err", err)
		os.Exit(1)
	}

	// locate the hub, load CA certificate, load service key and token and connect
	hc, err := hubclient.ConnectToHub("", env.ClientID, env.CertsDir, "")
	if err != nil {
		slog.Error("Failed connecting to the Hub", "err", err)
		os.Exit(1)
	}

	// start the service using the connection and hub server certificate
	svc := service.NewIdProvService(hc, DefaultIDProvPort, serverCert, env.CaCert)
	err = svc.Start()
	if err != nil {
		slog.Error("Failed starting idprov service", "err", err)
		os.Exit(1)
	}

	utils.WaitForSignal(context.Background())
	svc.Stop()
	slog.Warn("Stopped idprov service")
}
