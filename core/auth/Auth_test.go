package auth_test

import (
	authapi "github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/core/auth/authclient"
	"github.com/hiveot/hub/core/auth/authservice"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubcl"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/slog"
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

// the following are set by the testmain
var clientURL string
var msgServer msgserver.IMsgServer

//var useCallout = false

//func newClient(id string, kp interface{}) hubclient.IHubClient {
//	if core == "nats" {
//		var nkp nkeys.KeyPair
//		if kp != nil {
//			nkp = kp.(nkeys.KeyPair)
//		}
//		return natshubclient.NewNatsHubClient(id, nkp)
//	} else {
//		var ekp *ecdsa.PrivateKey
//		if kp != nil {
//			ekp = kp.(*ecdsa.PrivateKey)
//		}
//		return mqtthubclient.NewMqttHubClient(id, ekp)
//	}
//}

// add new user to test with
func addNewUser(userID string, displayName string, pass string, mng authapi.IAuthnManageClients) (token string, key nkeys.KeyPair, err error) {
	userKey, _ := nkeys.CreateUser()
	userKeyPub, _ := userKey.PublicKey()
	// FIXME: must set a password in order to be able to update it later
	userToken, err := mng.AddUser(userID, displayName, pass, userKeyPub, authapi.ClientRoleViewer)
	return userToken, userKey, err
}

// launch the authn service and return a client for using and managing it.
// the messaging server is already running (see TestMain)
func startTestAuthnService() (authnSvc *authservice.AuthService, mng authapi.IAuthnManageClients, stopFn func(), err error) {
	// the password file to use
	passwordFile := path.Join(testDir, "test.passwd")

	// TODO: put this in a test environment
	_ = os.Remove(passwordFile)
	cfg := authservice.AuthConfig{}
	_ = cfg.Setup(testDir)
	cfg.PasswordFile = passwordFile
	cfg.DeviceTokenValidityDays = 10
	cfg.Encryption = authapi.PWHASH_BCRYPT // nats requires bcrypt

	authnSvc, err = authservice.StartAuthService(cfg, msgServer)
	if err != nil {
		panic("cant start test authn service: " + err.Error())
	}

	//--- connect the authn management client for managing clients
	authClientKey, authClientPub := msgServer.CreateKP()
	_ = authClientKey
	authnSvc.MngClients.AddUser("authn-client", "authn client", "", authClientPub, authapi.ClientRoleAdmin)
	hc2, err := msgServer.ConnectInProc("authn-client")

	if err != nil {
		panic(err)
	}
	mngAuthn := authclient.NewAuthClientsClient(hc2)

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

	clientURL, msgServer, certBundle, err = testenv.StartTestServer(core)
	if err != nil {
		panic(err)
	}

	res := m.Run()

	msgServer.Stop()
	time.Sleep(time.Second)
	if res == 0 {
		_ = os.RemoveAll(testDir)
	}
	os.Exit(res)
}

// Start the authn service and list clients
func TestStartStop(t *testing.T) {
	slog.Info("--- TestStartStop start")
	defer slog.Info("--- TestStartStop end")

	_, mngAuthn, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	defer stopFn()
	time.Sleep(time.Millisecond * 10)

	clList, err := mngAuthn.GetProfiles()
	require.NoError(t, err)
	// auth service and the admin user are 2 existing clients
	assert.Equal(t, 2, len(clList))
}

// Create manage users
func TestAddRemoveClients(t *testing.T) {
	slog.Info("--- TestAddRemoveClients start")
	defer slog.Info("--- TestAddRemoveClients stop")

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
	// duplicate fail
	_, err = mng.AddUser("user1", "user 1", "pass1", "", authapi.ClientRoleViewer) // should fail
	assert.Error(t, err)
	// missing userID
	_, err = mng.AddUser("", "user 1", "pass1", "", authapi.ClientRoleViewer) // should fail
	assert.Error(t, err)

	_, err = mng.AddUser("user2", "user 2", "pass2", "", authapi.ClientRoleViewer)
	assert.NoError(t, err)
	_, err = mng.AddUser("user3", "user 3", "pass3", "", authapi.ClientRoleViewer)
	assert.NoError(t, err)
	_, err = mng.AddUser("user4", "user 4", "pass4", "", authapi.ClientRoleViewer)
	assert.NoError(t, err)

	_, err = mng.AddDevice(deviceID, "device 1", deviceKeyPub)
	assert.NoError(t, err)
	// duplicate fail
	_, err = mng.AddDevice(deviceID, "", "") // should fail
	assert.Error(t, err)
	// missing userID
	_, err = mng.AddDevice("", "", "") // should fail
	assert.Error(t, err)

	_, err = mng.AddService(serviceID, "service 1", serviceKeyPub)
	assert.NoError(t, err)
	// duplicate fail
	_, err = mng.AddService(serviceID, "", "") // should fail
	assert.Error(t, err)
	// missing userID
	_, err = mng.AddService("", "", "") // should fail
	assert.Error(t, err)

	// update the server. users can connect and have unlimited access
	authnEntries := svc.MngClients.GetAuthClientList()
	err = msgServer.ApplyAuth(authnEntries)
	require.NoError(t, err)

	clList, err := mng.GetProfiles()
	assert.NoError(t, err)
	assert.Equal(t, 6+2, len(clList))
	cnt, _ := mng.GetCount()
	assert.Equal(t, 6+2, cnt)

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
	assert.Equal(t, 2+2, len(clList))

	_, err = mng.AddUser("user1", "user 1", "", "", authapi.ClientRoleViewer)
	assert.NoError(t, err)
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
	slog.Info("--- TestUpdatePubKey start")
	defer slog.Info("--- TestUpdatePubKey end")

	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"

	_, mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add user to test with. don't set the public key yet
	_, err = mng.AddUser(tu1ID, "testuser 1", tu1Pass, "", authapi.ClientRoleViewer)
	require.NoError(t, err)

	// 1. connect to the added user using its password
	tu1Key, tu1KeyPub := msgServer.CreateKP()
	hc1 := hubcl.NewHubClient(clientURL, tu1ID, tu1Key, certBundle.CaCert, core)
	err = hc1.ConnectWithPassword(tu1Pass)
	require.NoError(t, err)
	defer hc1.Disconnect()

	// 2. update the public key and reconnect
	cl1 := authclient.NewAuthProfileClient(hc1)
	err = cl1.UpdatePubKey(tu1KeyPub)
	assert.NoError(t, err)
	//hc1.Disconnect()

	prof, err := cl1.GetProfile()
	require.NoError(t, err)
	assert.Equal(t, tu1KeyPub, prof.PubKey)
}

// Note: Refresh is only useful when using JWT. The nats nkey server ignores the token and uses nkeys
func TestLoginRefresh(t *testing.T) {
	slog.Info("--- TestLoginRefresh start")
	defer slog.Info("--- TestLoginRefresh end")

	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"
	var authToken1 string
	var authToken2 string

	svc, mng, stopFn, err := startTestAuthnService()
	_ = svc
	defer stopFn()
	require.NoError(t, err)

	// add user to test with
	tu1Key, tu1KeyPub := msgServer.CreateKP()
	// AddUser returns a token. JWT or Nkey public key depending on server
	tu1Token, err := mng.AddUser(tu1ID, "testuser 1", tu1Pass, tu1KeyPub, authapi.ClientRoleViewer)
	require.NoError(t, err)
	assert.NotEmpty(t, tu1Token)

	// 1. connect to the added user using its password
	hc1 := hubcl.NewHubClient(clientURL, tu1ID, tu1Key, certBundle.CaCert, core)
	err = hc1.ConnectWithPassword(tu1Pass)
	require.NoError(t, err)
	defer hc1.Disconnect()

	// 2. Request a new token.
	cl1 := authclient.NewAuthProfileClient(hc1)
	authToken1, err = cl1.NewToken(tu1Pass)
	require.NoError(t, err)

	// 3. login with the new token
	// (nkeys and callout auth doesn't need a server reload)
	hc2 := hubcl.NewHubClient(clientURL, tu1ID, tu1Key, certBundle.CaCert, core)
	err = hc2.ConnectWithToken(authToken1)
	require.NoError(t, err)
	cl2 := authclient.NewAuthProfileClient(hc2)
	prof2, err := cl2.GetProfile()
	require.NoError(t, err)
	require.Equal(t, tu1ID, prof2.ClientID)
	defer hc2.Disconnect()

	// 4. Refresh the token
	authToken2, err = cl1.Refresh()
	require.NoError(t, err)
	require.NotEmpty(t, authToken2)

	// 5. login with the refreshed token
	hc3 := hubcl.NewHubClient(clientURL, tu1ID, tu1Key, certBundle.CaCert, core)
	err = hc3.ConnectWithToken(authToken2)
	require.NoError(t, err)
	hc3.Disconnect()
	require.NoError(t, err)
}

func TestRefreshNoPubKey(t *testing.T) {
	slog.Info("--- TestRefreshNoPubKey start")
	defer slog.Info("--- TestRefreshNoPubKey end")
	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"
	var authToken1 string

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
	tu1Key, tu1Pub := msgServer.CreateKP()
	hc1 := hubcl.NewHubClient(clientURL, tu1ID, tu1Key, certBundle.CaCert, core)
	err = hc1.ConnectWithPassword(tu1Pass)
	defer hc1.Disconnect()
	require.NoError(t, err)
	cl1 := authclient.NewAuthProfileClient(hc1)

	//  refresh fails without a public key
	authToken1, err = cl1.Refresh()
	require.Error(t, err)
	assert.Empty(t, authToken1)

	// after setting pub key refresh should succeed
	t.Log("set public key and refresh should succeed")
	err = cl1.UpdatePubKey(tu1Pub)
	require.NoError(t, err)
	authToken1, err = cl1.Refresh()
	require.NoError(t, err)
	assert.NotEmpty(t, authToken1)
	t.Log("connecting with token")
	hc2 := hubcl.NewHubClient(clientURL, tu1ID, tu1Key, certBundle.CaCert, core)
	err = hc2.ConnectWithToken(authToken1)
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

	// add user to test with and connect
	_, _, err = addNewUser(tu1ID, tu1Name, "pass0", mng)
	require.NoError(t, err)
	tu1Key, _ := msgServer.CreateKP()
	hc1 := hubcl.NewHubClient(clientURL, tu1ID, tu1Key, certBundle.CaCert, core)
	err = hc1.ConnectWithPassword("pass0")
	require.NoError(t, err)
	defer hc1.Disconnect()

	// update display name
	const newDisplayName = "new display name"
	cl := authclient.NewAuthProfileClient(hc1)
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
	tu1Key, _ := msgServer.CreateKP()

	_, mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add user to test with and connect
	_, _, err = addNewUser(tu1ID, tu1Name, "pass0", mng)
	hc1 := hubcl.NewHubClient(clientURL, tu1ID, tu1Key, certBundle.CaCert, core)
	err = hc1.ConnectWithPassword("pass0")
	require.NoError(t, err)

	// update password
	cl := authclient.NewAuthProfileClient(hc1)
	err = cl.UpdatePassword("pass1")
	require.NoError(t, err)
	hc1.Disconnect()
	time.Sleep(time.Millisecond)

	// login with old password should now fail
	err = hc1.ConnectWithPassword("pass0")
	require.Error(t, err)

	// re-login with new password
	err = hc1.ConnectWithPassword("pass1")
	require.NoError(t, err)
	cl = authclient.NewAuthProfileClient(hc1)
	_, err = cl.GetProfile()
	require.NoError(t, err)
}
