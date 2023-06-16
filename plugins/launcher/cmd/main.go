package main

import (
	"github.com/hiveot/hub/lib/svcconfig"
	"github.com/hiveot/hub/plugins/launcher"
	"github.com/hiveot/hub/plugins/launcher/config"
	"github.com/hiveot/hub/plugins/launcher/service"
)

// Connect the launcher service
func main() {
	f, _, _ := svcconfig.SetupFolderConfig(launcher.ServiceName)
	cfg := config.NewLauncherConfig()
	_ = f.LoadConfig(&cfg)

	svc := service.NewLauncherService(f, cfg)

	svc.Start()
}
