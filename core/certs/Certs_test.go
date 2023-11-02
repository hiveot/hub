package certs_test

import (
	"crypto/x509"
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/certs/certsapi"
	"github.com/hiveot/hub/core/certs/certsclient"
	"github.com/hiveot/hub/core/certs/service/selfsigned"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient"
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
func StartService() (svc *certsclient.CertsClient, stopFunc func()) {
	adminKP, adminPub := testServer.MsgServer.CreateKeyPair()
	adminID := "admin"

	testClients := []msgserver.ClientAuthInfo{{
		ClientID:   certsapi.ServiceName,
		ClientType: authapi.ClientTypeService,
		//PubKey:       "",
		Role: authapi.ClientRoleService,
	}, {
		ClientID:   adminID,
		ClientType: authapi.ClientTypeUser,
		PubKey:     adminPub,
		Role:       authapi.ClientRoleAdmin,
	}}

	// pre-add service
	err := testServer.AddClients(testClients)
	if err != nil {
		panic(err)
	}
	hc1, err := testServer.AddConnectClient(certsapi.ServiceName, authapi.ClientTypeService, authapi.ClientRoleService)
	if err != nil {
		panic(err)
	}

	certSvc := selfsigned.NewSelfSignedCertsService(
		testServer.CertBundle.CaCert, testServer.CertBundle.CaKey)
	err = certSvc.Start(hc1)
	if err != nil {
		panic(err)
	}

	//--- connect the certs client as admin
	adminToken, err := testServer.MsgServer.CreateToken(testClients[1])
	hc2 := hubclient.NewHubClient(serverURL, adminID, testServer.CertBundle.CaCert, testServer.Core)
	err = hc2.ConnectWithToken(adminKP, adminToken)
	certClient := certsclient.NewCertsClient(hc2)

	return certClient, func() {
		hc2.Disconnect()
		certSvc.Stop()
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

	testServer, err = testenv.StartTestServer(core, false)
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
	keys, _ := certs.CreateECDSAKeys()
	pubKeyPEM, _ := certs.PublicKeyToPEM(&keys.PublicKey)

	deviceCertPEM, caCertPEM, err := svc.CreateDeviceCert(
		deviceID, pubKeyPEM, 1)
	require.NoError(t, err)

	deviceCert, err := certs.X509CertFromPEM(deviceCertPEM)
	require.NoError(t, err)
	require.NotNil(t, deviceCert)
	caCert2, err := certs.X509CertFromPEM(caCertPEM)
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

	keys, _ := certs.CreateECDSAKeys()
	pubKeyPEM, _ := certs.PublicKeyToPEM(&keys.PublicKey)

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
	keys, _ := certs.CreateECDSAKeys()
	pubKeyPEM, _ := certs.PublicKeyToPEM(&keys.PublicKey)

	serviceCertPEM, caCertPEM, err := svc.CreateServiceCert(
		serviceID, pubKeyPEM, names, 0)
	require.NoError(t, err)
	serviceCert, err := certs.X509CertFromPEM(serviceCertPEM)
	require.NoError(t, err)
	caCert2, err := certs.X509CertFromPEM(caCertPEM)
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

	caCert, caKey, _ := certs.CreateCA("Test CA", 1)
	keys, _ := certs.CreateECDSAKeys()
	pubKeyPEM, _ := certs.PublicKeyToPEM(&keys.PublicKey)

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
	svc, cancelFunc := StartService()
	defer cancelFunc()
	keys, _ := certs.CreateECDSAKeys()
	pubKeyPEM, _ := certs.PublicKeyToPEM(&keys.PublicKey)

	userCertPEM, caCertPEM, err := svc.CreateUserCert(userID, pubKeyPEM, 0)
	require.NoError(t, err)

	userCert, err := certs.X509CertFromPEM(userCertPEM)
	require.NoError(t, err)
	require.NotNil(t, userCert)
	caCert2, err := certs.X509CertFromPEM(caCertPEM)
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
