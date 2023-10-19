// Package main with the history store
package main

import (
	"context"
	"fmt"
	"github.com/hiveot/hub/core/history/config"
	"github.com/hiveot/hub/core/history/service"
	"github.com/hiveot/hub/lib/buckets/bucketstore"
	"github.com/hiveot/hub/lib/hubclient/hubconnect"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"os"
	"path"
)

// Connect the history store service
func main() {
	env := utils.GetAppEnvironment("", true)
	logging.SetLogging(env.LogLevel, "")

	// this locates the hub, load certificate, load service tokens and connect
	hc, err := hubconnect.ConnectToHub("", env.ClientID, env.CertsDir, "")
	if err != nil {
		slog.Error("Failed connecting to the Hub", "err", err)
		os.Exit(1)
	}

	storesDir := path.Join(env.StoresDir, env.ClientID)
	cfg := config.NewHistoryConfig(storesDir)
	_ = env.LoadConfig(env.ConfigFile, &cfg)

	// the service uses the bucket store to store history
	store := bucketstore.NewBucketStore(cfg.StoreDirectory, hc.ClientID(), cfg.Backend)
	err = store.Open()
	if err != nil {
		err = fmt.Errorf("can't open history bucket store: %w", err)
		slog.Error(err.Error())
		panic(err.Error())
	}
	svc := service.NewHistoryService(hc, store)
	err = svc.Start()
	if err != nil {
		slog.Error("Failed starting history service", "err", err)
		os.Exit(1)
	}
	utils.WaitForSignal(context.Background())
	err = svc.Stop()
	if err != nil {
		os.Exit(2)
	}
}
