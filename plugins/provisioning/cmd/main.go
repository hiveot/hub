// Package main with the provisioning service
package main

import (
	"github.com/hiveot/hub/core/certs"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/svcconfig"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/plugins/provisioning"
	"github.com/hiveot/hub/plugins/provisioning/service"
	"golang.org/x/exp/slog"
	"os"
)

// Connect the provisioning service
// - dependent on certs service
func main() {
	var certsClient certs.ICerts
	var err error
	var serviceID = provisioning.ServiceName

	// Determine the folder layout and handle commandline options
	f, clientCert, caCert := svcconfig.SetupFolderConfig(serviceID)

	fullUrl := hubclient.LocateHub(0)

	hc := hubclient.NewHubClient(serviceID)
	err = hc.ConnectWithCert(fullUrl, serviceID, clientCert, caCert)
	if err != nil {
		slog.Error("unable to connect to Hub", "url", fullUrl, "err", err)
		panic("hub not found")
	}

	// use the certificate service to get its capability for issuing device certificates
	// this is only allowed when authenticated with a certificate or a socket connection
	certsClient = NewCertsClient()

	if err != nil {
		panic("need access to device/verify certs: " + err.Error())
	}
	svc := service.NewProvisioningService(certsClient)
	utils.ExitOnSignal(func() {
		svc.Stop()
	})
	err := svc.Start()

	if err != nil {
		slog.Error("Failed to start", "err", err, "service", serviceID)
		os.Exit(1)
	}
	os.Exit(0)
}
