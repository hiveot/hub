// Package main with the thing directory store
package main

import (
	"context"
	"github.com/hiveot/hub/core/directory/service"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/hubclient/hubconnect"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"os"
	"path"
)

// name of the storage file
const storeFile = "directorystore.json"

// Connect the service
func main() {
	env := utils.GetAppEnvironment("", true)
	logging.SetLogging(env.LogLevel, "")

	// this locates the hub, load certificate, load service tokens and connect
	hc, err := hubconnect.ConnectToHub("", env.ClientID, env.CertsDir, "")
	if err != nil {
		slog.Error("Failed connecting to the Hub", "err", err)
		os.Exit(1)
	}

	// startup
	storePath := path.Join(env.StoresDir, env.ClientID, storeFile)
	store := kvbtree.NewKVStore(env.ClientID, storePath)
	err = store.Open()
	if err != nil {
		panic("unable to open the directory store")
	}
	svc := service.NewDirectoryService(store, hc)
	err = svc.Start()
	if err != nil {
		slog.Error("Failed starting directory service", "err", err)
		os.Exit(1)
	}
	utils.WaitForSignal(context.Background())
	err = svc.Stop()
	if err != nil {
		os.Exit(2)
	}
}
