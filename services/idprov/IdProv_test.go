package idprov_test

import (
	"fmt"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/services/idprov/idprovapi"
	"github.com/hiveot/hub/services/idprov/idprovclient"
	"github.com/hiveot/hub/services/idprov/service"

	"github.com/hiveot/hub/lib/hubclient/connect"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/lib/tlsclient"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/logging"
)

// when testing using the capnp RPC
var testFolder = path.Join(os.TempDir(), "test-provisioning")
var testPort = uint(23001)

const core = "mqtt"

// the following are set by the testmain
var testServer *testenv.TestServer

// Create a new store, delete if it already exists
func newIdProvService() (
	svc *service.IdProvService,
	mngCl *idprovclient.ManageIdProvClient,
	stopFn func()) {

	hc, token1 := testServer.AddConnectClient(api.ClientTypeService, idprovapi.AgentID)
	_ = token1
	svc = service.NewIdProvService(testPort, testServer.Certs.ServerCert, testServer.Certs.CaCert)
	err := svc.Start(hc)
	if err != nil {
		panic("failed starting service: " + err.Error())
	}

	// create an end user client for testing
	hc2, token2 := testServer.AddConnectClient(api.ClientTypeUser, "test-client")
	_ = token2
	if err != nil {
		panic("can't connect operator")
	}
	mngCl = idprovclient.NewIdProvManageClient(hc2)

	return svc, mngCl, func() {
		hc2.Disconnect()
		svc.Stop()
		hc.Disconnect()
	}
}

func TestMain(m *testing.M) {
	logging.SetLogging("info", "")

	_ = os.RemoveAll(testFolder)
	_ = os.MkdirAll(testFolder, 0700)

	testServer, _ = testenv.StartTestServer(core, true)

	res := m.Run()
	os.Exit(res)
}

// Test starting the provisioning service
func TestStartStop(t *testing.T) {
	svc, mngCl, stopFn := newIdProvService()
	_ = svc
	_ = mngCl
	time.Sleep(time.Second)
	stopFn()
}

func TestAutomaticProvisioning(t *testing.T) {
	const device1ID = "device1"
	const device2ID = "device2"
	device1KP := testServer.MsgServer.CreateKeyPair()
	device2KP := testServer.MsgServer.CreateKeyPair()

	svc, mngCl, stopFn := newIdProvService()
	_ = svc
	defer stopFn()

	approvedDevices := make([]idprovapi.PreApprovedClient, 2)
	approvedDevices[0] = idprovapi.PreApprovedClient{
		ClientID: device1ID, ClientType: authapi.ClientTypeDevice, PubKey: device1KP.ExportPublic()}
	approvedDevices[1] = idprovapi.PreApprovedClient{
		ClientID: device2ID, ClientType: authapi.ClientTypeDevice, PubKey: device2KP.ExportPublic()}

	err := mngCl.PreApproveDevices(approvedDevices)
	assert.NoError(t, err)

	// next, provisioning should succeed
	idProvServerURL := fmt.Sprintf("localhost:%d", testPort)
	tlsClient := tlsclient.NewTLSClient(idProvServerURL, testServer.CertBundle.CaCert)
	tlsClient.ConnectNoAuth()
	status, token1, err := idprovclient.SubmitIdProvRequest(
		device1ID, device1KP.ExportPublic(), "", tlsClient)
	require.NoError(t, err)

	assert.Equal(t, device1ID, status.ClientID)
	assert.NotEmpty(t, token1)
	assert.False(t, status.Pending)
	assert.NotEmpty(t, status.ApprovedMSE)

	// provisioned device should show up in the list of approved devices
	approved, err := mngCl.GetRequests(true, true, true)
	assert.NoError(t, err)
	require.True(t, len(approved) > 0)
	hasDevice1 := false
	for _, a := range approved {
		if a.ClientID == device1ID {
			hasDevice1 = true
		}
	}
	assert.True(t, hasDevice1)

	// token should be used to connect
	srvURL, _, _ := testServer.MsgServer.GetServerURLs()
	hc1 := connect.NewHubClient(srvURL, device1ID, testServer.CertBundle.CaCert, core)
	hc1.SetRetryConnect(false)
	err = hc1.ConnectWithToken(device1KP, token1)
	require.NoError(t, err)
	hc1.Disconnect()
}

func TestAutomaticProvisioningBadParameters(t *testing.T) {
	const device1ID = "device1"

	device1Keys := keys.NewKey(keys.KeyTypeECDSA)
	device1PubPEM := device1Keys.ExportPublic()

	svc, mngCl, stopFn := newIdProvService()
	_ = svc
	defer stopFn()

	approvedDevices := make([]idprovapi.PreApprovedClient, 2)
	approvedDevices[0] = idprovapi.PreApprovedClient{
		ClientID: device1ID, PubKey: device1PubPEM}

	err := mngCl.PreApproveDevices(approvedDevices)
	assert.NoError(t, err)

	// test missing deviceID
	idProvServerURL := "localhost:9002"
	tlsClient := tlsclient.NewTLSClient(idProvServerURL, testServer.CertBundle.CaCert)
	status, tokenEnc, err := idprovclient.SubmitIdProvRequest(
		"", device1PubPEM, "", tlsClient)
	assert.Error(t, err)
	assert.Empty(t, status)
	assert.Empty(t, tokenEnc)

	// test missing public key
	status, tokenEnc, err = idprovclient.SubmitIdProvRequest(
		device1ID, "", "", tlsClient)
	assert.Error(t, err)

	// test bad public key
	status, tokenEnc, err = idprovclient.SubmitIdProvRequest(
		device1ID, "badpubkey", "", tlsClient)
	require.Error(t, err)
}

func TestManualProvisioning(t *testing.T) {
	const device1ID = "device1"
	device1Keys := keys.NewKey(keys.KeyTypeECDSA)
	device1PubPEM := device1Keys.ExportPublic()

	svc, mngCl, stopFn := newIdProvService()
	_ = svc
	defer stopFn()

	// request provisioning
	idProvServerAddr := fmt.Sprintf("localhost:%d", testPort)
	tlsClient := tlsclient.NewTLSClient(idProvServerAddr, testServer.CertBundle.CaCert)
	tlsClient.ConnectNoAuth()
	status, token, err := idprovclient.SubmitIdProvRequest(device1ID, device1PubPEM, "", tlsClient)
	require.NoError(t, err)

	assert.Equal(t, device1ID, status.ClientID)
	assert.Empty(t, token)
	assert.True(t, status.Pending)
	assert.NotEmpty(t, status.ReceivedMSE)

	// provisioned device should be added to the list of pending devices
	pendingList, err := mngCl.GetRequests(true, false, false)
	require.True(t, len(pendingList) > 0)
	assert.Equal(t, device1ID, pendingList[0].ClientID)
	approvedList, err := mngCl.GetRequests(false, true, false)
	assert.NoError(t, err)
	assert.True(t, len(approvedList) == 0)

	// Stage 2: approve the request
	err = mngCl.ApproveRequest(device1ID, authapi.ClientTypeDevice)
	assert.NoError(t, err)

	// provisioning request should now succeed
	status, token, err = idprovclient.SubmitIdProvRequest(
		device1ID, device1PubPEM, "", tlsClient)
	require.NoError(t, err)
	require.False(t, status.Pending)
	require.NotEmpty(t, status.ReceivedMSE)
	require.NotEmpty(t, status.ApprovedMSE)
	require.Empty(t, status.RejectedMSE)

	// provisioned device should now show up in the list of approved devices
	approvedList, err = mngCl.GetRequests(false, true, false)
	assert.NoError(t, err)
	require.True(t, len(approvedList) > 0)
	assert.Equal(t, device1ID, approvedList[0].ClientID)

	pendingList, err = mngCl.GetRequests(true, false, false)
	require.True(t, len(pendingList) == 0)
}
