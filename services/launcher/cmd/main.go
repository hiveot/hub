package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/hiveot/hivekit/go/api"
	rcdevice_recipe "github.com/hiveot/hivekit/go/factory/recipes/rcdevice"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/services/launcher"
	launcher_service "github.com/hiveot/hub/services/launcher/service"
)

// Connect the launcher service to the hub
func main() {
	// setup environment and config
	env := api.NewAppEnvironment("", true)
	utils.SetLogging(env.LogLevel, "")

	cfg := launcher.NewLauncherConfig()
	err := env.LoadConfig(env.AppID, &cfg)
	if err != nil {
		slog.Error("Failed loading launcher config: ", "err", err)
		os.Exit(1)
	}
	if cfg.LogLevel != "" {
		env.LogLevel = cfg.LogLevel
	}
	utils.SetLogging(env.LogLevel, "")

	// start the launcher but do not connect yet as the runtime can be started by the launcher itself.
	// the runtime will generate the launcher key and token.
	svc, err := launcher_service.NewLauncherService(env, cfg)

	if err != nil {
		slog.Error("Failed creating launcher: ", "err", err)
		os.Exit(1)
	}
	r, _, err := rcdevice_recipe.NewRCDeviceRecipe(env, svc)
	if err != nil {
		fmt.Println("Launcher startup failed: " + err.Error())
		os.Exit(1)
	}
	r.Run()
}
