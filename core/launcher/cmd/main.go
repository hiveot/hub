package main

import (
	"github.com/hiveot/hub/core/launcher/config"
	"github.com/hiveot/hub/core/launcher/service"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"os"
)

// Connect the launcher service
func main() {
	// setup environment and config
	env := utils.GetAppEnvironment("", true)
	logging.SetLogging(env.LogLevel, "")
	cfg := config.NewLauncherConfig()
	err := env.LoadConfig(env.ConfigFile, &cfg)
	if err != nil {
		slog.Error("Failed loading launcher config: ", "err", err)
		os.Exit(1)
	}
	if cfg.LogLevel != "" {
		logging.SetLogging(cfg.LogLevel, "")
	}

	// start the launcher but do not connect yet as the core can be started by the launcher itself.
	// the core will generate the launcher key and token.
	svc := service.NewLauncherService(env, cfg, nil)
	err = svc.Start()
	if err != nil {
		slog.Error("Failed starting launcher: ", "err", err)
		os.Exit(1)
	}

	// wait for a stop signal
	service.WaitForSignal()
	err = svc.Stop()
	if err != nil {
		os.Exit(2)
	}
}
