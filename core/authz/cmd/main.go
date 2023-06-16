package main

import (
	"github.com/hiveot/hub/core/authz"
	"github.com/hiveot/hub/core/authz/service"
	"os"
	"path/filepath"

	"github.com/hiveot/hub/lib/svcconfig"
)

const aclStoreFile = "authz.acl"

// main entry point to start the authorization service
func main() {
	f, _, _ := svcconfig.SetupFolderConfig(authz.ServiceName)
	aclStoreFolder := filepath.Join(f.Stores, authz.ServiceName)
	aclStorePath := filepath.Join(aclStoreFolder, aclStoreFile)
	_ = os.Mkdir(aclStoreFolder, 0700)

	svc := service.NewAuthzService(aclStorePath)
	err := svc.Start()
	_ = err
}
