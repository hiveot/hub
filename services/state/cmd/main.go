package main

import (
	"github.com/hiveot/hub/core/state/service"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"log/slog"
	"path"
)

// Start the service.
// Precondition: A loginID and keys for this service must already have been added.
// This can be done manually using the hubcli or simply be starting it using the launcher.
func main() {
	env := plugin.GetAppEnvironment("", true)
	logging.SetLogging(env.LogLevel, "")
	slog.Warn("Starting state service", "clientID", env.ClientID, "loglevel", env.LogLevel)

	// startup
	storePath := path.Join(env.StoresDir, env.ClientID)
	svc := service.NewStateService(storePath)
	plugin.StartPlugin(svc, &env)
}
