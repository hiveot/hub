package certs_test

import (
	"crypto/x509"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	authz "github.com/hiveot/hub/runtime/authz/api"
	"github.com/hiveot/hub/services/certs/certsapi"
	"github.com/hiveot/hub/services/certs/certsclient"
	"github.com/hiveot/hub/services/certs/service/selfsigned"
	"github.com/hiveot/hub/wot/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

var ts *testenv.TestServer

const agentUsesWSS = true

// Factory for creating service instance. Currently the only implementation is selfsigned.
func startService() (cl *certsclient.CertsClient, stopFunc func()) {
	ts = testenv.StartTestServer(true)

	// the service needs a server connection
	// FIXME: certs service need to register their service
	hc1, token1 := ts.AddConnectService(certsapi.CertsAdminAgentID)
	_ = token1

	//storeDir := path.Join(ts.TestDir, "test-certs")

	svc := selfsigned.NewSelfSignedCertsService(ts.Certs.CaCert, ts.Certs.CaKey)
	err := svc.Start(hc1)
	if err != nil {
		panic(err)
	}

	//--- connect the certs client as admin
	hc2, _ := ts.AddConnectConsumer("admin1", authz.ClientRoleAdmin)
	certAdmin := certsclient.NewCertsClient(hc2)

	return certAdmin, func() {
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

func TestCreateDeviceCert(t *testing.T) {
	t.Log("--- TestCreateDeviceCert ---")
	agentID := "agent1"
	agThingID := "thing1"

	certAdmin, stopFunc := startService()
	defer stopFunc()

	k := keys.NewKey(keys.KeyTypeECDSA)
	pubKeyPEM := k.ExportPublic()

	// a TD is needed first
	td1 := td.NewTD(agThingID, "Title", vocab.ThingSensorMulti)
	ts.AddTD(agentID, td1)

	deviceCertPEM, caCertPEM, err := certAdmin.CreateDeviceCert(
		agentID, pubKeyPEM, 1)
	require.NoError(t, err)

	deviceCert, err := certs.X509CertFromPEM(deviceCertPEM)
	require.NoError(t, err)
	require.NotNil(t, deviceCert)
	caCert2, err := certs.X509CertFromPEM(caCertPEM)
	require.NoError(t, err)
	require.NotNil(t, caCert2)

	// verify certificate
	err = certAdmin.VerifyCert(agentID, deviceCertPEM)
	assert.NoError(t, err)
	err = certAdmin.VerifyCert("notanid", deviceCertPEM)
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
	t.Log("--- TestCreateDeviceCert ended ---")
}

// test device cert with bad parameters
func TestDeviceCertBadParms(t *testing.T) {
	t.Log("--- TestDeviceCertBadParms ---")
	agentID := "agent1"
	agThingID := "thing1"

	// test creating hub certificate
	certAdmin, stopFunc := startService()
	defer stopFunc()
	// a TD is needed first
	td1 := td.NewTD(agThingID, "Title", vocab.ThingSensorMulti)
	ts.AddTD(agentID, td1)

	k := keys.NewKey(keys.KeyTypeECDSA)
	pubKeyPEM := k.ExportPublic()

	// missing device ID
	certPEM, _, err := certAdmin.CreateDeviceCert("", pubKeyPEM, 0)
	require.Error(t, err)
	assert.Empty(t, certPEM)

	// missing public key
	certPEM, _, err = certAdmin.CreateDeviceCert(agentID, "", 1)
	require.Error(t, err)
	assert.Empty(t, certPEM)
	t.Log("--- TestDeviceCertBadParms ended ---")

}

func TestCreateServiceCert(t *testing.T) {
	t.Log("--- TestCreateServiceCert ---")
	// test creating hub certificate
	const serviceID = "testService"
	names := []string{"127.0.0.1", "localhost"}

	certAdmin, stopFunc := startService()
	defer stopFunc()
	k := keys.NewKey(keys.KeyTypeECDSA)
	pubKeyPEM := k.ExportPublic()

	serviceCertPEM, caCertPEM, err := certAdmin.CreateServiceCert(
		serviceID, pubKeyPEM, names, 0)
	require.NoError(t, err)
	serviceCert, err := certs.X509CertFromPEM(serviceCertPEM)
	require.NoError(t, err)
	caCert2, err := certs.X509CertFromPEM(caCertPEM)
	require.NoError(t, err)

	// verify service certificate against CA
	err = certAdmin.VerifyCert(serviceID, serviceCertPEM)
	assert.NoError(t, err)

	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCert2)
	opts := x509.VerifyOptions{
		Roots:     caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	_, err = serviceCert.Verify(opts)
	assert.NoError(t, err)
	t.Log("--- TestCreateServiceCert ended ---")
}

// test with bad parameters
func TestServiceCertBadParms(t *testing.T) {
	t.Log("--- TestServiceCertBadParms ---")
	const serviceID = "testService"
	hostnames := []string{"127.0.0.1"}

	certAdmin, stopFunc := startService()
	defer stopFunc()

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
	//certAdmin := selfsigned.NewSelfSignedCertsService(caCert, caKey, nil)

	serviceCertPEM, _, err := certAdmin.CreateServiceCert(
		"", pubKeyPEM, hostnames, 1)

	require.Error(t, err)
	require.Empty(t, serviceCertPEM)

	// missing public key
	serviceCertPEM, _, err = certAdmin.CreateServiceCert(
		serviceID, "", hostnames, 1)
	require.Error(t, err)
	require.Empty(t, serviceCertPEM)
	t.Log("--- TestServiceCertBadParms ended ---")

}

func TestCreateUserCert(t *testing.T) {
	t.Log("--- TestCreateUserCert ---")
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
	t.Log("--- TestCreateUserCert ended ---")
}
