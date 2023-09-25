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
	f := utils.GetFolders("", false)
	config := internal.NewConfig()
	_ = f.LoadConfig(ServiceName+".yaml", &config)

	// When discovery is not used, HubClient bases it on the URL.
	core := ""
	fullUrl := config.ServerURL
	if fullUrl == "" {
		fullUrl, core = discovery.LocateHub(0, true)
	}
	caCertFile := path.Join(f.Certs, certs.DefaultCaCertFile)
	caCert, err := certs.LoadX509CertFromPEM(caCertFile)
	if err != nil {
		slog.Error("Unable to load CA cert", "err", err, "caCertFile", caCertFile)
	}
	// This service uses pre-generated keys and auth token for authentication & authorization.
	// These are generated in by the hubcli or the service launcher. The file names
	// match the serviceID from the config.
	hc := hubcl.NewHubClient(fullUrl, config.BindingID, nil, caCert, core)
	err = hc.ConnectWithTokenFile(config.AuthTokenFile, config.KeyFile)

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
