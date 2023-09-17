package main

import (
	"github.com/hiveot/hub/lib/discovery"
	"github.com/hiveot/hub/lib/utils"
	"golang.org/x/exp/slog"
	"os"

	"github.com/hiveot/hub/plugins/owserver/internal"
)

// ServiceName is the default instance ID of this service.
// Used to name the configuration file, the certificate ID and as the
// publisher ID.
const ServiceName = "owserver"

func main() {
	f := utils.GetFolders("", false)
	config := internal.NewConfig()
	_ = f.LoadConfig(ServiceName, &config)

	fullUrl := config.HubURL
	if fullUrl == "" {
		fullUrl = discovery.LocateHub(0)
	}
	hc := NewHubClient(config.ID)
	err := hc.ConnectWithCert(fullUrl, config.ID, clientCert, caCert)
	if err != nil {
		slog.Error("unable to connect to Hub", "url", fullUrl, "err", err)
		panic("hub not found")
	}

	// start the service
	binding := internal.NewOWServerBinding(config, hc)
	utils.ExitOnSignal(func() {
		binding.Stop()
	})
	err = binding.Start()

	if err != nil {
		slog.Error("Failed to start", "err", err, "service", ServiceName)
		os.Exit(1)
	}
	os.Exit(0)
}
