package main

import (
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authn/service"
	"github.com/hiveot/hub/core/authn/service/unpwstore"
	"github.com/hiveot/hub/core/config"
	"github.com/hiveot/hub/lib/svcconfig"
	"golang.org/x/exp/slog"
)

// main entry point to start the authentication service
func main() {
	// get defaults
	f, _, _ := svcconfig.SetupFolderConfig(authn.AuthnServiceName)
	authServiceConfig := config.NewAuthnConfig(f.Stores)
	_ = f.LoadConfig(&authServiceConfig)

	pwStore := unpwstore.NewPasswordFileStore(authServiceConfig.PasswordFile)
	accountToken := LoadAccountToken(f.Certs)
	svc := service.NewAuthnService(pwStore, accountToken)
	err := svc.Start()
	if err != nil {
		slog.Error("Failed starting authn service", "err", err)
	}
	//err := svc.Stop()
}
