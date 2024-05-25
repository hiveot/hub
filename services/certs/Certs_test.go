package certs_test

import (
	"crypto/x509"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/services/certs/certsapi"
	"github.com/hiveot/hub/services/certs/certsclient"
	"github.com/hiveot/hub/services/certs/service/selfsigned"
	"os"
	"path"
	"testing"

	"github.com/hiveot/hub/lib/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testDir = path.Join(os.TempDir(), "test-certs")

// the following are set by the testmain
var serverURL string

var ts *testenv.TestServer

// Factory for creating service instance. Currently the only implementation is selfsigned.
func startService() (cl *certsclient.CertsClient, stopFunc func()) {
	ts = testenv.StartTestServer(true)

	// the service needs a server connection
	hc1, token1 := ts.AddConnectAgent(api.ClientTypeService, certsapi.AgentID)
	_ = token1

	//storeDir := path.Join(ts.TestDir, "test-certs")

	svc := selfsigned.NewSelfSignedCertsService(ts.Certs.CaCert, ts.Certs.CaKey)
	err := svc.Start(hc1)
	if err != nil {
		panic(err)
	}

	//--- connect the certs client as admin
	hc2, _ := ts.AddConnectUser("admin1", api.ClientRoleAdmin)
	certClient := certsclient.NewCertsClient(hc2)

	return certClient, func() {
		hc2.Disconnect()
		hc1.Disconnect()
		svc.Stop()
		ts.Stop()
	}
}

// TestMain clears the certs folder for clean testing
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")

	res := m.Run()
	if res == 0 {
	}
	os.Exit(res)
}

//func TestStartStop(t *testing.T) {
//	svc, cancelFunc := StartPlugin()
//	defer cancelFunc()
//	require.NotNil(t, svc)
//}

func TestCreateDeviceCert(t *testing.T) {
	deviceID := "device1"

	cl, cancelFunc := startService()
	defer cancelFunc()
	k := keys.NewKey(keys.KeyTypeECDSA)
	pubKeyPEM := k.ExportPublic()

	deviceCertPEM, caCertPEM, err := cl.CreateDeviceCert(
		deviceID, pubKeyPEM, 1)
	require.NoError(t, err)

	deviceCert, err := certs.X509CertFromPEM(deviceCertPEM)
	require.NoError(t, err)
	require.NotNil(t, deviceCert)
	caCert2, err := certs.X509CertFromPEM(caCertPEM)
	require.NoError(t, err)
	require.NotNil(t, caCert2)

	// verify certificate
	err = cl.VerifyCert(deviceID, deviceCertPEM)
	assert.NoError(t, err)
	err = cl.VerifyCert("notanid", deviceCertPEM)
	assert.Error(t, err)

	// verify certificate against CA
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCert2)
	opts := x509.VerifyOptions{
		Roots:     caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	_, err = deviceCert.Verify(opts)
	assert.NoError(t, err)
}

// test device cert with bad parameters
func TestDeviceCertBadParms(t *testing.T) {
	deviceID := "device1"

	// test creating hub certificate
	cl, cancelFunc := startService()
	defer cancelFunc()

	k := keys.NewKey(keys.KeyTypeECDSA)
	pubKeyPEM := k.ExportPublic()

	// missing device ID
	certPEM, _, err := cl.CreateDeviceCert("", pubKeyPEM, 0)
	require.Error(t, err)
	assert.Empty(t, certPEM)

	// missing public key
	certPEM, _, err = cl.CreateDeviceCert(deviceID, "", 1)
	require.Error(t, err)
	assert.Empty(t, certPEM)

}

func TestCreateServiceCert(t *testing.T) {
	// test creating hub certificate
	const serviceID = "testService"
	names := []string{"127.0.0.1", "localhost"}

	cl, cancelFunc := startService()
	defer cancelFunc()
	k := keys.NewKey(keys.KeyTypeECDSA)
	pubKeyPEM := k.ExportPublic()

	serviceCertPEM, caCertPEM, err := cl.CreateServiceCert(
		serviceID, pubKeyPEM, names, 0)
	require.NoError(t, err)
	serviceCert, err := certs.X509CertFromPEM(serviceCertPEM)
	require.NoError(t, err)
	caCert2, err := certs.X509CertFromPEM(caCertPEM)
	require.NoError(t, err)

	// verify service certificate against CA
	err = cl.VerifyCert(serviceID, serviceCertPEM)
	assert.NoError(t, err)

	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCert2)
	opts := x509.VerifyOptions{
		Roots:     caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	_, err = serviceCert.Verify(opts)
	assert.NoError(t, err)
}

// test with bad parameters
func TestServiceCertBadParms(t *testing.T) {
	const serviceID = "testService"
	hostnames := []string{"127.0.0.1"}

	svc, cancelFunc := startService()
	defer cancelFunc()

	caCert, caKey, _ := certs.CreateCA("Test CA", 1)
	k := keys.NewKey(keys.KeyTypeECDSA)
	pubKeyPEM := k.ExportPublic()

	// Bad CA certificate
	badCa := x509.Certificate{}
	assert.Panics(t, func() {
		selfsigned.NewSelfSignedCertsService(&badCa, caKey)
	})

	// missing CA private key
	assert.Panics(t, func() {
		selfsigned.NewSelfSignedCertsService(caCert, nil)
	})

	// missing service ID
	//svc := selfsigned.NewSelfSignedCertsService(caCert, caKey, nil)

	serviceCertPEM, _, err := svc.CreateServiceCert(
		"", pubKeyPEM, hostnames, 1)

	require.Error(t, err)
	require.Empty(t, serviceCertPEM)

	// missing public key
	serviceCertPEM, _, err = svc.CreateServiceCert(
		serviceID, "", hostnames, 1)
	require.Error(t, err)
	require.Empty(t, serviceCertPEM)

}

func TestCreateUserCert(t *testing.T) {
	userID := "bob"
	// test creating hub certificate
	cl, cancelFunc := startService()
	defer cancelFunc()
	k := keys.NewKey(keys.KeyTypeECDSA)
	pubKeyPEM := k.ExportPublic()

	userCertPEM, caCertPEM, err := cl.CreateUserCert(userID, pubKeyPEM, 0)
	require.NoError(t, err)

	userCert, err := certs.X509CertFromPEM(userCertPEM)
	require.NoError(t, err)
	require.NotNil(t, userCert)
	caCert2, err := certs.X509CertFromPEM(caCertPEM)
	require.NoError(t, err)
	require.NotNil(t, caCert2)

	// verify service certificate against CA
	err = cl.VerifyCert(userID, userCertPEM)
	assert.NoError(t, err)

	// verify client certificate against CA
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCert2)
	opts := x509.VerifyOptions{
		Roots:     caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	_, err = userCert.Verify(opts)
	assert.NoError(t, err)
}
