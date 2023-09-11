package auth_test

import (
	authapi "github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/core/auth/authclient"
	"github.com/hiveot/hub/core/auth/authservice"
	"github.com/hiveot/hub/core/hubclient/mqtthubclient"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/hiveot/hub/lib/certs"
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
	//hc2, err := msgServer.ConnectInProc("authn-client")

	hc2, err := mqtthubclient.ConnectWithPassword(clientURL, "authclient", "", certBundle.CaCert)

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

	//clientURL, msgServer, certBundle, err = testenv.StartTestServer("nats")
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
	assert.Equal(t, 0, len(clList))
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
	assert.Equal(t, 6, len(clList))
	cnt, _ := mng.GetCount()
	assert.Equal(t, 6, cnt)

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
	assert.Equal(t, 2, len(clList))

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

	// add user to test with
	_, err = mng.AddUser(tu1ID, "testuser 1", tu1Pass, "", authapi.ClientRoleViewer)
	require.NoError(t, err)

	// 1. connect to the added user using its password
	hc1, err := natshubclient.ConnectWithPassword(clientURL, tu1ID, tu1Pass, certBundle.CaCert)
	require.NoError(t, err)
	defer hc1.Disconnect()

	// 2. update the public key and reconnect
	tu1Key, _ := nkeys.CreateUser()
	tu1KeyPub, _ := tu1Key.PublicKey()
	cl1 := authclient.NewAuthProfileClient(hc1)
	err = cl1.UpdatePubKey(tu1ID, tu1KeyPub)
	assert.NoError(t, err)
}

// this requires the JWT server. It cannot be used together with NKeys :/
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
	tu1Key, _ := nkeys.CreateUser()
	tu1KeyPub, _ := tu1Key.PublicKey()
	// AddUser returns a token. JWT or Nkey public key depending on server
	tu1Token, err := mng.AddUser(tu1ID, "testuser 1", tu1Pass, tu1KeyPub, authapi.ClientRoleViewer)
	require.NoError(t, err)
	assert.NotEmpty(t, tu1Token)

	// 1. connect to the added user using its password
	hc1, err := natshubclient.ConnectWithPassword(clientURL, tu1ID, tu1Pass, certBundle.CaCert)
	require.NoError(t, err)
	defer hc1.Disconnect()

	// 2. Request a new token.
	cl1 := authclient.NewAuthProfileClient(hc1)
	authToken1, err = cl1.NewToken(tu1ID, tu1Pass)
	require.NoError(t, err)

	// 3. login with the new token
	// (nkeys and callout auth doesn't need a server reload)
	hc2, err := natshubclient.Connect(clientURL, tu1ID, tu1Key, authToken1, certBundle.CaCert)
	require.NoError(t, err)
	cl2 := authclient.NewAuthProfileClient(hc2)
	prof2, err := cl2.GetProfile(tu1ID)
	require.NoError(t, err)
	require.Equal(t, tu1ID, prof2.ClientID)
	defer hc2.Disconnect()

	// 4. Obtain a refresh token using the new token
	authToken2, err = cl1.Refresh(tu1ID, authToken1)
	require.NoError(t, err)
	require.NotEmpty(t, authToken2)

	// 5. login with the refreshed token
	hc3, err := natshubclient.Connect(clientURL, tu1ID, tu1Key, authToken2, certBundle.CaCert)
	require.NoError(t, err)
	hc3.Disconnect()
	require.NoError(t, err)
}

func TestRefreshFakeToken(t *testing.T) {
	slog.Info("--- TestRefreshFakeToken start")
	defer slog.Info("--- TestRefreshFakeToken end")
	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"
	var authToken1 string

	svc, mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add user to test with. password and no public key
	tu1Key, _ := nkeys.CreateUser()
	tu1KeyPub, _ := tu1Key.PublicKey()
	tu1Token, err := mng.AddUser(tu1ID, "testuser 1", tu1Pass, tu1KeyPub, authapi.ClientRoleViewer)
	_ = tu1Token
	require.NoError(t, err)
	_, err = mng.AddUser(testenv.TestUser2ID, "user 2", "", testenv.TestUser2Pub, authapi.ClientRoleViewer)
	require.NoError(t, err)

	entries := svc.MngClients.GetAuthClientList()
	err = msgServer.ApplyAuth(entries)

	// 1. connect with the added user token
	//hc1, err := connectUser(tu1ID, tu1Key, tu1Token)
	hc1, err := natshubclient.ConnectWithPassword(clientURL, tu1ID, tu1Pass, certBundle.CaCert)
	defer hc1.Disconnect()
	require.NoError(t, err)
	cl1 := authclient.NewAuthProfileClient(hc1)

	// 2: test refresh without any token
	authToken1, err = cl1.Refresh(tu1ID, "")
	require.Error(t, err)
	assert.Empty(t, authToken1)

	// 3. Use a jwt token from another user
	fakeToken, err := msgServer.CreateToken(testenv.TestUser2ID)
	require.NoError(t, err)
	authToken1, err = cl1.Refresh(testenv.TestUser2ID, fakeToken)
	require.Error(t, err)
	assert.Empty(t, authToken1)

	//// 4. Use a fake public key, eg from another user
	//fakeToken, _ = serverCfg.CoreServiceKP.PublicKey()
	//authToken1, err = cl1.Refresh(tu1ID, fakeToken)
	//require.Error(t, err)
	//assert.Empty(t, authToken1)

	//// 5. Try refreshing a self generated token
	//appAcctPub, _ := serverCfg.AppAccountKP.PublicKey()
	//fakeAcct, _ := nkeys.CreateAccount()
	//forgedClaims := jwt.NewUserClaims(tu1KeyPub)
	//forgedClaims.Issuer = appAcctPub
	//forgedJWT, err := forgedClaims.Encode(fakeAcct) // <- forged
	//authToken1, err = cl1.Refresh(tu1ID, forgedJWT)
	//require.Error(t, err)
	//assert.Empty(t, authToken1)
}

func TestUpdate(t *testing.T) {
	t.Log("--- TestUpdate start")
	defer t.Log("--- TestUpdate end")
	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	svc, mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add user to test with and connect
	tu1Token, tu1Key, err := addNewUser(tu1ID, tu1Name, "pass0", mng)
	require.NoError(t, err)

	entries := svc.MngClients.GetAuthClientList()
	err = msgServer.ApplyAuth(entries)
	require.NoError(t, err)

	hc, err := natshubclient.Connect(clientURL, tu1ID, tu1Key, tu1Token, certBundle.CaCert)
	require.NoError(t, err)
	//defer hc.Disconnect()

	// update display name, password and public key and reload
	const newDisplayName = "new display name"
	newPK, _ := nkeys.CreateUser()
	newPKPub, _ := newPK.PublicKey()
	cl := authclient.NewAuthProfileClient(hc)
	err = cl.UpdateName(tu1ID, newDisplayName)
	assert.NoError(t, err)
	err = cl.UpdatePassword(tu1ID, "new password")
	assert.NoError(t, err)
	err = cl.UpdatePubKey(tu1ID, newPKPub)
	assert.NoError(t, err)
	hc.Disconnect()

	entries = svc.MngClients.GetAuthClientList()
	err = msgServer.ApplyAuth(entries)

	//reconnect using the new key
	hc, err = natshubclient.Connect(clientURL, tu1ID, newPK, newPKPub, certBundle.CaCert)
	require.NoError(t, err)
	defer hc.Disconnect()
	cl = authclient.NewAuthProfileClient(hc)

	prof, err := cl.GetProfile(tu1ID)
	assert.Equal(t, newDisplayName, prof.DisplayName)
	assert.Equal(t, newPKPub, prof.PubKey)

	prof2, err := mng.GetProfile(tu1ID)
	assert.Equal(t, prof, prof2)
	prof2.DisplayName = "after update"
	err = mng.UpdateClient(tu1ID, prof2)
	assert.NoError(t, err)

	prof, err = cl.GetProfile(tu1ID)
	assert.Equal(t, prof2.DisplayName, prof.DisplayName)
	// clean shutdown background processes
	time.Sleep(time.Millisecond)
}
