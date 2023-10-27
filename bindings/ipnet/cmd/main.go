package main

import (
	"context"
	"github.com/hiveot/hub/bindings/ipnet/service"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"os"
)

// Run the ipnet service binding
func main() {
	// setup environment
	env := utils.GetAppEnvironment("", true)
	logging.SetLogging(env.LogLevel, "")
	slog.Warn("Starting IpNet binding", "clientID", env.ClientID, "loglevel", env.LogLevel)

	// locate the hub, load CA certificate, load service key and token and connect
	hc, err := hubclient.ConnectToHub("", env.ClientID, env.CertsDir, "")
	if err != nil {
		slog.Error("Failed connecting to the Hub", "err", err)
		os.Exit(1)
	}
	svc := service.NewIpNetBinding(hc)
	err = svc.Start()
	if err != nil {
		slog.Error("Failed starting service", "err", err)
		os.Exit(1)
	}
	utils.WaitForSignal(context.Background())
	svc.Stop()
	slog.Warn("IpNet binding has stopped")
}
