// Package main with the history store
package main

import (
	"fmt"
	"github.com/hiveot/hub/core/history/config"
	"github.com/hiveot/hub/core/history/service"
	"github.com/hiveot/hub/lib/buckets/bucketstore"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"log/slog"
	"path"
)

// Connect the history store service
func main() {
	env := plugin.GetAppEnvironment("", true)
	logging.SetLogging(env.LogLevel, "")
	slog.Warn("Starting history service", "clientID", env.ClientID, "loglevel", env.LogLevel)

	storesDir := path.Join(env.StoresDir, env.ClientID)
	cfg := config.NewHistoryConfig(storesDir)
	_ = env.LoadConfig(&cfg)

	// the service uses the bucket store to store history
	store := bucketstore.NewBucketStore(cfg.StoreDirectory, "history", cfg.Backend)
	err := store.Open()
	if err != nil {
		err = fmt.Errorf("can't open history bucket store: %w", err)
		slog.Error(err.Error())
		panic(err.Error())
	}
	svc := service.NewHistoryService(store)
	plugin.StartPlugin(svc, &env)
}
