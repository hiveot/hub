package auth_test

import (
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/auth/authclient"
	"github.com/hiveot/hub/core/auth/authservice"
	"github.com/hiveot/hub/core/auth/config"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/nats-io/nkeys"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/logging"
)

var core = "mqtt"
var certBundle certs.TestCertBundle
var testDir = path.Join(os.TempDir(), "test-authn")
var authConfig config.AuthConfig

// the following are set by the testmain
var testServer *testenv.TestServer

// add new user to test with
func addNewUser(userID string, displayName string, pass string, mng *authclient.ManageClients) (token string, key nkeys.KeyPair, err error) {
	userKey, _ := nkeys.CreateUser()
	userKeyPub, _ := userKey.PublicKey()
	// FIXME: must set a password in order to be able to update it later
	userToken, err := mng.AddUser(userID, displayName, pass, userKeyPub, authapi.ClientRoleViewer)
	return userToken, userKey, err
}

// launch the authn service and return a client for using and managing it.
// the messaging server is already running (see TestMain)
func startTestAuthnService() (authnSvc *authservice.AuthService, mng *authclient.ManageClients, stopFn func(), err error) {
	// the password file to use
	passwordFile := path.Join(testDir, "test.passwd")

	// TODO: put this in a test environment
	_ = os.Remove(passwordFile)
	authConfig = config.AuthConfig{}
	_ = authConfig.Setup(testDir, testDir)
	authConfig.PasswordFile = passwordFile
	authConfig.DeviceTokenValidityDays = 10
	authConfig.Encryption = authapi.PWHASH_BCRYPT // nats requires bcrypt

	authnSvc, err = authservice.StartAuthService(authConfig, testServer.MsgServer, testServer.CertBundle.CaCert)
	if err != nil {
		panic("cant start test authn service: " + err.Error())
	}

	//--- connect the authn management client for managing clients
	serverURL, _, _ := testServer.MsgServer.GetServerURLs()
	hc2 := hubclient.NewHubClient(serverURL, "auth-test-client", testServer.CertBundle.CaCert, core)
	clKP := hc2.CreateKeyPair()

	args := authapi.AddUserArgs{
		UserID:      "auth-test-client",
		DisplayName: "auth test client",
		PubKey:      clKP.ExportPublic(),
		Role:        authapi.ClientRoleAdmin,
	}
	ctx := hubclient.ServiceContext{SenderID: "test-client"}
	resp, err := authnSvc.MngClients.AddUser(ctx, args)
	hc2.SetRetryConnect(false)
	err = hc2.ConnectWithToken(clKP, resp.Token)

	if err != nil {
		panic(err)
	}
	mngAuthn := authclient.NewManageClients(hc2)

	return authnSvc, mngAuthn, func() {
		hc2.Disconnect()
		authnSvc.Stop()

		// let background tasks finish
		time.Sleep(time.Millisecond * 100)
	}, err
}

// TestMain creates a test environment
// Used for all test cases in this package
func TestMain(m *testing.M) {
	var err error

	logging.SetLogging("info", "")
	_ = os.RemoveAll(testDir)
	_ = os.MkdirAll(testDir, 0700)

	testServer, err = testenv.StartTestServer(core, false)
	if err != nil {
		panic(err)
	}

	res := m.Run()

	testServer.Stop()
	time.Sleep(time.Second)
	if res == 0 {
		_ = os.RemoveAll(testDir)
	}
	os.Exit(res)
}

// TODO: test both cores at once
//func TestCores(t *testing.T) {
//	var cores = []string{"mqtt", "nats"}
//	for _, core = range cores {
//		t.Run("TestStartStop", TestStartStop)
//	}
//}

// Start the authn service and list clients
func TestStartStop(t *testing.T) {
	t.Log("--- TestStartStop start")
	defer t.Log("--- TestStartStop end")

	// this creates the admin user key
	_, mngAuthn, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	defer stopFn()
	serverURL, _, _ := testServer.MsgServer.GetServerURLs()
	time.Sleep(time.Millisecond * 10)

	clList, err := mngAuthn.GetProfiles()
	require.NoError(t, err)
	// auth service, test client, admin and launcher are 4 pre-existing clients
	assert.Equal(t, 4, len(clList))

	// should be able to connect as admin, using the saved key and token
	hc1 := hubclient.NewHubClient(serverURL, authapi.DefaultAdminUserID, certBundle.CaCert, core)
	hc1.SetRetryConnect(false)
	err = hc1.ConnectWithTokenFile(authConfig.KeysDir)
	require.NoError(t, err)
	hc1.Disconnect()
}

// Create manage users
func TestAddRemoveClientsSuccess(t *testing.T) {
	t.Log("--- TestAddRemoveClientsSuccess start")
	defer t.Log("--- TestAddRemoveClientsSuccess stop")

	deviceID := "device1"
	deviceKP, _ := nkeys.CreateUser()
	deviceKeyPub, _ := deviceKP.PublicKey()
	serviceID := "service1"
	serviceKP, _ := nkeys.CreateUser()
	serviceKeyPub, _ := serviceKP.PublicKey()

	svc, mng, stopFn, err := startTestAuthnService()
	_ = svc
	require.NoError(t, err)
	defer stopFn()

	_, err = mng.AddUser("user1", "user 1", "pass1", "", authapi.ClientRoleViewer)
	assert.NoError(t, err)
	// duplicate should update
	_, err = mng.AddUser("user1", "user 1 updated", "pass1", "", authapi.ClientRoleViewer) // should fail
	assert.NoError(t, err)

	_, err = mng.AddUser("user2", "user 2", "pass2", "", authapi.ClientRoleViewer)
	assert.NoError(t, err)
	_, err = mng.AddUser("user3", "user 3", "pass3", "", authapi.ClientRoleViewer)
	assert.NoError(t, err)
	_, err = mng.AddUser("user4", "user 4", "pass4", "", authapi.ClientRoleViewer)
	assert.NoError(t, err)

	_, err = mng.AddDevice(deviceID, "device 1", deviceKeyPub)
	assert.NoError(t, err)

	_, err = mng.AddService(serviceID, "service 1", serviceKeyPub)
	assert.NoError(t, err)

	// update the server. users can connect and have unlimited access
	authnEntries := svc.MngClients.GetAuthClientList()
	err = testServer.MsgServer.ApplyAuth(authnEntries)
	require.NoError(t, err)

	clList, err := mng.GetProfiles()
	assert.NoError(t, err)
	assert.Equal(t, 6+4, len(clList))
	cnt, _ := mng.GetCount()
	assert.Equal(t, 6+4, cnt)

	err = mng.RemoveClient("user1")
	assert.NoError(t, err)
	err = mng.RemoveClient("user1") // remove is idempotent
	assert.NoError(t, err)
	err = mng.RemoveClient("user2")
	assert.NoError(t, err)
	err = mng.RemoveClient(deviceID)
	assert.NoError(t, err)
	err = mng.RemoveClient(serviceID)
	assert.NoError(t, err)

	require.NoError(t, err)
	clList, err = mng.GetProfiles()
	assert.Equal(t, 2+4, len(clList))

	_, err = mng.AddUser("user1", "user 1", "", "", authapi.ClientRoleViewer)
	assert.NoError(t, err)
	// a bad key
	_, err = mng.AddUser("user2", "user 2", "", "badkey", authapi.ClientRoleViewer)
	assert.NoError(t, err)
}

// Create manage users
func TestAddRemoveClientsFail(t *testing.T) {
	t.Log("--- TestAddRemoveClientsFail start")
	defer t.Log("--- TestAddRemoveClientsFail stop")

	svc, mng, stopFn, err := startTestAuthnService()
	_ = svc
	require.NoError(t, err)
	defer stopFn()

	// missing userID should fail
	_, err = mng.AddUser("", "user 1", "pass1", "", authapi.ClientRoleViewer) // should fail
	assert.Error(t, err)

	// missing deviceID should fail
	_, err = mng.AddDevice("", "", "") // should fail
	assert.Error(t, err)

	// missing serviceID
	_, err = mng.AddService("", "", "") // should fail
	assert.Error(t, err)

	// a bad key
	_, err = mng.AddUser("user2", "user 2", "", "badkey", authapi.ClientRoleViewer)
	assert.NoError(t, err)

	// bad public key
	//_, err = mng.AddDevice("device66", "", "badkey", 0)
	//assert.Error(t, err)
	//_, err = mng.AddService("service66", "", "badkey", 0)
	//assert.Error(t, err)

}

func TestUpdatePubKey(t *testing.T) {
	t.Log("--- TestUpdatePubKey start")
	defer t.Log("--- TestUpdatePubKey end")

	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"

	_, mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)
	serverURL, _, _ := testServer.MsgServer.GetServerURLs()

	// add user to test with. don't set the public key yet
	token, err := mng.AddUser(tu1ID, "testuser 1", tu1Pass, "", authapi.ClientRoleViewer)
	assert.Empty(t, token) // without a public key there is no token
	require.NoError(t, err)

	// 1. connect to the added user using its password
	kp := testServer.MsgServer.CreateKeyPair()
	hc1 := hubclient.NewHubClient(serverURL, tu1ID, certBundle.CaCert, core)
	hc1.SetRetryConnect(false)
	err = hc1.ConnectWithPassword(tu1Pass)
	require.NoError(t, err)
	defer hc1.Disconnect()

	// 2. update the public key and reconnect
	cl1 := authclient.NewProfileClient(hc1)
	tu1Pub := kp.ExportPublic()
	err = cl1.UpdatePubKey(tu1Pub)
	assert.NoError(t, err)
	//hc1.Disconnect()

	prof, err := cl1.GetProfile()
	require.NoError(t, err)
	assert.Equal(t, tu1Pub, prof.PubKey)
}

// Note: Refresh is only useful when using JWT. The nats nkey server ignores the token and uses nkeys
func TestLoginRefresh(t *testing.T) {
	t.Log("--- TestLoginRefresh start")
	defer t.Log("--- TestLoginRefresh end")

	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"
	var authToken1 string
	var authToken2 string

	svc, mng, stopFn, err := startTestAuthnService()
	_ = svc
	defer stopFn()
	require.NoError(t, err)
	serverURL, _, _ := testServer.MsgServer.GetServerURLs()

	// add user to test with
	hc1 := hubclient.NewHubClient(serverURL, tu1ID, certBundle.CaCert, core)
	tu1Key := hc1.CreateKeyPair()
	tu1KeyPub := tu1Key.ExportPublic()
	// AddUser returns a token. JWT or Nkey public key depending on server
	tu1Token, err := mng.AddUser(tu1ID, "testuser 1", tu1Pass, tu1KeyPub, authapi.ClientRoleViewer)
	require.NoError(t, err)
	assert.NotEmpty(t, tu1Token)
	err = testServer.MsgServer.ValidateToken(tu1ID, tu1Token, "", "")
	require.NoError(t, err)

	// 1. connect to the added user using its password
	//hc1 := hubclient.NewHubClient(serverURL, tu1ID,  certBundle.CaCert, core)
	err = hc1.ConnectWithPassword(tu1Pass)
	require.NoError(t, err)
	defer hc1.Disconnect()

	// 2. PubAction a new token.
	cl1 := authclient.NewProfileClient(hc1)
	authToken1, err = cl1.NewToken(tu1Pass)
	require.NoError(t, err)

	// 3. login with the new token
	// (nkeys and callout auth doesn't need a server reload)
	hc2 := hubclient.NewHubClient(serverURL, tu1ID, certBundle.CaCert, core)
	hc2.SetRetryConnect(false)
	err = hc2.ConnectWithToken(tu1Key, authToken1)
	require.NoError(t, err)
	cl2 := authclient.NewProfileClient(hc2)
	prof2, err := cl2.GetProfile()
	require.NoError(t, err)
	require.Equal(t, tu1ID, prof2.ClientID)
	defer hc2.Disconnect()

	// 4. Refresh the token
	authToken2, err = cl1.RefreshToken()
	require.NoError(t, err)
	require.NotEmpty(t, authToken2)

	// 5. login with the refreshed token
	hc3 := hubclient.NewHubClient(serverURL, tu1ID, certBundle.CaCert, core)
	hc2.SetRetryConnect(false)
	err = hc3.ConnectWithToken(tu1Key, authToken2)
	require.NoError(t, err)
	hc3.Disconnect()
	require.NoError(t, err)
}

func TestRefreshNoPubKey(t *testing.T) {
	t.Log("--- TestRefreshNoPubKey start")
	defer t.Log("--- TestRefreshNoPubKey end")
	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"
	var authToken1 string
	serverURL, _, _ := testServer.MsgServer.GetServerURLs()

	svc, mng, stopFn, err := startTestAuthnService()
	_ = svc
	defer stopFn()
	require.NoError(t, err)

	// add user to test with. password and no public key
	// a token is not returned when no pubkey is provided
	tu1Token, err := mng.AddUser(tu1ID, "testuser 1", tu1Pass, "", authapi.ClientRoleViewer)
	require.NoError(t, err)
	assert.Empty(t, tu1Token)

	// connect with the added user token
	hc1 := hubclient.NewHubClient(serverURL, tu1ID, certBundle.CaCert, core)
	hc1.SetRetryConnect(false)
	err = hc1.ConnectWithPassword(tu1Pass)
	defer hc1.Disconnect()
	require.NoError(t, err)
	cl1 := authclient.NewProfileClient(hc1)

	//  refresh fails without a public key
	authToken1, err = cl1.RefreshToken()
	require.Error(t, err)
	assert.Empty(t, authToken1)

	// after setting pub key refresh should succeed
	t.Log("set public key and refresh should succeed")
	tu1Key := hc1.CreateKeyPair()
	err = cl1.UpdatePubKey(tu1Key.ExportPublic())
	require.NoError(t, err)
	authToken1, err = cl1.RefreshToken()
	require.NoError(t, err)
	assert.NotEmpty(t, authToken1)
	t.Log("connecting with token")
	hc2 := hubclient.NewHubClient(serverURL, tu1ID, certBundle.CaCert, core)
	hc2.SetRetryConnect(false)
	err = hc2.ConnectWithToken(tu1Key, authToken1)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 10)
	hc2.Disconnect()
	t.Log("done")
}

func TestUpdateProfile(t *testing.T) {
	t.Log("--- TestUpdateProfile start")
	defer t.Log("--- TestUpdateProfile end")
	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	_, mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)
	serverURL, _, _ := testServer.MsgServer.GetServerURLs()

	// add user to test with and connect
	_, _, err = addNewUser(tu1ID, tu1Name, "pass0", mng)
	require.NoError(t, err)
	//tu1Key, _ := testServer.MsgServer.CreateKP()
	hc1 := hubclient.NewHubClient(serverURL, tu1ID, certBundle.CaCert, core)
	hc1.SetRetryConnect(false)
	err = hc1.ConnectWithPassword("pass0")
	require.NoError(t, err)
	defer hc1.Disconnect()

	// update display name
	const newDisplayName = "new display name"
	cl := authclient.NewProfileClient(hc1)
	err = cl.UpdateName(newDisplayName)
	assert.NoError(t, err)
	prof, err := cl.GetProfile()
	require.NoError(t, err)
	assert.Equal(t, newDisplayName, prof.DisplayName)
}

func TestUpdatePassword(t *testing.T) {
	t.Log("--- TestUpdate start")
	defer t.Log("--- TestUpdate end")
	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	_, mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)
	serverURL, _, _ := testServer.MsgServer.GetServerURLs()

	// add user to test with and connect
	_, _, err = addNewUser(tu1ID, tu1Name, "oldpass", mng)
	hc1 := hubclient.NewHubClient(serverURL, tu1ID, certBundle.CaCert, core)
	hc1.SetRetryConnect(false)
	err = hc1.ConnectWithPassword("oldpass")
	require.NoError(t, err)
	authProfileClient := authclient.NewProfileClient(hc1)
	p1, err := authProfileClient.GetProfile()
	require.NoError(t, err)
	assert.NotEmpty(t, p1)

	// update password
	cl := authclient.NewProfileClient(hc1)
	err = cl.UpdatePassword("newpass")
	require.NoError(t, err)
	hc1.Disconnect()
	time.Sleep(time.Millisecond)

	// login with old password should now fail
	// see also: https://github.com/eclipse/paho.golang/issues/192
	// suggest to hook into OnConnectError and determine cause to decide to retry
	t.Log("an error is expected logging in with the old password")
	err = hc1.ConnectWithPassword("oldpass")
	require.Error(t, err)

	// re-login with new password
	err = hc1.ConnectWithPassword("newpass")
	require.NoError(t, err)
	defer hc1.Disconnect()
	//time.Sleep(time.Second * 1)
	cl = authclient.NewProfileClient(hc1)
	_, err = cl.GetProfile()
	//time.Sleep(time.Second * 3)
	require.NoError(t, err)
}
