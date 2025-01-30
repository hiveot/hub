package idprov_test

import (
	"fmt"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/testenv"
	authn "github.com/hiveot/hub/runtime/authn/api"
	authz "github.com/hiveot/hub/runtime/authz/api"
	"github.com/hiveot/hub/services/idprov/idprovapi"
	"github.com/hiveot/hub/services/idprov/idprovclient"
	"github.com/hiveot/hub/services/idprov/service"
	"github.com/hiveot/hub/transports/clients"
	"github.com/hiveot/hub/transports/messaging"
	"github.com/hiveot/hub/transports/tputils/tlsclient"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/logging"
)

// when testing using the capnp RPC
var testPort = 23001

// the following are set by newIdProvService
var ts *testenv.TestServer

// Create a new store, delete if it already exists
func newIdProvService() (svc *service.IdProvService, co *messaging.Consumer, stopFn func()) {

	ts = testenv.StartTestServer(true)
	agentConn, token1 := ts.AddConnectService(idprovapi.AgentID)
	_ = token1
	svc = service.NewIdProvService(testPort, ts.Certs.ServerCert, ts.Certs.CaCert)
	err := svc.Start(agentConn)
	if err != nil {
		panic("failed starting service: " + err.Error())
	}

	//ag := service.StartIdProvAgent(svc.ManageIdProv, ag)
	//_ = ag

	// create an end user client for testing
	co, token2 := ts.AddConnectConsumer("test-client", authz.ClientRoleManager)
	_ = token2
	if err != nil {
		panic("can't connect operator")
	}
	return svc, co, func() {
		co.Disconnect()
		svc.Stop()
		agentConn.Disconnect()
		ts.Stop()
	}
}

func TestMain(m *testing.M) {
	logging.SetLogging("info", "")

	res := m.Run()

	os.Exit(res)
}

// Test starting the provisioning service
func TestStartStop(t *testing.T) {
	t.Log("TestStartStop")
	svc, hc, stopFn := newIdProvService()
	_ = svc
	_ = hc
	time.Sleep(time.Second)
	stopFn()
}

func TestAutomaticProvisioning(t *testing.T) {
	t.Log("TestAutomaticProvisioning")
	const device1ID = "device1"
	const device2ID = "device2"

	svc, co, stopFn := newIdProvService()
	_ = svc
	defer stopFn()

	mngCl := idprovclient.NewIdProvManageClient(co)
	device1KP := keys.NewEcdsaKey()
	device2KP := keys.NewEcdsaKey()

	approvedDevices := make([]idprovapi.PreApprovedClient, 2)
	approvedDevices[0] = idprovapi.PreApprovedClient{
		ClientID: device1ID, ClientType: authn.ClientTypeAgent, PubKey: device1KP.ExportPublic()}
	approvedDevices[1] = idprovapi.PreApprovedClient{
		ClientID: device2ID, ClientType: authn.ClientTypeAgent, PubKey: device2KP.ExportPublic()}

	err := mngCl.PreApproveDevices(approvedDevices)
	assert.NoError(t, err)

	// next, provisioning should succeed
	hostPort := fmt.Sprintf("localhost:%d", testPort)
	tlsClient := tlsclient.NewTLSClient(hostPort, nil, ts.Certs.CaCert, 0)
	//tlsClient.ConnectNoAuth()
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
	srvURL := ts.GetServerURL(authn.ClientTypeAgent)

	agConn, err := clients.NewClient(srvURL, device1ID, ts.Certs.CaCert, ts.GetForm, 0)
	require.NoError(t, err)
	err = agConn.ConnectWithToken(token1)
	require.NoError(t, err)
	ag1 := messaging.NewAgent(agConn, nil, nil, nil, 0)
	//ag1.SetRetryConnect(false)
	ag1.Disconnect()
}

func TestAutomaticProvisioningBadParameters(t *testing.T) {
	t.Log("TestAutomaticProvisioningBadParameters")
	const device1ID = "device1"

	device1Keys := keys.NewKey(keys.KeyTypeEd25519)
	device1PubPEM := device1Keys.ExportPublic()

	svc, co, stopFn := newIdProvService()
	_ = svc
	defer stopFn()
	mngCl := idprovclient.NewIdProvManageClient(co)

	approvedDevices := make([]idprovapi.PreApprovedClient, 2)
	approvedDevices[0] = idprovapi.PreApprovedClient{
		ClientID: device1ID, PubKey: device1PubPEM}

	err := mngCl.PreApproveDevices(approvedDevices)
	assert.NoError(t, err)

	// test missing deviceID
	hostPort := "localhost:9002"
	tlsClient := tlsclient.NewTLSClient(hostPort, nil, ts.Certs.CaCert, 0)
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
	t.Log("TestManualProvisioning")
	const device1ID = "device1"
	device1Keys := keys.NewKey(keys.KeyTypeEd25519)
	device1PubPEM := device1Keys.ExportPublic()

	svc, co, stopFn := newIdProvService()
	mngCl := idprovclient.NewIdProvManageClient(co)
	_ = svc
	defer stopFn()

	// request provisioning
	hostPort := fmt.Sprintf("localhost:%d", testPort)
	tlsClient := tlsclient.NewTLSClient(hostPort, nil, ts.Certs.CaCert, 0)
	//tlsClient.ConnectNoAuth()
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
	err = mngCl.ApproveRequest(device1ID, authn.ClientTypeAgent)
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
