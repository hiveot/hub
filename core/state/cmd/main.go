package main

import (
	"context"
	"github.com/hiveot/hub/core/state/service"
	"github.com/hiveot/hub/lib/hubclient/hubconnect"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"os"
	"path"
)

// Start the service.
// Precondition: A loginID and keys for this service must already have been added.
// This can be done manually using the hubcli or simply be starting it using the launcher.
func main() {
	env := utils.GetAppEnvironment("", true)
	logging.SetLogging(env.LogLevel, "")
	slog.Warn("Starting state service", "clientID", env.ClientID, "loglevel", env.LogLevel)

	// locate the hub, load CA certificate, load service key and token and connect
	hc, err := hubconnect.ConnectToHub("", env.ClientID, env.CertsDir, "")
	if err != nil {
		slog.Error("Failed connecting to the Hub", "err", err)
		os.Exit(1)
	}

	// startup
	storePath := path.Join(env.StoresDir, env.ClientID)
	svc := service.NewStateService(hc, storePath)
	err = svc.Start()
	if err != nil {
		slog.Error("Failed starting directory service", "err", err)
		os.Exit(1)
	}
	utils.WaitForSignal(context.Background())
	svc.Stop()
	slog.Warn("Stopped state service")
}
