// Package main with the history store
package main

import (
	"fmt"
	"log/slog"
	"path"

	"github.com/hiveot/hub/lib/buckets/bucketstore"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/services/history/config"
	"github.com/hiveot/hub/services/history/service"
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
	store, err := bucketstore.NewBucketStore(storesDir, "history", cfg.Backend)
	if err == nil {
		err = store.Open()
	}
	if err != nil {
		err = fmt.Errorf("can't open history bucket store: %w", err)
		slog.Error(err.Error())
		panic(err.Error())
	}
	svc := service.NewHistoryService(store)
	plugin.StartPlugin(svc, env.ClientID, env.CertsDir, env.ServerURL)
}
