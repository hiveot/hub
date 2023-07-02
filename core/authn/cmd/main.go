package main

import (
	"github.com/hiveot/hub/core/api"
	"github.com/hiveot/hub/core/authn/service"
	"github.com/hiveot/hub/lib/svcconfig"
)

// main entry point to start the authentication service
func main() {
	// get defaults
	f, _, _ := svcconfig.SetupFolderConfig(api.ServiceName)
	authServiceConfig := service.NewAuthnConfig(f.Stores)
	_ = f.LoadConfig(&authServiceConfig)

	svc := service.NewAuthnService(authServiceConfig)
	svc.Start()
	//err := svc.Stop()
}
