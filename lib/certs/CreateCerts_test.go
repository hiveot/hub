package certs_test

import (
	"os"
	"path"
	"testing"

	"github.com/hiveot/hub/lib/certs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testFolder = path.Join(os.TempDir(), "test-certs")
var testSocket = path.Join(testFolder, "certs.socket")

func TestCreateCerts(t *testing.T) {
	// test creating hub certificate
	const serverID = "testService"
	names := []string{"127.0.0.1", "localhost"}

	caCert, caKey, _ := certs.CreateCA("testca", 1)

	serverKeys := certs.CreateECDSAKeys()
	serverCert, err := certs.CreateServerCert(
		serverID, "myou", &serverKeys.PublicKey, names, 0, caCert, caKey)

	serverCertPEM := certs.X509CertToPEM(serverCert)
	// verify service certificate against CA
	err = certs.VerifyCert(serverID, serverCertPEM, caCert)
	assert.NoError(t, err)

}

// test with bad parameters
func TestServerCertBadParms(t *testing.T) {
	const serverID = "testService"
	names := []string{"127.0.0.1", "localhost"}

	caCert, caKey, _ := certs.CreateCA("testca", 1)
	serverKeys := certs.CreateECDSAKeys()

	// Missing CA certificate
	assert.Panics(t, func() {
		_, _ = certs.CreateServerCert(
			serverID, "myou", &serverKeys.PublicKey, names, 0, nil, caKey)
	})

	// missing CA private key
	assert.Panics(t, func() {
		_, _ = certs.CreateServerCert(
			serverID, "myou", &serverKeys.PublicKey, names, 0, caCert, nil)
	})

	// missing service ID
	serverCert, err := certs.CreateServerCert(
		"", "myou", &serverKeys.PublicKey, names, 0, caCert, caKey)
	_ = serverCert
	require.Error(t, err)
	require.Empty(t, serverCert)

	// missing public key
	serverCert, err = certs.CreateServerCert(
		serverID, "myou", nil, names, 0, caCert, caKey)
	require.Error(t, err)
	require.Empty(t, serverCert)

}
