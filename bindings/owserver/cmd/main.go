package main

import (
	"context"
	"github.com/hiveot/hub/bindings/owserver/config"
	"github.com/hiveot/hub/bindings/owserver/service"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"os"
)

// ServiceName is the default instance ID of this service.
// Used to name the configuration file, the certificate ID and as the
// publisher ID.
const ServiceName = "owserver"

func main() {
	// setup environment and config
	env := utils.GetAppEnvironment("", true)
	logging.SetLogging(env.LogLevel, "")
	cfg := config.NewConfig()

	_ = env.LoadConfig(env.ConfigFile, &cfg)
	if cfg.LogLevel != "" {
		logging.SetLogging(cfg.LogLevel, "")
	}

	// locate the hub, load CA certificate, load service key and token and connect
	hc, err := hubclient.ConnectToHub("", env.ClientID, env.CertsDir, "")
	if err != nil {
		slog.Error("Failed connecting to the Hub", "err", err)
		os.Exit(1)
	}

	// start the service
	binding := service.NewOWServerBinding(cfg, hc)
	err = binding.Start()
	if err != nil {
		slog.Error("failed starting owserver", "err", err.Error())
		os.Exit(1)
	}
	utils.WaitForSignal(context.Background())
	binding.Stop()

	os.Exit(0)
}
