package authn_test

import (
	authn2 "github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authn/authnclient"
	"github.com/hiveot/hub/core/authn/authnservice"
	"github.com/hiveot/hub/core/authn/authnstore"
	"github.com/hiveot/hub/core/authn/natsauthn"
	"github.com/hiveot/hub/core/hubclient"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/hiveot/hub/core/msgserver/natsserver"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/slog"
	"os"
	"path"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/logging"
)

var authBundle = testenv.CreateTestAuthBundle()
var testDir = path.Join(os.TempDir(), "test-authn")

// the following are set by the testmain
var clientURL string
var msgServer *natsserver.HubNatsServer

// run the test for different cores
var useCore = "nats" // nats vs mqtt

// add new user to test with
func addNewUser(userID string, displayName string, mng authn2.IAuthnManage) (token string, key nkeys.KeyPair, err error) {
	tu1Key, _ := nkeys.CreateUser()
	tu1KeyPub, _ := tu1Key.PublicKey()
	tu1Token, err := mng.AddUser(userID, displayName, "", tu1KeyPub)
	return tu1Token, tu1Key, err
}
func connectUser(key nkeys.KeyPair, token string) (hc hubclient.IHubClient, err error) {
	// 1. connect with the added user token
	hc = natshubclient.NewHubClient(key)
	err = hc.ConnectWithJWT(clientURL, token, authBundle.CaCert)
	return hc, err
}

// launch the authn service and return a client for using and managing it.
func startTestAuthnService() (cl authn2.IAuthnUser, mng authn2.IAuthnManage, stopFn func(), err error) {
	slog.Info(">>> startTestAuthnService -- begin")
	// the password file to use
	passwordFile := path.Join(testDir, "test.passwd")

	// TODO: put this in a test environment
	_ = os.Remove(passwordFile)
	cfg := authn.AuthnConfig{}
	_ = cfg.InitConfig("", testDir)
	cfg.PasswordFile = passwordFile
	cfg.DeviceTokenValidity = 10

	// setup the authn service
	tokenizer := natsauthn.NewAuthnNatsTokenizer(authBundle.AppAccountKey)
	hcSvc := natshubclient.NewHubClient(authBundle.ServiceKey)

	authnJWT, _ := tokenizer.CreateToken(
		authn2.AuthnServiceName,
		authn2.ClientTypeService,
		authBundle.ServiceKeyPub,
		authn2.DefaultServiceTokenValiditySec)
	err = hcSvc.ConnectWithJWT(clientURL, authnJWT, authBundle.CaCert)
	if err != nil {
		panic("can't connect authn to server: " + err.Error())
	}
	authStore := authnstore.NewAuthnFileStore(passwordFile)
	authnSvc := authnservice.NewAuthnService(authStore, tokenizer, hcSvc)
	err = authnSvc.Start()
	if err != nil {
		logrus.Panicf("cant start test authn service: %s", err)
	}

	//--- create a hub client for the authn client and management api's
	hcClient := natshubclient.NewHubClient(authBundle.ServiceKey)

	//noAuthKey, _ := nkeys.FromSeed([]byte(natshubclient.PublicUnauthenticatedNKey))

	err = hcClient.ConnectWithJWT(clientURL, authBundle.ServiceJWT, authBundle.CaCert)
	if err != nil {
		panic(err)
	}
	mngAuthn := authnclient.NewAuthnManageClient(hcClient)
	clientAuthn := authnclient.NewAuthnUserClient(hcClient)
	//
	return clientAuthn, mngAuthn, func() {
		authnSvc.Stop()
		hcSvc.Disconnect()
		hcClient.Disconnect()
		authStore.Close()
		slog.Info("<<< startTestAuthnService - stopped")

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

	clientURL, msgServer, err = testenv.StartTestServer(testDir, &authBundle)
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

// Create and verify a JWT token
func TestStartStop(t *testing.T) {
	slog.Info("--- TestStartStop start")
	_, _, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 10)
	stopFn()
	slog.Info("--- TestStartStop end")
}

//
//func TestLoginWithPassword(t *testing.T) {
//	const tu2ID = "user2"
//	const tu2Pass = "pass2"
//	slog.Info("--- TestLoginWithPassword start")
//	_, mng, stopFn, err := startTestAuthnService()
//	defer stopFn()
//	require.NoError(t, err)
//
//	tu2Key, _ := nkeys.CreateUser()
//	tu2KeyPub, _ := tu2Key.PublicKey()
//	tu2Token, err := mng.AddUser(tu2ID, "another user", tu2Pass, tu2KeyPub)
//	assert.NoError(t, err)
//	assert.NotEmpty(t, tu2Token)
//
//	hc1 := natshubclient.NewHubClient(tu2Key)
//	// FIXME: support for ConnectWithPassword
//	tu2Token2, err := hc1.ConnectWithPassword(clientURL, tu2ID, tu2Pass, authBundle.CaCert)
//	require.NoError(t, err)
//	//
//	err = hc1.ConnectWithJWT(clientURL, tu2Token2, authBundle.CaCert)
//	require.NoError(t, err)
//	hc1.Disconnect()
//
//	_, err = hc1.ConnectWithPassword(clientURL, tu2ID, "wrongpass", authBundle.CaCert)
//	require.Error(t, err)
//	hc1.Disconnect()
//}

func TestLoginWithJWT(t *testing.T) {
	slog.Info("--- TestLoginWithJWT start")
	_, _, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	defer stopFn()

	// raw generate a jwt token
	userKey, _ := nkeys.CreateUser()
	userPub, _ := userKey.PublicKey()
	userClaims := jwt.NewUserClaims(userPub)
	userClaims.IssuerAccount, _ = authBundle.AppAccountKey.PublicKey()
	userClaims.Name = "user1"
	token, _ := userClaims.Encode(authBundle.AppAccountKey)
	hc1 := natshubclient.NewHubClient(userKey)
	err = hc1.ConnectWithJWT(clientURL, token, authBundle.CaCert)
	require.NoError(t, err)
	hc1.Disconnect()

	slog.Info("--- TestLoginWithJWT end")
}

func TestLoginWithInvalidJWT(t *testing.T) {
	slog.Info("--- TestLoginWithInvalidJWT start")
	_, _, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	defer stopFn()

	// token signed by fake account should fail
	fakeAccountKey, _ := nkeys.CreateAccount()
	userKey := authBundle.UserKey
	userPub, _ := userKey.PublicKey()
	userClaims := jwt.NewUserClaims(userPub)
	userClaims.IssuerAccount, _ = fakeAccountKey.PublicKey()
	badToken, _ := userClaims.Encode(fakeAccountKey)
	hc1 := natshubclient.NewHubClient(userKey)
	err = hc1.ConnectWithJWT(clientURL, badToken, authBundle.CaCert)
	require.Error(t, err)

	slog.Info("--- TestLoginWithInvalidJWT end")
}

// Create manage users
func TestAddRemoveClients(t *testing.T) {
	slog.Info("--- TestAddRemoveClients start")
	_, mng, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	defer stopFn()

	_, err = mng.AddUser("user1", "user 1", "pass1", "")
	assert.NoError(t, err)
	// duplicate fail
	_, err = mng.AddUser("user1", "user 1", "pass1", "") // should fail
	assert.Error(t, err)
	// missing userID
	_, err = mng.AddUser("", "user 1", "pass1", "") // should fail
	assert.Error(t, err)

	_, err = mng.AddUser("user2", "user 2", "pass2", "")
	assert.NoError(t, err)
	_, err = mng.AddUser("user3", "user 3", "pass3", "")
	assert.NoError(t, err)
	_, err = mng.AddUser("user4", "user 4", "pass4", "")
	assert.NoError(t, err)

	_, err = mng.AddDevice(authBundle.DeviceID, "device 1", authBundle.DeviceKeyPub, 100)
	assert.NoError(t, err)
	// duplicate fail
	_, err = mng.AddDevice(authBundle.DeviceID, "", "", 100) // should fail
	assert.Error(t, err)
	// missing userID
	_, err = mng.AddDevice("", "", "", 100) // should fail
	assert.Error(t, err)

	_, err = mng.AddService(authBundle.ServiceID, "service 1", authBundle.ServiceKeyPub, 100)
	assert.NoError(t, err)
	// duplicate fail
	_, err = mng.AddService(authBundle.ServiceID, "", "", 100) // should fail
	assert.Error(t, err)
	// missing userID
	_, err = mng.AddService("", "", "", 100) // should fail
	assert.Error(t, err)

	clList, err := mng.ListClients()
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
	err = mng.RemoveClient(authBundle.DeviceID)
	assert.NoError(t, err)
	err = mng.RemoveClient(authBundle.ServiceID)
	assert.NoError(t, err)

	require.NoError(t, err)
	clList, err = mng.ListClients()
	assert.Equal(t, 2, len(clList))

	_, err = mng.AddUser("user1", "user 1", "", "")
	assert.NoError(t, err)
	_, err = mng.AddUser("user2", "user 2", "", "badkey")
	assert.Error(t, err)

	// missing public key
	_, err = mng.AddDevice("device66", "", "badkey", 0)
	assert.Error(t, err)
	_, err = mng.AddService("service66", "", "badkey", 0)
	assert.Error(t, err)

	slog.Info("--- TestAddRemoveClients stop")
}

func TestLoginRefresh(t *testing.T) {
	slog.Info("--- TestLoginRefresh start")
	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"
	var authToken1 string
	var authToken2 string

	_, mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add user to test with
	tu1Key, _ := nkeys.CreateUser()
	tu1KeyPub, _ := tu1Key.PublicKey()
	tu1Token, err := mng.AddUser(tu1ID, "testuser 1", tu1Pass, tu1KeyPub)
	require.NoError(t, err)
	require.NotEmpty(t, tu1Token)

	// 1. connect with the added user token
	hc1 := natshubclient.NewHubClient(tu1Key)
	defer hc1.Disconnect()
	err = hc1.ConnectWithJWT(clientURL, tu1Token, authBundle.CaCert)
	require.NoError(t, err)

	// 2. Request a new token
	// use the authentication client to request a new token
	cl1 := authnclient.NewAuthnUserClient(hc1)
	authToken1, err = cl1.NewToken(tu1ID, tu1Pass)
	require.NoError(t, err)
	assert.NotEmpty(t, authToken1)
	// wrong ID should fail
	_, err = cl1.NewToken("nottu1", "badpass")
	require.Error(t, err)
	// bad pass should fail
	_, err = cl1.NewToken(tu1ID, "badpass")
	require.Error(t, err)

	// 3. login with the new token
	hc2 := natshubclient.NewHubClient(tu1Key)
	err = hc2.ConnectWithJWT(clientURL, authToken1, authBundle.CaCert)
	require.NoError(t, err)
	cl2 := authnclient.NewAuthnUserClient(hc2)
	prof2, err := cl2.GetProfile(authBundle.UserID)
	require.NoError(t, err)
	require.Equal(t, tu1ID, prof2.ClientID)
	defer hc2.Disconnect()

	// 4. Obtain a refresh token using the new token
	authToken2, err = cl1.Refresh(tu1ID, authToken1)
	require.NoError(t, err)
	require.NotEmpty(t, authToken2)

	// 5. login with the refreshed token
	hc3 := natshubclient.NewHubClient(tu1Key)
	err = hc3.ConnectWithJWT(clientURL, authToken2, authBundle.CaCert)
	require.NoError(t, err)
	hc3.Disconnect()
	require.NoError(t, err)

	// 6. login with a forged token should fail
	cl4 := natshubclient.NewHubClient(tu1Key)
	appAcctPub, _ := authBundle.AppAccountKey.PublicKey()
	fakeAcct, _ := nkeys.CreateAccount()
	forgedClaims := jwt.NewUserClaims(tu1KeyPub)
	forgedClaims.Issuer = appAcctPub
	forgedJWT, err := forgedClaims.Encode(fakeAcct) // <- forged
	require.NoError(t, err)

	err = cl4.ConnectWithJWT(clientURL, forgedJWT, authBundle.CaCert)
	require.Error(t, err)

	slog.Info("--- TestLoginRefresh end")
}

func TestRefreshFakeToken(t *testing.T) {
	slog.Info("--- TestRefreshFakeToken start")
	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"
	var authToken1 string

	_, mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add user to test with
	tu1Key, _ := nkeys.CreateUser()
	tu1KeyPub, _ := tu1Key.PublicKey()
	tu1Token, err := mng.AddUser(tu1ID, "testuser 1", tu1Pass, tu1KeyPub)
	require.NoError(t, err)
	require.NotEmpty(t, tu1Token)

	// 1. connect with the added user token
	hc1 := natshubclient.NewHubClient(tu1Key)
	defer hc1.Disconnect()
	err = hc1.ConnectWithJWT(clientURL, tu1Token, authBundle.CaCert)
	require.NoError(t, err)

	// 2. Refresh a fake token from another service
	cl1 := authnclient.NewAuthnUserClient(hc1)
	fakeToken := authBundle.ServiceJWT
	authToken1, err = cl1.Refresh(tu1ID, fakeToken)
	require.Error(t, err)
	assert.Empty(t, authToken1)

	// 3. Refresh a self generated fake token
	appAcctPub, _ := authBundle.AppAccountKey.PublicKey()
	fakeAcct, _ := nkeys.CreateAccount()
	forgedClaims := jwt.NewUserClaims(tu1KeyPub)
	forgedClaims.Issuer = appAcctPub
	forgedJWT, err := forgedClaims.Encode(fakeAcct) // <- forged
	authToken1, err = cl1.Refresh(tu1ID, forgedJWT)
	require.Error(t, err)
	assert.Empty(t, authToken1)
}

func TestLoginFail(t *testing.T) {
	slog.Info("--- TestLoginFail start")
	var testuser1 = "testuser1"
	var testpass1 = "testpass1"

	_, mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add a user to test with
	time.Sleep(time.Second)
	_, err = mng.AddUser(testuser1, "user 1", testpass1, "")
	require.NoError(t, err)

	// login and get tokens
	hc := natshubclient.NewHubClient(nil)
	_, err = hc.ConnectWithPassword(clientURL, testuser1, "badpass", authBundle.CaCert)

	//pubKey, _ := authBundle.UserKey.PublicKey()
	//authToken, err := cl.NewToken(authBundle.UserID, "badpass", pubKey)
	assert.Error(t, err)
	//assert.Empty(t, authToken)
	slog.Info("--- TestLoginFail end")
}

func TestUpdate(t *testing.T) {
	slog.Info("--- TestRefreshFakeToken start")
	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	_, mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add user to test with and connect
	tu1Token, tu1Key, err := addNewUser(tu1ID, tu1Name, mng)
	hc, err := connectUser(tu1Key, tu1Token)
	require.NoError(t, err)
	defer hc.Disconnect()

	// update display name, password and public key
	const newDisplayName = "new display name"
	newPK, _ := nkeys.CreateUser()
	newPKPub, _ := newPK.PublicKey()
	cl := authnclient.NewAuthnUserClient(hc)
	err = cl.UpdateName(tu1ID, newDisplayName)
	assert.NoError(t, err)
	err = cl.UpdatePassword(tu1ID, "new password")
	assert.NoError(t, err)
	err = cl.UpdatePubKey(tu1ID, newPKPub)
	assert.NoError(t, err)

	prof, err := cl.GetProfile(tu1ID)
	assert.Equal(t, newDisplayName, prof.DisplayName)
	assert.Equal(t, newPKPub, prof.PubKey)

	prof2, err := mng.GetClientProfile(tu1ID)
	assert.Equal(t, prof, prof2)
	prof2.DisplayName = "after update"
	err = mng.UpdateClient(tu1ID, prof2)
	assert.NoError(t, err)

	prof, err = cl.GetProfile(tu1ID)
	assert.Equal(t, prof2.DisplayName, prof.DisplayName)
}
