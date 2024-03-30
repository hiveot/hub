// Package main with the things directory store
package main

import (
	"github.com/hiveot/hub/core/directory/service"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"log/slog"
	"path"
)

// name of the storage file
const storeFile = "directory.kvbtree"

// Start the service.
// Precondition: A loginID and keys for this service must already have been added.
// This can be done manually using the hubcli or simply be starting it using the launcher.
func main() {
	env := plugin.GetAppEnvironment("", true)
	logging.SetLogging(env.LogLevel, "")
	slog.Warn("Starting directory service", "clientID", env.ClientID, "loglevel", env.LogLevel)

	// startup
	storePath := path.Join(env.StoresDir, env.ClientID, storeFile)
	store := kvbtree.NewKVStore(storePath)
	err := store.Open()
	if err != nil {
		panic("unable to open the directory store")
	}
	svc := service.NewDirectoryService(store)
	plugin.StartPlugin(svc, &env)
}
