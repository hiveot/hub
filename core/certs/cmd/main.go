package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"github.com/hiveot/hub/core/certs/service/selfsigned"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"log/slog"
	"os"
	"path"
)

// Connect the certs service
//
//	commandline options:
//	--certs <certificate folder>
func main() {
	var caCert *x509.Certificate
	var caKey *ecdsa.PrivateKey
	var err error

	env := plugin.GetAppEnvironment("", true)
	logging.SetLogging(env.LogLevel, "")
	slog.Warn("Starting certs service", "clientID", env.ClientID, "loglevel", env.LogLevel)

	// This service needs the CA certificate and key to operate
	caCertPath := path.Join(env.CertsDir, certs.DefaultCaCertFile)
	caKeyPath := path.Join(env.CertsDir, certs.DefaultCaKeyFile)

	slog.Info("Loading CA certificate and key", "dir", env.CertsDir)
	caCert, err = certs.LoadX509CertFromPEM(caCertPath)
	if err != nil {
		slog.Error("Failed loading CA certificate",
			"caCertPath", caCertPath, "err", err)
		os.Exit(1)
	}
	caKey, err = certs.LoadKeysFromPEM(caKeyPath)
	if err != nil {
		slog.Error("Error loading CA key",
			"caKeyPath", caKeyPath, "err", err)
		os.Exit(1)
	}

	svc := selfsigned.NewSelfSignedCertsService(caCert, caKey)
	plugin.StartPlugin(svc, &env)

	// this locates the hub, load certificate, load service tokens and connect
	//hc, err := hubclient.ConnectToHub("", env.ClientID, env.CertsDir, "")
	//if err != nil {
	//	slog.Error("Failed connecting to the Hub", "err", err)
	//	os.Exit(1)
	//}
	//// startup
	//svc := selfsigned.NewSelfSignedCertsService(caCert, caKey)
	//err = svc.Start(hc)
	//if err != nil {
	//	slog.Error("Failed starting certs service", "err", err)
	//	os.Exit(1)
	//}
	//plugin.WaitForSignal()
	//err = svc.Stop()
	//slog.Warn("Stopped certs service")
	//if err != nil {
	//	os.Exit(2)
	//}
}
