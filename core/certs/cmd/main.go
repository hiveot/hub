package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"github.com/hiveot/hub/core/certs"
	"github.com/hiveot/hub/core/certs/service/selfsigned"
	"path"

	"github.com/sirupsen/logrus"

	"github.com/hiveot/hub/lib/certsclient"
	"github.com/hiveot/hub/lib/svcconfig"
)

// Connect the certs service
//
//	commandline options:
//	--certs <certificate folder>
func main() {
	var caCert *x509.Certificate
	var caKey *ecdsa.PrivateKey
	var err error
	f, _, caCert := svcconfig.SetupFolderConfig(certs.ServiceName)

	// This service needs the CA certificate and key to operate
	caKeyPath := path.Join(f.Certs, certs.DefaultCaKeyFile)

	logrus.Infof("Loading CA certificate and key from %s", f.Certs)
	if caCert == nil {
		logrus.Fatalf("Error loading CA certificate : %v", err)
	}
	caKey, err = certsclient.LoadKeysFromPEM(caKeyPath)
	if err != nil {
		logrus.Fatalf("Error loading CA key from '%s': %v", caKeyPath, err)
	}

	svc := selfsigned.NewSelfSignedCertsService(caCert, caKey)

	err = svc.Start()
	_ = err
}
