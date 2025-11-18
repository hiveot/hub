// Package main with the history store
package main

import (
	"fmt"
	"log/slog"
	"path"

	"github.com/hiveot/hivehub/lib/buckets/bucketstore"
	"github.com/hiveot/hivehub/lib/plugin"
	"github.com/hiveot/hivehub/services/history/config"
	"github.com/hiveot/hivehub/services/history/service"
	"github.com/hiveot/hivekitgo/logging"
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
