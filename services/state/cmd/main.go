package main

import (
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/services/state/service"
	"github.com/hiveot/hub/services/state/stateapi"
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
	// the clientID is that of the application binary
	storePath := path.Join(env.StoresDir, env.ClientID)
	svc := service.NewStateService(storePath)

	// TODO: support multiple instances at the client side.
	// The agentID is fixed as stateapi.StateAgentID so the client API knows who to call.
	env.ClientID = stateapi.AgentID
	plugin.StartPlugin(svc, env.ClientID, env.CertsDir)
}
