package certs_test

import (
	"crypto/x509"
	auth2 "github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/core/certs"
	"github.com/hiveot/hub/core/certs/certsclient"
	"github.com/hiveot/hub/core/certs/service/selfsigned"
	"github.com/hiveot/hub/core/msgserver"
	certs2 "github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient/hubconnect"
	"github.com/hiveot/hub/lib/testenv"
	"os"
	"path"
	"testing"

	"github.com/hiveot/hub/lib/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var core = "mqtt"
var testDir = path.Join(os.TempDir(), "test-certs")

// the following are set by the testmain
var testServer *testenv.TestServer
var serverURL string

// Factory for creating service instance. Currently the only implementation is selfsigned.
func StartService() (svc certs.ICertService, stopFunc func()) {
	adminKey, adminPub := testServer.MsgServer.CreateKP()
	adminID := "admin"

	testClients := []msgserver.ClientAuthInfo{{
		ClientID:   certs.ServiceName,
		ClientType: auth2.ClientTypeService,
		//PubKey:       "",
		Role: auth2.ClientRoleService,
	}, {
		ClientID:   adminID,
		ClientType: auth2.ClientTypeUser,
		PubKey:     adminPub,
		Role:       auth2.ClientRoleAdmin,
	}}

	// pre-add service
	err := testServer.AddClients(testClients)
	if err != nil {
		panic(err)
	}
	hc1, err := testServer.AddConnectClient(certs.ServiceName, auth2.ClientTypeService, auth2.ClientRoleService)
	if err != nil {
		panic(err)
	}

	certSvc := selfsigned.NewSelfSignedCertsService(
		testServer.CertBundle.CaCert, testServer.CertBundle.CaKey, hc1)
	err = certSvc.Start()
	if err != nil {
		panic(err)
	}

	//--- connect the certs client as admin
	adminToken, err := testServer.MsgServer.CreateToken(testClients[1])
	hc2 := hubconnect.NewHubClient(serverURL, adminID, adminKey, testServer.CertBundle.CaCert, testServer.Core)
	err = hc2.ConnectWithToken(adminToken)
	certClient := certsclient.NewCertsSvcClient(hc2)

	return certClient, func() {
		hc2.Disconnect()
		_ = certSvc.Stop()
		hc1.Disconnect()
	}
}

// TestMain clears the certs folder for clean testing
func TestMain(m *testing.M) {
	var err error
	logging.SetLogging("info", "")
	// clean start
	_ = os.RemoveAll(testDir)
	_ = os.MkdirAll(testDir, 0700)

	testServer, err = testenv.StartTestServer(core)
	serverURL, _, _ = testServer.MsgServer.GetServerURLs()
	if err != nil {
		panic(err)
	}

	res := m.Run()
	testServer.Stop()
	if res == 0 {
		//os.RemoveAll(tempFolder)
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

	svc, cancelFunc := StartService()
	defer cancelFunc()
	keys, _ := certs2.CreateECDSAKeys()
	pubKeyPEM, _ := certs2.PublicKeyToPEM(&keys.PublicKey)

	deviceCertPEM, caCertPEM, err := svc.CreateDeviceCert(
		deviceID, pubKeyPEM, 1)
	require.NoError(t, err)

	deviceCert, err := certs2.X509CertFromPEM(deviceCertPEM)
	require.NoError(t, err)
	require.NotNil(t, deviceCert)
	caCert2, err := certs2.X509CertFromPEM(caCertPEM)
	require.NoError(t, err)
	require.NotNil(t, caCert2)

	// verify certificate
	err = svc.VerifyCert(deviceID, deviceCertPEM)
	assert.NoError(t, err)
	err = svc.VerifyCert("notanid", deviceCertPEM)
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
	svc, cancelFunc := StartService()
	defer cancelFunc()

	keys, _ := certs2.CreateECDSAKeys()
	pubKeyPEM, _ := certs2.PublicKeyToPEM(&keys.PublicKey)

	// missing device ID
	certPEM, _, err := svc.CreateDeviceCert("", pubKeyPEM, 0)
	require.Error(t, err)
	assert.Empty(t, certPEM)

	// missing public key
	certPEM, _, err = svc.CreateDeviceCert(deviceID, "", 1)
	require.Error(t, err)
	assert.Empty(t, certPEM)

}

func TestCreateServiceCert(t *testing.T) {
	// test creating hub certificate
	const serviceID = "testService"
	names := []string{"127.0.0.1", "localhost"}

	svc, cancelFunc := StartService()
	defer cancelFunc()
	keys, _ := certs2.CreateECDSAKeys()
	pubKeyPEM, _ := certs2.PublicKeyToPEM(&keys.PublicKey)

	serviceCertPEM, caCertPEM, err := svc.CreateServiceCert(
		serviceID, pubKeyPEM, names, 0)
	require.NoError(t, err)
	serviceCert, err := certs2.X509CertFromPEM(serviceCertPEM)
	require.NoError(t, err)
	caCert2, err := certs2.X509CertFromPEM(caCertPEM)
	require.NoError(t, err)

	// verify service certificate against CA
	err = svc.VerifyCert(serviceID, serviceCertPEM)
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

	svc, cancelFunc := StartService()
	defer cancelFunc()

	caCert, caKey, _ := certs2.CreateCA("Test CA", 1)
	keys, _ := certs2.CreateECDSAKeys()
	pubKeyPEM, _ := certs2.PublicKeyToPEM(&keys.PublicKey)

	// Bad CA certificate
	badCa := x509.Certificate{}
	assert.Panics(t, func() {
		selfsigned.NewSelfSignedCertsService(&badCa, caKey, nil)
	})

	// missing CA private key
	assert.Panics(t, func() {
		selfsigned.NewSelfSignedCertsService(caCert, nil, nil)
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
	svc, cancelFunc := StartService()
	defer cancelFunc()
	keys, _ := certs2.CreateECDSAKeys()
	pubKeyPEM, _ := certs2.PublicKeyToPEM(&keys.PublicKey)

	userCertPEM, caCertPEM, err := svc.CreateUserCert(userID, pubKeyPEM, 0)
	require.NoError(t, err)

	userCert, err := certs2.X509CertFromPEM(userCertPEM)
	require.NoError(t, err)
	require.NotNil(t, userCert)
	caCert2, err := certs2.X509CertFromPEM(caCertPEM)
	require.NoError(t, err)
	require.NotNil(t, caCert2)

	// verify service certificate against CA
	err = svc.VerifyCert(userID, userCertPEM)
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
