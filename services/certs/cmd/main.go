package main

import (
	"crypto/x509"
	"log/slog"
	"os"
	"path"

	"github.com/hiveot/hivehub/lib/plugin"
	"github.com/hiveot/hivehub/services/certs/service/selfsigned"
	"github.com/hiveot/hivekitgo/certs"
	"github.com/hiveot/hivekitgo/keys"
	"github.com/hiveot/hivekitgo/logging"
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
