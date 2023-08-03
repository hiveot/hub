package authn_test

import (
	authn2 "github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authn/authnclient"
	"github.com/hiveot/hub/core/authn/authnservice"
	"github.com/hiveot/hub/core/authn/authnstore"
	"github.com/hiveot/hub/core/authn/natsauthn"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/hiveot/hub/core/server"
	"github.com/hiveot/hub/core/server/natsserver"
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

var storeFolder string // set in TestMain
var tempFolder string

var authBundle testenv.TestAuthBundle

// clientURL is set by the testmain
var clientURL string

var hubServer *natsserver.HubNatsServer

// run the test for different cores
var useCore = "nats" // nats vs mqtt

// launch the authn service and return a client for using and managing it.
func startTestAuthnService() (cl authn2.IAuthnUser, mng authn2.IAuthnManage, stopFn func(), err error) {

	// the password file to use
	passwordFile := path.Join(tempFolder, "test.passwd")

	// TODO: put this in a test environment
	_ = os.Remove(passwordFile)
	cfg := authn.AuthnConfig{}
	_ = cfg.InitConfig("", storeFolder)
	cfg.PasswordFile = passwordFile
	cfg.DeviceTokenValidity = 10

	// setup the authn service
	authStore := authnstore.NewAuthnFileStore(passwordFile)
	tokenizer := natsauthn.NewNatsTokenizer(authBundle.AppAccountNKey)
	hcSvc := natshubclient.NewHubClient(authBundle.ServiceNKey)
	err = hcSvc.ConnectWithJWT(clientURL, authBundle.ServiceJWT, authBundle.CaCert)
	if err != nil {
		panic("can't connect authn to server: " + err.Error())
	}
	authnSvc := authnservice.NewAuthnService(authStore, tokenizer, hcSvc)
	err = authnSvc.Start()
	if err != nil {
		logrus.Panicf("cant start test authn service: %s", err)
	}

	//--- create a hub client for the authn client and management api's
	hcClient := natshubclient.NewHubClient(authBundle.ServiceNKey)
	//err = hcAuthn.ConnectWithNKey(clientURL, authBundle.ServiceID, authBundle.ServiceNKey, authBundle.CaCert)
	err = hcClient.ConnectWithJWT(clientURL, authBundle.ServiceJWT, authBundle.CaCert)
	if err != nil {
		panic(err)
	}
	mngAuthn := authnclient.NewAuthnManageClient(authBundle.ServiceID, hcClient)
	clientAuthn := authnclient.NewAuthnUserClient(authBundle.ServiceID, hcClient)
	//
	return clientAuthn, mngAuthn, func() {
		authnSvc.Stop()
		hcSvc.Disconnect()
		hcClient.Disconnect()
		// let background tasks finish
		time.Sleep(time.Millisecond * 10)
	}, err
}

// TestMain creates a test environment
// Used for all test cases in this package
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	var err error

	// a working folder for the data
	tempFolder = path.Join(os.TempDir(), "test-authn")
	_ = os.MkdirAll(tempFolder, 0700)
	authBundle = testenv.CreateTestAuthBundle()
	// run the test server
	serverCfg := &server.ServerConfig{
		Host: "127.0.0.1",
		Port: 9990,
		//AppAccountName: authBundle.AppAccountName,
	}

	serverCfg.InitConfig("", storeFolder)
	hubServer = natsserver.NewHubNatsServer(
		authBundle.ServerCert, authBundle.CaCert)

	serverOpts := hubServer.CreateServerConfig(
		serverCfg, authBundle.OperatorJWT, authBundle.SystemAccountJWT,
		authBundle.AppAccountJWT, authBundle.ServiceNKey)

	clientURL, err = hubServer.Start(serverOpts)
	if err != nil {
		panic(err)
	}
	res := m.Run()

	hubServer.Stop()
	time.Sleep(time.Second)
	if res == 0 {
		_ = os.RemoveAll(tempFolder)
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

func TestLoginWithPassword(t *testing.T) {
	const testuser2 = "user2"
	const testpass2 = "pass2"
	slog.Info("--- TestLoginWithPassword start")
	_, mng, stopFn, err := startTestAuthnService()
	require.NoError(t, err)

	err = mng.AddUser(testuser2, "another user", testpass2)
	assert.NoError(t, err)

	//token, err := cl.Login(testuser2, testpass2)
	hc1 := natshubclient.NewHubClient(nil)
	token, err := hc1.ConnectWithPassword(clientURL, testuser2, testpass2, authBundle.CaCert)
	err = hc1.ConnectWithJWT(clientURL, token, authBundle.CaCert)
	require.NoError(t, err)
	hc1.Disconnect()

	_, err = hc1.ConnectWithPassword(clientURL, testuser2, "wrongpass", authBundle.CaCert)
	require.Error(t, err)
	hc1.Disconnect()

	stopFn()
}

func TestLoginWithJWT(t *testing.T) {
	slog.Info("--- TestLoginWithJWT start")
	_, mng, stopFn, err := startTestAuthnService()
	require.NoError(t, err)

	err = mng.AddUser(authBundle.UserID, "a jwt user", "")
	assert.NoError(t, err)

	hc1 := natshubclient.NewHubClient(nil)
	err = hc1.ConnectWithJWT(clientURL, authBundle.UserJWT, authBundle.CaCert)
	assert.NoError(t, err)
	hc1.Disconnect()

	// improper signed token should fail
	appAcctPub, _ := authBundle.AppAccountNKey.PublicKey()
	signerKey, _ := nkeys.CreateAccount()
	_, badCreds := testenv.CreateUserCreds(
		authBundle.UserID,
		authBundle.UserNKey,
		signerKey,
		appAcctPub,
		authBundle.AppAccountName, nil, nil) // sign itself
	token, _ := nkeys.ParseDecoratedJWT(badCreds)
	err = hc1.ConnectWithJWT(clientURL, token, authBundle.CaCert)
	require.Error(t, err)

	stopFn()
	slog.Info("--- TestLoginWithJWT end")
}

// Create manage users
func TestMultiRequests(t *testing.T) {
	slog.Info("--- TestMultiRequests start")
	_, mng, stopFn, err := startTestAuthnService()
	require.NoError(t, err)

	err = mng.AddUser("user1", "user 1", "pass1")
	assert.NoError(t, err)
	err = mng.AddUser("user2", "user 2", "pass2")
	assert.NoError(t, err)
	err = mng.AddUser("user3", "user 3", "pass3")
	assert.NoError(t, err)
	err = mng.AddUser("user4", "user 4", "pass4")
	assert.NoError(t, err)

	stopFn()

	_, err = mng.ListClients()
	assert.Error(t, err)

	_, mng, stopFn, err = startTestAuthnService()
	require.NoError(t, err)
	clList, err := mng.ListClients()
	assert.Equal(t, 0, len(clList))

	err = mng.AddUser("user1", "user 1", "pass1")
	assert.NoError(t, err)
	err = mng.AddUser("user2", "user 2", "pass2")
	assert.NoError(t, err)

	stopFn()
	slog.Info("--- TestMultiRequests stop")
}

// Create manage users
func TestManageUser(t *testing.T) {
	slog.Info("--- TestManageUser start")
	var testuser1 = "testuser1"
	var testuser2 = "testuser2"
	var testpass1 = "testpass1"
	var testpass2 = "testpass2"

	_, mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	userList, err := mng.ListClients()
	require.NoError(t, err)
	assert.Equal(t, 0, len(userList))

	// add two users
	err = mng.AddUser(testuser1, "user 1", testpass1)
	require.NoError(t, err)
	err = mng.AddUser(testuser2, "user 2", testpass2)
	require.NoError(t, err)
	userList, err = mng.ListClients()
	assert.NoError(t, err)
	require.Equal(t, 2, len(userList))

	// remove one user
	err = mng.RemoveClient(testuser1)
	assert.NoError(t, err)
	userList, err = mng.ListClients()
	assert.NoError(t, err)
	require.Equal(t, 1, len(userList))

	// add existing user should fail
	err = mng.AddUser(testuser2, "", testpass2)
	assert.Error(t, err)
	slog.Info("--- TestManageUser end")
}

func TestLoginRefresh(t *testing.T) {
	slog.Info("--- TestLoginRefresh start")
	var testuser1 = "testuser1"
	var testpass1 = "testpass1"
	var authToken1 string
	var authToken2 string

	_, mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add users to test with
	err = mng.AddUser(testuser1, "testuser 1", testpass1)
	require.NoError(t, err)
	err = mng.AddUser(authBundle.UserID, "user one", "")
	require.NoError(t, err)

	// 1. login with password
	key1, _ := nkeys.CreateUser()
	hc1 := natshubclient.NewHubClient(key1)
	defer hc1.Disconnect()
	token, err := hc1.ConnectWithPassword(clientURL, testuser1, testpass1, authBundle.CaCert)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// 2. request a new login token
	// use the authentication client to request a new token
	cl1 := authnclient.NewAuthnUserClient(authBundle.ServiceID, hc1)
	userKey, _ := nkeys.CreateUser()
	userPub, _ := userKey.PublicKey()
	authToken1, err = cl1.Login(testuser1, testpass1)
	require.NoError(t, err)
	assert.NotEmpty(t, authToken1)

	// 3. login with the new token
	hc2 := natshubclient.NewHubClient(key1)
	cl2 := authnclient.NewAuthnUserClient(authBundle.ServiceID, hc2)
	defer hc2.Disconnect()

	// create a new key pair using the returned token and private key
	//userSeed, _ := userKey.Seed()
	//authToken1Cred, err := jwt.FormatUserConfig(authToken1, userSeed)
	require.NoError(t, err)
	err = hc2.ConnectWithJWT(clientURL, authToken1, authBundle.CaCert)
	require.NoError(t, err)
	prof2, err := cl2.GetProfile("")
	require.NoError(t, err)
	require.Equal(t, testuser1, prof2.ClientID)

	// 4. Obtain a refresh token using the new token
	authToken2, err = cl2.Refresh(authBundle.UserID, authToken1)
	assert.NoError(t, err)
	assert.NotEmpty(t, authToken2)

	// 5. login with the refresh token
	cl3 := natshubclient.NewHubClient(key1)
	//authToken2Cred, err := jwt.FormatUserConfig(authToken2, userSeed)
	err = cl3.ConnectWithJWT(clientURL, authToken2, authBundle.CaCert)
	require.NoError(t, err)
	prof3, err := authnclient.NewAuthnUserClient(authBundle.ServiceID, cl3).GetProfile(authBundle.UserID)
	cl3.Disconnect()
	require.NoError(t, err)
	require.NotEmpty(t, prof3)
	require.Equal(t, testuser1, prof3.ClientID)

	// 6. login with a forged token should fail
	cl4 := natshubclient.NewHubClient(key1)
	appAcctPub, _ := authBundle.AppAccountNKey.PublicKey()
	fakeAcct, _ := nkeys.CreateAccount()
	forgedClaims := jwt.NewUserClaims(userPub)
	forgedClaims.Subject = userPub
	forgedClaims.Issuer = appAcctPub
	forgedJWT, err := forgedClaims.Encode(fakeAcct) // <- forged
	require.NoError(t, err)
	//forgedCred, err := jwt.FormatUserConfig(forgedJWT, userSeed)

	err = cl4.ConnectWithJWT(clientURL, forgedJWT, authBundle.CaCert)
	require.Error(t, err)

	slog.Info("--- TestLoginRefresh end")
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
	err = mng.AddUser(testuser1, "user 1", testpass1)
	require.NoError(t, err)

	// login and get tokens
	hc := natshubclient.NewHubClient(nil)
	_, err = hc.ConnectWithPassword(clientURL, testuser1, "badpass", authBundle.CaCert)

	//pubKey, _ := authBundle.UserNKey.PublicKey()
	//authToken, err := cl.NewToken(authBundle.UserID, "badpass", pubKey)
	assert.Error(t, err)
	//assert.Empty(t, authToken)
	slog.Info("--- TestLoginFail end")
}

func TestGetProfile(t *testing.T) {
	slog.Info("--- TestGetProfile start")
	var testuser1 = "testuser1"
	var testpass1 = "testpass1"
	_, mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add a user to test with
	err = mng.AddUser(testuser1, "user 1", testpass1)
	require.NoError(t, err)

	// after authentication get/set profile and get password should succeed
	prof1, err := mng.GetClientProfile(testuser1)
	assert.NoError(t, err)
	assert.Equal(t, testuser1, prof1.ClientID)
	assert.Equal(t, "user 1", prof1.DisplayName)

	// login and get profile as a user
	hc := natshubclient.NewHubClient(nil)
	_, err = hc.ConnectWithPassword(clientURL, testuser1, testpass1, authBundle.CaCert)
	require.NoError(t, err)

	cl := authnclient.NewAuthnUserClient(authBundle.ServiceID, hc)

	prof2, err := cl.GetProfile(testuser1)
	require.NoError(t, err)
	assert.Equal(t, prof1.ClientID, prof2.ClientID)
	assert.Equal(t, prof1.DisplayName, prof2.DisplayName)
	slog.Info("--- TestGetProfile end")
}
