package certs_test

import (
	"testing"

	"github.com/hiveot/hub/lib/keys"

	"github.com/hiveot/hub/lib/certs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCerts(t *testing.T) {
	// test creating hub certificate
	const serverID = "testService"
	const clientID = "testClient"
	names := []string{"127.0.0.1", "localhost"}

	caCert, caKey, _ := certs.CreateCA("testca", 1)

	serverKey := keys.NewKey(keys.KeyTypeECDSA)
	serverCert, err := certs.CreateServerCert(
		serverID, "myou", 0, serverKey, names, caCert, caKey)

	serverCertPEM := certs.X509CertToPEM(serverCert)
	// verify service certificate against CA
	_, err = certs.VerifyCert(serverCertPEM, caCert)
	assert.NoError(t, err)

	// create a server TLS cert
	tlsCert := certs.X509CertToTLS(serverCert, serverKey)
	assert.NotEmpty(t, tlsCert)

	// create a client cert
	clientKey := keys.NewKey(keys.KeyTypeEd25519)
	clientCert, err := certs.CreateClientCert(clientID, "", 0, clientKey, caCert, caKey)
	assert.NoError(t, err)
	assert.NotEmpty(t, clientCert)
}

// test with bad parameters
func TestServerCertBadParms(t *testing.T) {
	const serverID = "testService"
	names := []string{"127.0.0.1", "localhost"}

	caCert, caKey, _ := certs.CreateCA("testca", 1)
	serverKey := keys.NewKey(keys.KeyTypeECDSA)

	// Missing CA certificate
	assert.Panics(t, func() {
		_, _ = certs.CreateServerCert(
			serverID, "myou", 0, serverKey, names, nil, caKey)
	})

	// missing CA private key
	assert.Panics(t, func() {
		_, _ = certs.CreateServerCert(
			serverID, "myou", 0, serverKey, names, caCert, nil)
	})

	// missing service ID
	serverCert, err := certs.CreateServerCert(
		"", "myou", 0, serverKey, names, caCert, caKey)
	_ = serverCert
	require.Error(t, err)
	require.Empty(t, serverCert)

	// missing public key
	serverCert, err = certs.CreateServerCert(
		serverID, "myou", 0, nil, names, caCert, caKey)
	require.Error(t, err)
	require.Empty(t, serverCert)

}
