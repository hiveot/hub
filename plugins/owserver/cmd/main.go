package main

import (
	"context"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/discovery"
	"github.com/hiveot/hub/lib/hubcl"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"os"
	"path"

	"github.com/hiveot/hub/plugins/owserver/internal"
)

// ServiceName is the default instance ID of this service.
// Used to name the configuration file, the certificate ID and as the
// publisher ID.
const ServiceName = "owserver"

func main() {
	env := utils.GetAppEnvironment("", false)
	config := internal.NewConfig()
	_ = env.LoadConfig(env.ConfigFile, &config)

	// If no URL is given, use discovery to locate the hub
	core := ""
	fullUrl := config.ServerURL
	if fullUrl == "" {
		fullUrl, core = discovery.LocateHub(0, true)
	}
	caCertFile := path.Join(env.CertsDir, certs.DefaultCaCertFile)
	caCert, err := certs.LoadX509CertFromPEM(caCertFile)
	if err != nil {
		slog.Error("Unable to load CA cert", "err", err, "caCertFile", caCertFile)
	}
	// Use the convention for clientID, token and key files
	hc := hubcl.NewHubClient(fullUrl, env.ClientID, nil, caCert, core)
	err = hc.ConnectWithTokenFile(env.TokenFile, env.KeyFile)

	if err != nil {
		slog.Error("unable to connect to Hub", "url", fullUrl, "err", err)
		panic("hub not found")
	}

	// start the service
	binding := internal.NewOWServerBinding(config, hc)
	err = binding.Start()
	if err != nil {
		slog.Error("failed starting owserver", "err", err.Error())
		os.Exit(1)
	}
	utils.WaitForSignal(context.Background())
	binding.Stop()

	os.Exit(0)
}
