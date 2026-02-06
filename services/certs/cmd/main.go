package main

import (
	"crypto/x509"
	"log/slog"
	"os"
	"path"

	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/services/certs/service/selfsigned"
)

// Connect the certs service
//
//	commandline options:
//	--certs <certificate folder>
func main() {
	var caCert *x509.Certificate
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
	caKey, err := keys.NewKeyFromFile(caKeyPath)
	if err != nil {
		slog.Error("Error loading CA key",
			"caKeyPath", caKeyPath, "err", err)
		os.Exit(1)
	}

	svc := selfsigned.NewSelfSignedCertsService(caCert, caKey)

	plugin.StartPlugin(svc, env.ClientID, env.CertsDir, env.ServerURL)
}
