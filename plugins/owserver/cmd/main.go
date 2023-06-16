package main

import (
	"github.com/hiveot/hub/lib/utils"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/svcconfig"

	"github.com/hiveot/hub/plugins/owserver/internal"
)

func main() {
	f, bindingCert, caCert := svcconfig.SetupFolderConfig(internal.DefaultID)
	config := internal.NewConfig()
	_ = f.LoadConfig(&config)

	fullUrl := config.HubURL
	if fullUrl == "" {
		fullUrl = hubclient.LocateHub(0)
	}
	hc := hubclient.NewHubClient(config.ID) // <-
	err := hc.ConnectWithCert(fullUrl, bindingCert, caCert)
	if err != nil {
		logrus.Fatalf("unable to connect to Hub on %s: %s", fullUrl, err)
	}

	// start the service
	binding := internal.NewOWServerBinding(config, hc)
	utils.ExitOnSignal(func() {
		binding.Stop()
	})
	err = binding.Start()

	if err != nil {
		logrus.Errorf("%s: Failed to start: %s", internal.DefaultID, err)
		os.Exit(1)
	}
	os.Exit(0)
}
