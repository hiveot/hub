package main

import (
	"log/slog"
	"os"

	"github.com/hiveot/hivekit/go/cells/transport/clients"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/services/launcher"
	launcherpkg "github.com/hiveot/hub/services/launcher/pkg"
)

// Connect the launcher service to the hub
func main() {
	// setup environment and config
	env := api.NewAppEnvironment("", true)
	utils.SetLogging(env.LogLevel, "")

	cfg := launcher.NewLauncherConfig()
	cfg.LogLevel = env.LogLevel

	err := env.LoadConfig(&cfg)
	if err != nil {
		slog.Error("Failed loading launcher config: ", "err", err)
		os.Exit(1)
	}
	utils.SetLogging(cfg.LogLevel, "")

	// TODO: need authn TD. For now use the default serverURL config
	// should hubURL be discovered?
	cl, err := clients.NewTransportClient("", env.ServerURL, env.CaCert)

	// start the launcher but do not connect yet as the runtime can be started by the launcher itself.
	// the runtime will generate the launcher key and token.
	svc := launcherpkg.NewLauncherService(cl, env, cfg)

	err = svc.Start()
	if err != nil {
		slog.Error("Failed starting launcher: ", "err", err)
		// we're going to exit. Don't leave the core running
		svc.Stop()
		os.Exit(1)
	}
	slog.Warn("Launcher has started")
	// wait for a stop signal
	utils.WaitForSignal()
	svc.Stop()
}
