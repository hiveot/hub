package certs

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"golang.org/x/exp/slog"
	"os"
)

// SetupCerts loads CA certificate and create a new server cert
// This creates a new self-signed CA if it doesn't yet exist.
func SetupCerts(certsDir string, caCertFile string, caKeyFile string) (
	serverTLS *tls.Certificate,
	caCert *x509.Certificate,
	caKey *ecdsa.PrivateKey,
) {
	// always create a new server certificate on restart
	var err error

	// 1: Load or create the CA certificate
	//caCertFile := path.Join(certsDir, certs.DefaultCaCertFile)
	//caKeyFile := path.Join(certsDir, certs.DefaultCaKeyFile)
	//serverCertFile := path.Join(certsDir, certs.DefaultServerCertsFile)
	//serverKeyFile := path.Join(certsDir, certs.DefaultServerKeyFile)
	if err2 := os.MkdirAll(certsDir, 0755); err2 != nil && errors.Is(err, os.ErrExist) {
		errMsg := fmt.Errorf("unable to create certs directory '%s': %w", certsDir, err.Error())
		panic(errMsg)
	}
	// TODO: if CA is expired create a new one
	if _, err2 := os.Stat(caKeyFile); err2 == nil {
		slog.Info("loading CA certificate and key")
		// load the CA cert and key
		caKey, err = LoadKeysFromPEM(caKeyFile)
		if err == nil {
			caCert, err = LoadX509CertFromPEM(caCertFile)
		}
		if err != nil {
			panic("unable to load CA certificate: " + err.Error())
		}
	} else {
		slog.Info("creating a self-signed CA certificate and key")
		//
		caCert, caKey, err = CreateCA("hiveot", 365*10)
		if err != nil {
			panic("Unable to create a CA cert: " + err.Error())
		}
		err = SaveKeysToPEM(caKey, caKeyFile)
		if err == nil {
			err = SaveX509CertToPEM(caCert, caCertFile)
		}
	}
	// 3: Always create a new MsgServer cert and private key
	serverKey := CreateECDSAKeys()
	hostName, _ := os.Hostname()
	serverID := "nats-" + hostName
	ou := "hiveot"
	names := []string{"localhost", "127.0.0.1", hostName}
	serverCert, err := CreateServerCert(
		serverID, ou, 365, &serverKey.PublicKey, names, caCert, caKey)
	if err != nil {
		panic("Unable to create a server cert: " + err.Error())
	}
	serverTLS = X509CertToTLS(serverCert, serverKey)
	return serverTLS, caCert, caKey
}