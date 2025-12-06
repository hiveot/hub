package main

import (
	"log/slog"
	"os"

	"github.com/hiveot/hivekit/go/lib/logging"
	"github.com/hiveot/hivekit/go/lib/plugin"
	"github.com/hiveot/hub/services/launcher/config"
	"github.com/hiveot/hub/services/launcher/service"
)

// Connect the launcher service
func main() {
	// setup environment and config
	env := plugin.GetAppEnvironment("", true)
	logging.SetLogging(env.LogLevel, "")

	cfg := config.NewLauncherConfig()
	cfg.LogLevel = env.LogLevel
	cfg.LogsDir = env.LogsDir

	err := env.LoadConfig(&cfg)
	if err != nil {
		slog.Error("Failed loading launcher config: ", "err", err)
		os.Exit(1)
	}
	logging.SetLogging(cfg.LogLevel, "")

	// start the launcher but do not connect yet as the runtime can be started by the launcher itself.
	// the runtime will generate the launcher key and token.
	svc := service.NewLauncherService(env.ServerURL, env.ClientID,
		env.BinDir, env.PluginsDir, env.CertsDir, cfg)
	err = svc.Start()
	if err != nil {
		slog.Error("Failed starting launcher: ", "err", err)
		// we're going to exit. Don't leave the core running
		_ = svc.Stop()
		os.Exit(1)
	}
	slog.Warn("Launcher has started")
	// wait for a stop signal
	service.WaitForSignal()
	err = svc.Stop()
	if err != nil {
		os.Exit(2)
	}
}
