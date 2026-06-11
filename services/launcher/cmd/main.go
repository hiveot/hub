package main

import (
	"log/slog"
	"os"

	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/services/launcher/config"
	"github.com/hiveot/hub/services/launcher/service"
)

// Connect the launcher service
func main() {
	// setup environment and config
	env := factory.NewAppEnvironment("", true)
	utils.SetLogging(env.LogLevel, "")

	cfg := config.NewLauncherConfig()
	cfg.LogLevel = env.LogLevel
	cfg.LogsDir = env.LogsDir

	err := env.LoadConfig(&cfg)
	if err != nil {
		slog.Error("Failed loading launcher config: ", "err", err)
		os.Exit(1)
	}
	utils.SetLogging(cfg.LogLevel, "")

	// start the launcher but do not connect yet as the runtime can be started by the launcher itself.
	// the runtime will generate the launcher key and token.
	svc := service.NewLauncherService(env.ServerURL, env.AppID,
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
