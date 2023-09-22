package main

import (
	"github.com/hiveot/hub/core/launcher"
	"github.com/hiveot/hub/core/launcher/config"
	service2 "github.com/hiveot/hub/core/launcher/service"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"os"
)

// Connect the launcher service
func main() {
	logging.SetLogging("info", "")
	f := utils.GetFolders("", false)
	cfg := config.NewLauncherConfig()

	err := f.LoadConfig(launcher.ServiceName+".yaml", &cfg)
	if err != nil {
		slog.Error("Failed loading launcher config: ", "err", err)
		os.Exit(1)
	}
	svc := service2.NewLauncherService(f, cfg)
	err = svc.Start()
	if err != nil {
		slog.Error("Failed starting launcher: ", "err", err)
		os.Exit(1)
	}
	service2.WaitForSignal()
	err = svc.Stop()
	if err != nil {
		os.Exit(2)
	}
}
