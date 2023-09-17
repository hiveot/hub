// Package testenv with managing certificates for testing
package certs

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
)

const ServerAddress = "127.0.0.1"
const TestServerID = "server1"
const TestClientID = "client1"

// TestCertBundle creates a set of CA, server and client certificates intended for testing
type TestCertBundle struct {
	CaCert *x509.Certificate
	CaKey  *ecdsa.PrivateKey

	// server certificate
	ServerKey  *ecdsa.PrivateKey
	ServerCert *tls.Certificate

	// client cert auth
	ClientKey  *ecdsa.PrivateKey
	ClientCert *tls.Certificate
}

// CreateTestCertBundle creates a bundle of ca, server certificates and keys for testing.
// The server cert is valid for the 127.0.0.1, localhost and os.hostname.
func CreateTestCertBundle() TestCertBundle {
	certBundle := TestCertBundle{}
	// Setup CA and server TLS certificates
	certBundle.CaCert, certBundle.CaKey, _ = CreateCA("testing", 1)
	certBundle.ServerKey, _ = CreateECDSAKeys()
	certBundle.ClientKey, _ = CreateECDSAKeys()

	names := []string{ServerAddress, "localhost"}
	serverCert, err := CreateServerCert(
		TestServerID, "server", 1,
		&certBundle.ServerKey.PublicKey,
		names,
		certBundle.CaCert, certBundle.CaKey)
	if err != nil {
		panic("unable to create server cert: " + err.Error())
	}
	certBundle.ServerCert = X509CertToTLS(serverCert, certBundle.ServerKey)

	clientCert, err := CreateClientCert(TestClientID, "service", 1,
		&certBundle.ClientKey.PublicKey,
		certBundle.CaCert, certBundle.CaKey)
	if err != nil {
		panic("unable to create client cert: " + err.Error())
	}
	certBundle.ClientCert = X509CertToTLS(clientCert, certBundle.ClientKey)

	return certBundle
}
