package authn_test

import (
	authn2 "github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authn/authnadapter"
	"github.com/hiveot/hub/core/authn/authnclient"
	"github.com/hiveot/hub/core/authn/authnservice"
	"github.com/hiveot/hub/core/authn/authnstore"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/hiveot/hub/core/msgserver/natsserver"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
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

var certBundle certs.TestCertBundle
var serverCfg *natsserver.NatsServerConfig
var testDir = path.Join(os.TempDir(), "test-authn")

// the following are set by the testmain
var clientURL string
var msgServer *natsserver.NatsNKeyServer

// run the test for different cores
var useCore = "natsnkey" // natsnkey, natsjwt, natscallout, mqtt

// add new user to test with
func addNewUser(userID string, displayName string, pass string, mng authn2.IAuthnManage) (token string, key nkeys.KeyPair, err error) {
	userKey, _ := nkeys.CreateUser()
	userKeyPub, _ := userKey.PublicKey()
	// FIXME: must set a password in order to update it later
	userToken, err := mng.AddUser(userID, displayName, pass, userKeyPub)
	return userToken, userKey, err
}
func connectUser(clientID string, key nkeys.KeyPair, token string) (hc hubclient.IHubClient, err error) {
	hc, err = natshubclient.Connect(clientURL, clientID, key, token, certBundle.CaCert)
	return hc, err
}

// launch the authn service and return a client for using and managing it.
func startTestAuthnService() (mng authn2.IAuthnManage, stopFn func(), err error) {
	// the password file to use
	passwordFile := path.Join(testDir, "test.passwd")

	// TODO: put this in a test environment
	_ = os.Remove(passwordFile)
	cfg := authn.AuthnConfig{}
	_ = cfg.Setup(testDir)
	cfg.PasswordFile = passwordFile
	cfg.DeviceTokenValidity = 10

	// setup the authn service
	// TODO: support JWT tokens
	tokenizer := authnadapter.NewNatsAuthnTokenizer(serverCfg.AppAccountKP, true)
	nc1, err := msgServer.ConnectInProc("authn", nil)
	hc1, _ := natshubclient.ConnectWithNC(nc1, "authn")
	if err != nil {
		panic("can't connect authn to server: " + err.Error())
	}
	// check js access
	js, err := nc1.JetStream()
	if err != nil {
		panic("no js")
	}
	ai, err := js.AccountInfo()
	_ = ai
	if err != nil {
		panic("no js")
	}

	authStore := authnstore.NewAuthnFileStore(passwordFile)
	authnSvc := authnservice.NewAuthnService(authStore, msgServer, tokenizer, hc1)
	err = authnSvc.Start()
	if err != nil {
		logrus.Panicf("cant start test authn service: %s", err)
	}

	//--- create a hub client for the authn management
	nc2, err := msgServer.ConnectInProc("authn-client", nil)
	hc2, err := natshubclient.ConnectWithNC(nc2, "authn-client")
	if err != nil {
		panic(err)
	}
	mngAuthn := authnclient.NewAuthnManageClient(hc2)
	//
	return mngAuthn, func() {
		authnSvc.Stop()
		hc1.Disconnect()
		hc2.Disconnect()
		authStore.Close()

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

	clientURL, msgServer, certBundle, serverCfg, err = testenv.StartNatsTestServer()
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
	defer slog.Info("--- TestStartStop end")

	mng, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	defer stopFn()
	time.Sleep(time.Millisecond * 10)
	clList, err := mng.ListClients()
	require.NoError(t, err)
	assert.Equal(t, 0, len(clList))

}

func TestLoginWithNKey(t *testing.T) {

	slog.Info("--- TestLoginWithNKey start")
	defer slog.Info("--- TestLoginWithNKey end")
	var rxMsg string
	mng, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	defer stopFn()

	tu1ID := "user1"
	tu1Token, tu1Key, err := addNewUser(tu1ID, "user-1", "", mng)
	assert.NoError(t, err)
	assert.NotEmpty(t, tu1Token)

	hc1, err := connectUser(tu1ID, tu1Key, tu1Token)
	require.NoError(t, err)
	defer hc1.Disconnect()

	_, err = hc1.Sub("things.>", func(topic string, data []byte) {
		rxMsg = string(data)
		slog.Info("received message", "msg", rxMsg)
	})
	assert.NoError(t, err)
	err = hc1.Pub("things.service1.event", []byte("hello world"))
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	assert.Equal(t, "hello world", rxMsg)

}

func TestLoginWithPassword(t *testing.T) {
	var rxMsg string
	const tu2ID = "user2"
	const tu2Pass = "pass2"
	slog.Info("--- TestLoginWithPassword start")
	defer slog.Info("--- TestLoginWithPassword end")

	mng, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	defer stopFn()

	tu2Token, err := mng.AddUser(tu2ID, "another user", tu2Pass, "")
	assert.NoError(t, err)
	assert.Empty(t, tu2Token)

	hc1, err := natshubclient.ConnectWithPassword(clientURL, tu2ID, tu2Pass, certBundle.CaCert)
	require.NoError(t, err)
	defer hc1.Disconnect()

	_, err = hc1.Sub("things.>", func(topic string, data []byte) {
		rxMsg = string(data)
		slog.Info("received message", "msg", rxMsg)
	})
	assert.NoError(t, err)
	err = hc1.Pub("things.service1.event", []byte("hello world"))
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	assert.Equal(t, "hello world", rxMsg)
}

// this requires the JWT server. It cannot be used together with NKeys :/
func TestLoginWithJWT(t *testing.T) {
	slog.Info("--- TestLoginWithJWT start")
	defer slog.Info("--- TestLoginWithJWT end")

	rxMsg := ""
	_, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	defer stopFn()

	// raw generate a jwt token
	//userKey, _ := nkeys.CreateUser()
	userKey := serverCfg.CoreServiceKP
	userJWT := serverCfg.CoreServiceJWT
	hc1, err := natshubclient.ConnectWithJWT(clientURL, userKey, userJWT, certBundle.CaCert)
	require.NoError(t, err)

	_, err = hc1.Subscribe("things.>", func(msg *nats.Msg) {
		rxMsg = string(msg.Data)
		slog.Info("received message", "msg", rxMsg)
	})
	assert.NoError(t, err, "unable to subscribe")
	err = hc1.Pub("things.service1.event", []byte("hello world"))
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	assert.Equal(t, "hello world", rxMsg)

	hc1.Disconnect()
}

func TestLoginWithInvalidJWT(t *testing.T) {
	slog.Info("--- TestLoginWithInvalidJWT start")
	defer slog.Info("--- TestLoginWithInvalidJWT end")
	_, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	defer stopFn()

	// token signed by fake account should fail
	fakeAccountKey, _ := nkeys.CreateAccount()
	userKey, _ := nkeys.CreateUser()
	userPub, _ := userKey.PublicKey()
	userClaims := jwt.NewUserClaims(userPub)
	userClaims.IssuerAccount, _ = fakeAccountKey.PublicKey()
	badToken, _ := userClaims.Encode(fakeAccountKey)
	hc1, err := natshubclient.ConnectWithJWT(clientURL, userKey, badToken, certBundle.CaCert)
	require.Error(t, err)
	require.Empty(t, hc1)

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

	mng, stopFn, err := startTestAuthnService()
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

	_, err = mng.AddDevice(deviceID, "device 1", deviceKeyPub, 100)
	assert.NoError(t, err)
	// duplicate fail
	_, err = mng.AddDevice(deviceID, "", "", 100) // should fail
	assert.Error(t, err)
	// missing userID
	_, err = mng.AddDevice("", "", "", 100) // should fail
	assert.Error(t, err)

	_, err = mng.AddService(serviceID, "service 1", serviceKeyPub, 100)
	assert.NoError(t, err)
	// duplicate fail
	_, err = mng.AddService(serviceID, "", "", 100) // should fail
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
	err = mng.RemoveClient(deviceID)
	assert.NoError(t, err)
	err = mng.RemoveClient(serviceID)
	assert.NoError(t, err)

	require.NoError(t, err)
	clList, err = mng.ListClients()
	assert.Equal(t, 2, len(clList))

	_, err = mng.AddUser("user1", "user 1", "", "")
	assert.NoError(t, err)
	// a bad key
	_, err = mng.AddUser("user2", "user 2", "", "badkey")
	assert.NoError(t, err)

	// bad public key
	//_, err = mng.AddDevice("device66", "", "badkey", 0)
	//assert.Error(t, err)
	//_, err = mng.AddService("service66", "", "badkey", 0)
	//assert.Error(t, err)

}

// this requires the JWT server. It cannot be used together with NKeys :/
func TestLoginRefresh(t *testing.T) {
	slog.Info("--- TestLoginRefresh start")
	defer slog.Info("--- TestLoginRefresh end")

	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"
	var authToken1 string
	var authToken2 string

	mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add user to test with
	tu1Key, _ := nkeys.CreateUser()
	tu1KeyPub, _ := tu1Key.PublicKey()
	// AddUser returns a token. JWT or Nkey public key depending on server
	tu1Token, err := mng.AddUser(tu1ID, "testuser 1", tu1Pass, "")
	require.NoError(t, err)
	assert.Empty(t, tu1Token)

	// 1. connect to the added user using its password
	hc1, err := connectUser(tu1ID, tu1Key, tu1Pass)
	require.NoError(t, err)
	defer hc1.Disconnect()

	// 2. Request a new token
	cl1 := authnclient.NewAuthnUserClient(hc1)
	// without a pubkey NewToken should fail
	authToken1, err = cl1.NewToken(tu1ID, tu1Pass)
	assert.Error(t, err)

	// use the authentication client to request a new token
	err = cl1.UpdatePubKey(tu1ID, tu1KeyPub)
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
	hc2, err := connectUser(tu1ID, tu1Key, authToken1)
	require.NoError(t, err)
	cl2 := authnclient.NewAuthnUserClient(hc2)
	prof2, err := cl2.GetProfile(tu1ID)
	require.NoError(t, err)
	require.Equal(t, tu1ID, prof2.ClientID)
	defer hc2.Disconnect()

	// 4. Obtain a refresh token using the new token
	authToken2, err = cl1.Refresh(tu1ID, authToken1)
	require.NoError(t, err)
	require.NotEmpty(t, authToken2)

	// 5. login with the refreshed token
	hc3, err := connectUser(tu1ID, tu1Key, authToken2)
	require.NoError(t, err)
	hc3.Disconnect()
	require.NoError(t, err)

	// 6. login with a forged token should fail
	appAcctPub, _ := serverCfg.AppAccountKP.PublicKey()
	fakeAcct, _ := nkeys.CreateAccount()
	forgedClaims := jwt.NewUserClaims(tu1KeyPub)
	forgedClaims.Issuer = appAcctPub
	forgedJWT, err := forgedClaims.Encode(fakeAcct) // <- forged
	require.NoError(t, err)

	hc4, err := connectUser(tu1ID, tu1Key, forgedJWT)
	require.Error(t, err)
	assert.Empty(t, hc4)

}

func TestRefreshFakeToken(t *testing.T) {
	slog.Info("--- TestRefreshFakeToken start")
	defer slog.Info("--- TestRefreshFakeToken end")
	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"
	var authToken1 string

	mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add user to test with. no password and no public key
	tu1Key, _ := nkeys.CreateUser()
	tu1KeyPub, _ := tu1Key.PublicKey()
	tu1Token, err := mng.AddUser(tu1ID, "testuser 1", tu1Pass, "")
	_ = tu1Token
	require.NoError(t, err)
	//require.NotEmpty(t, tu1Token)

	// 1. connect with the added user token
	//hc1, err := connectUser(tu1ID, tu1Key, tu1Token)
	hc1, err := natshubclient.ConnectWithPassword(clientURL, tu1ID, tu1Pass, certBundle.CaCert)
	defer hc1.Disconnect()
	require.NoError(t, err)
	cl1 := authnclient.NewAuthnUserClient(hc1)

	// 2: test refresh without any token
	authToken1, err = cl1.Refresh(tu1ID, "")
	require.Error(t, err)
	assert.Empty(t, authToken1)

	// 3. Use a fake jwt token, eg from another user
	fakeToken := serverCfg.CoreServiceJWT
	authToken1, err = cl1.Refresh(tu1ID, fakeToken)
	require.Error(t, err)
	assert.Empty(t, authToken1)

	// 4. Use a fake public key, eg from another user
	//fakeToken := serverCfg.CoreServiceJWT
	fakeToken, _ = serverCfg.CoreServiceKP.PublicKey()
	authToken1, err = cl1.Refresh(tu1ID, fakeToken)
	require.Error(t, err)
	assert.Empty(t, authToken1)

	// 5. Refresh a self generated fake token
	err = cl1.UpdatePubKey(tu1ID, tu1KeyPub)
	require.NoError(t, err)
	appAcctPub, _ := serverCfg.AppAccountKP.PublicKey()
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
	defer slog.Info("--- TestLoginFail end")

	var testuser1 = "testuser1"
	var testpass1 = "testpass1"

	mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add a user to test with
	time.Sleep(time.Second)
	_, err = mng.AddUser(testuser1, "user 1", testpass1, "")
	require.NoError(t, err)

	// login and get tokens
	_, err = natshubclient.ConnectWithPassword(clientURL, testuser1, "badpass", certBundle.CaCert)

	//pubKey, _ := authBundle.UserKey.PublicKey()
	//authToken, err := cl.NewToken(authBundle.UserID, "badpass", pubKey)
	assert.Error(t, err)
	//assert.Empty(t, authToken)
}

func TestUpdate(t *testing.T) {
	slog.Info("--- TestRefreshFakeToken start")
	defer slog.Info("--- TestRefreshFakeToken end")
	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	// make sure JS is enabled for account
	hc0, err := msgServer.ConnectInProc("test", nil)
	require.NoError(t, err)
	js, err := hc0.JetStream()
	require.NoError(t, err)
	ai, err := js.AccountInfo()
	require.NoError(t, err)
	_ = ai
	hc0.Close()

	mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add user to test with and connect
	tu1Token, tu1Key, err := addNewUser(tu1ID, tu1Name, "pass0", mng)
	hc, err := connectUser(tu1ID, tu1Key, tu1Token)
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

	//reconnect using the new key
	hc, err = connectUser(tu1ID, newPK, newPKPub)
	require.NoError(t, err)
	defer hc.Disconnect()
	cl = authnclient.NewAuthnUserClient(hc)

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

	hc0, err = msgServer.ConnectInProc("test", nil)
	require.NoError(t, err)
	js, err = hc0.JetStream()
	require.NoError(t, err)
	ai, err = js.AccountInfo()
	require.NoError(t, err)
	_ = ai
}
