package main

import (
	"context"
	"github.com/hiveot/hub/lib/discovery"
	"github.com/hiveot/hub/lib/hubclient/hubconnect"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/plugins/owserver/config"
	"github.com/hiveot/hub/plugins/owserver/service"
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
	// If no URL is given, use discovery to locate the hub
	core := ""
	fullUrl := cfg.ServerURL

	if fullUrl == "" {
		fullUrl, core = discovery.LocateHub(0, true)
	}
	// Use the convention for clientID, token and key files
	hc := hubconnect.NewHubClient(fullUrl, env.ClientID, nil, env.CaCert, core)
	err := hc.ConnectWithTokenFile(env.TokenFile, env.KeyFile)

	if err != nil {
		slog.Error("unable to connect to Hub", "url", fullUrl, "err", err)
		panic("hub not found")
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
