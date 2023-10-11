package provisioning_test

import (
	"context"
	"crypto/md5"
	"fmt"
	"github.com/hiveot/hub/lib/resolver"
	"github.com/hiveot/hub/pkg/certs"
	"github.com/hiveot/hub/pkg/certs/service/selfsigned"
	"github.com/hiveot/hub/pkg/provisioning"
	"github.com/hiveot/hub/pkg/provisioning/capnpclient"
	"github.com/hiveot/hub/pkg/provisioning/capnpserver"
	"github.com/hiveot/hub/pkg/provisioning/service"

	//"github.com/hiveot/hub/lib/resolver"
	"net"
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/certsclient"
	"github.com/hiveot/hub/lib/logging"
)

// when testing using the capnp RPC
var testFolder = path.Join(os.TempDir(), "test-provisioning")
var testSocket = path.Join(testFolder, provisioning.ServiceName+".socket")

const useTestCapnp = true

// provide the capability to create and verify device certificates
// this creates a test instance of the certificate service
func getCertCap() certs.ICerts {
	caCert, caKey, _ := selfsigned.CreateHubCA(1)
	certCap := selfsigned.NewSelfSignedCertsService(caCert, caKey)
	return certCap
}

func newServer(useCapnp bool) (closeFn func()) {
	certCap := getCertCap()
	ctx, cancelFunc := context.WithCancel(context.Background())

	capDeviceCert, _ := certCap.CapDeviceCerts(ctx, "test")
	capVerifyCert, _ := certCap.CapVerifyCerts(ctx, "test")
	svc := service.NewProvisioningService(capDeviceCert, capVerifyCert)

	// optionally test with capnp RPC
	if useCapnp {
		_ = syscall.Unlink(testSocket)
		//lis, _ := net.Listen("unix", testSocket)
		srvListener, _ := net.Listen("tcp", ":0")
		go capnpserver.StartProvisioningCapnpServer(svc, srvListener)

		// Set the marshaller to use with the listener address
		resolver.RegisterCapnpMarshaller[provisioning.IProvisioning](capnpclient.NewProvisioningCapnpClient, srvListener.Addr().String())

		//cl := resolver.GetCapability[provisioning.IProvisioning]()

		// connect the client to the server above
		//clConn, _ := net.Dial("unix", testSocket)
		//clConn, _ := net.Dial("tcp", srvListener.Addr().String())
		//cl := capnpclient.NewProvisioningCapnpClientConnection(ctx, clConn)

		return func() {
			cancelFunc()
			_ = srvListener.Close()
			_ = svc.Stop()
		}
	} else {
		resolver.RegisterService[provisioning.IProvisioning](svc)
	}
	return func() {
		cancelFunc()
		_ = svc.Stop()
	}
}

func TestMain(m *testing.M) {
	logging.SetLogging("info", "")

	_ = os.RemoveAll(testFolder)
	_ = os.MkdirAll(testFolder, 0700)
	resolver.RegisterCapnpMarshaller[provisioning.IProvisioning](capnpclient.NewProvisioningCapnpClient, "")
	//resolver.RegisterHubMarshallers()
	res := m.Run()
	os.Exit(res)
}

// Test starting the provisioning service
func TestStartStop(t *testing.T) {
	// this needs a certificate service capability
	closeFn := newServer(useTestCapnp)
	provCl := resolver.GetCapability[provisioning.IProvisioning]()
	require.NotNil(t, provCl)
	provCl.Release()
	closeFn()
}

func TestAutomaticProvisioning(t *testing.T) {
	const device1ID = "device1"
	const secret1 = "secret1"
	device1Keys := certsclient.CreateECDSAKeys()
	ctx := context.Background()

	secrets := make([]provisioning.OOBSecret, 2)
	secrets[0] = provisioning.OOBSecret{DeviceID: device1ID, OobSecret: secret1}
	secrets[1] = provisioning.OOBSecret{DeviceID: "device2", OobSecret: "secret2"}
	closeFn := newServer(useTestCapnp)
	defer closeFn()
	provCl := resolver.GetCapability[provisioning.IProvisioning]()
	defer provCl.Release()

	capManage, _ := provCl.CapManageProvisioning(ctx, "test")
	defer capManage.Release()
	capProv, _ := provCl.CapRequestProvisioning(ctx, device1ID)
	defer capProv.Release()

	err := capManage.AddOOBSecrets(ctx, secrets)
	assert.NoError(t, err)

	// next, provisioning should succeed
	secret1md5 := fmt.Sprint(md5.Sum([]byte(secret1)))
	pubKeyPEM, err := certsclient.PublicKeyToPEM(&device1Keys.PublicKey)
	assert.NoError(t, err)
	status, err := capProv.SubmitProvisioningRequest(
		ctx, "device1", secret1md5, pubKeyPEM)
	require.NoError(t, err)
	assert.Equal(t, device1ID, status.DeviceID)
	assert.NotEmpty(t, status.ClientCertPEM)
	assert.NotEmpty(t, status.CaCertPEM)
	assert.False(t, status.Pending)
	assert.NotEmpty(t, status.RequestTime)

	// provisioned device should show up in the list of approved devices
	approved, err := capManage.GetApprovedRequests(ctx)
	assert.NoError(t, err)
	require.True(t, len(approved) > 0)
	assert.Equal(t, device1ID, approved[0].DeviceID)
}

func TestAutomaticProvisioningBadParameters(t *testing.T) {
	const device1ID = "device1"
	const secret1 = "secret1"
	ctx := context.Background()
	device1Keys := certsclient.CreateECDSAKeys()
	pubKeyPEM, _ := certsclient.PublicKeyToPEM(&device1Keys.PublicKey)
	secrets := make([]provisioning.OOBSecret, 1)
	secrets[0] = provisioning.OOBSecret{DeviceID: device1ID, OobSecret: secret1}

	closeFn := newServer(useTestCapnp)
	defer closeFn()
	provCl := resolver.GetCapability[provisioning.IProvisioning]()
	defer provCl.Release()
	capProv, _ := provCl.CapRequestProvisioning(ctx, device1ID)
	defer capProv.Release()
	capManage, _ := provCl.CapManageProvisioning(ctx, "test")
	defer capManage.Release()

	// add a secret for testing
	err := capManage.AddOOBSecrets(context.Background(), secrets)
	assert.NoError(t, err)

	// test missing deviceID
	_, err = capProv.SubmitProvisioningRequest(
		ctx, "", "", pubKeyPEM)
	require.Error(t, err)

	// test missing public key
	_, err = capProv.SubmitProvisioningRequest(
		ctx, device1ID, "", "")
	require.Error(t, err)

	// test bad public key
	_, err = capProv.SubmitProvisioningRequest(
		ctx, device1ID, "", "badpubkey")
	require.Error(t, err)

	// test bad secret. This should return an error and pending status
	status, err := capProv.SubmitProvisioningRequest(
		ctx, device1ID, "badsecret", pubKeyPEM)
	require.NoError(t, err)
	require.True(t, status.Pending)
}

func TestManualProvisioning(t *testing.T) {
	const device1ID = "device1"

	// setup
	device1Keys := certsclient.CreateECDSAKeys()
	ctx := context.Background()
	closeFn := newServer(useTestCapnp)
	provCl := resolver.GetCapability[provisioning.IProvisioning]()
	defer provCl.Release()
	defer closeFn()

	capProv, _ := provCl.CapRequestProvisioning(ctx, device1ID)
	defer capProv.Release()
	capManage, _ := provCl.CapManageProvisioning(ctx, "test")
	defer capManage.Release()

	// Stage 1: request provisioning without a secret.
	pubKeyPEM, _ := certsclient.PublicKeyToPEM(&device1Keys.PublicKey)
	status, err := capProv.SubmitProvisioningRequest(
		ctx, device1ID, "", pubKeyPEM)
	// This should return a 'pending' status
	require.NoError(t, err)
	assert.Equal(t, device1ID, status.DeviceID)
	assert.Empty(t, status.ClientCertPEM)
	//assert.NotEmpty(t, status.CaCertPEM)
	assert.True(t, status.Pending)
	assert.NotEmpty(t, status.RequestTime)

	// provisioned device should be added to the list of pending devices
	pendingList, err := capManage.GetPendingRequests(ctx)
	require.True(t, len(pendingList) > 0)
	assert.Equal(t, device1ID, pendingList[0].DeviceID)
	approvedList, err := capManage.GetApprovedRequests(ctx)
	assert.NoError(t, err)
	assert.True(t, len(approvedList) == 0)

	// Stage 2: approve the request
	err = capManage.ApproveRequest(ctx, device1ID)
	assert.NoError(t, err)

	// provisioning request should now succeed
	status, err = capProv.SubmitProvisioningRequest(
		ctx, "device1", "", pubKeyPEM)
	// This should now succeed
	require.NoError(t, err)
	require.False(t, status.Pending)
	require.NotEmpty(t, status.ClientCertPEM)
	require.NotEmpty(t, status.CaCertPEM)

	// provisioned device should now show up in the list of approved devices
	approvedList, err = capManage.GetApprovedRequests(ctx)
	assert.NoError(t, err)
	require.True(t, len(approvedList) > 0)
	assert.Equal(t, device1ID, approvedList[0].DeviceID)

	pendingList, err = capManage.GetPendingRequests(ctx)
	require.True(t, len(pendingList) == 0)
}

func TestRefreshProvisioning(t *testing.T) {

	const device1ID = "device1"
	const secret1 = "secret1"
	//setup and generate a certificate
	device1Keys := certsclient.CreateECDSAKeys()
	pubKeyPEM, _ := certsclient.PublicKeyToPEM(&device1Keys.PublicKey)
	secrets := make([]provisioning.OOBSecret, 1)
	secrets[0] = provisioning.OOBSecret{DeviceID: device1ID, OobSecret: secret1}
	ctx := context.Background()

	// request provisioning with a valid secret.
	closeFn := newServer(useTestCapnp)
	defer closeFn()
	provCl := resolver.GetCapability[provisioning.IProvisioning]()
	defer provCl.Release()

	capProv, _ := provCl.CapRequestProvisioning(ctx, device1ID)
	defer capProv.Release()
	capRefresh, _ := provCl.CapRefreshProvisioning(ctx, device1ID)
	defer capRefresh.Release()
	capManage, _ := provCl.CapManageProvisioning(ctx, "test")
	defer capManage.Release()

	// obtain a certificate
	err := capManage.AddOOBSecrets(context.Background(), secrets)
	assert.NoError(t, err)
	secret1md5 := fmt.Sprint(md5.Sum([]byte(secret1)))
	status, err := capProv.SubmitProvisioningRequest(
		context.Background(), device1ID, secret1md5, pubKeyPEM)
	require.NoError(t, err)
	assert.NotEmpty(t, status.ClientCertPEM)

	// refresh
	status2, err := capRefresh.RefreshDeviceCert(
		context.Background(), status.ClientCertPEM)
	// This should succeed
	require.NoError(t, err)
	require.False(t, status2.Pending)
	require.NotEmpty(t, status2.ClientCertPEM)
	require.NotEmpty(t, status2.CaCertPEM)

	// refresh with bad certificate should fail
	_, err = capRefresh.RefreshDeviceCert(
		context.Background(), "bad certificate")
	require.Error(t, err)
}
