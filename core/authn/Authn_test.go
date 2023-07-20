package authn_test

import (
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authn/client"
	"github.com/hiveot/hub/core/authn/service"
	"github.com/hiveot/hub/core/authn/service/unpwstore"
	"github.com/hiveot/hub/core/config"
	"github.com/hiveot/hub/core/server"
	"github.com/hiveot/hub/lib/hubclient"
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

var hubServer *server.HubNatsServer

// create a new authn service and set the password for testuser1
// containing a password for testuser1
func startTestAuthnService() (mng authn.IManageAuthn, stopFn func(), err error) {

	// the password file to use
	passwordFile := path.Join(tempFolder, "test.passwd")

	// TODO: put this in a test environment
	_ = os.Remove(passwordFile)
	cfg := config.NewAuthnConfig(storeFolder)
	cfg.PasswordFile = passwordFile
	cfg.DeviceTokenValidity = 10

	pwStore := unpwstore.NewPasswordFileStore(passwordFile)
	svc := service.NewAuthnService(
		authBundle.AppAccountName,
		authBundle.AppAccountNKey,
		pwStore,
		authBundle.CaCert,
	)
	err = svc.Start()
	if err != nil {
		logrus.Panicf("cant start test authn service: %s", err)
	}
	// Create the authn service for authentication requests
	// This must be an app-account client as it subscribes to app authn messages
	// It can use in-proc connections to reduce latency
	// the binding service needs a server connection
	hcSvc := hubclient.NewHubClient()
	err = hcSvc.ConnectWithNKey(clientURL, authBundle.ServiceID, authBundle.ServiceNKey, authBundle.CaCert)
	if err != nil {
		panic("can't connect authn to server: " + err.Error())
	}
	authnBinding := service.NewAuthnNatsBinding(svc)
	authnBinding.Start(hcSvc)
	verifier := service.NewAuthnNatsVerify(svc)
	hubServer.SetAuthnVerifier(verifier.VerifyAuthnReq)

	// create an hub client for the authn client and management api's
	hcAuthn := hubclient.NewHubClient()
	err = hcAuthn.ConnectWithNKey(clientURL, authBundle.ServiceID, authBundle.ServiceNKey, authBundle.CaCert)
	if err != nil {
		panic(err)
	}
	mngAuthn := client.NewManageAuthn(authBundle.ServiceID, hcAuthn)
	//clientAuthn = client.NewClientAuthn(hcAuthn)
	//
	return mngAuthn, func() {
		_ = svc.Stop()
		authnBinding.Stop()
		hcSvc.Disconnect()
		hcAuthn.Disconnect()
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
	hubServer = server.NewHubNatsServer(
		&config.ServerConfig{
			Host:           "127.0.0.1",
			Port:           9990,
			StoresDir:      "/tmp/nats-server",
			AppAccountName: authBundle.AppAccountName,
			AppAccountKey:  authBundle.AppAccountNKey,
			CaCert:         authBundle.CaCert,
			ServerCert:     authBundle.ServerCert,
		})

	clientURL, err = hubServer.Start()
	if err != nil {
		panic(err)
	}
	// authn service uses the service key to connect
	err = hubServer.AddServiceKey(authBundle.ServiceNKey)
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
	_, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 10)
	stopFn()
	slog.Info("--- TestStartStop end")
}

func TestLoginWithPassword(t *testing.T) {
	const testuser2 = "user2"
	const testpass2 = "pass2"
	slog.Info("--- TestLoginWithPassword start")
	mng, stopFn, err := startTestAuthnService()
	require.NoError(t, err)

	err = mng.AddUser(testuser2, "another user", testpass2)
	assert.NoError(t, err)

	hc1 := hubclient.NewHubClient()
	err = hc1.ConnectWithPassword(clientURL, testuser2, testpass2, authBundle.CaCert)
	require.NoError(t, err)
	hc1.Disconnect()

	err = hc1.ConnectWithPassword(clientURL, testuser2, "wrongpass", authBundle.CaCert)
	require.Error(t, err)
	hc1.Disconnect()

	stopFn()
}

func TestLoginWithJWT(t *testing.T) {
	slog.Info("--- TestLoginWithJWT start")
	mng, stopFn, err := startTestAuthnService()
	require.NoError(t, err)

	err = mng.AddUser(authBundle.UserID, "a jwt user", "")
	assert.NoError(t, err)

	hc1 := hubclient.NewHubClient()
	err = hc1.ConnectWithJWT(clientURL, authBundle.UserCreds, authBundle.CaCert)
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
	err = hc1.ConnectWithJWT(clientURL, badCreds, authBundle.CaCert)
	require.Error(t, err)

	stopFn()
	slog.Info("--- TestLoginWithJWT end")
}

// Create manage users
func TestMultiRequests(t *testing.T) {
	slog.Info("--- TestMultiRequests start")
	mng, stopFn, err := startTestAuthnService()
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

	mng, stopFn, err = startTestAuthnService()
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

	mng, stopFn, err := startTestAuthnService()
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
	mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add users to test with
	err = mng.AddUser(testuser1, "testuser 1", testpass1)
	require.NoError(t, err)
	err = mng.AddUser(authBundle.UserID, "user one", "")
	require.NoError(t, err)

	// 1. login with password
	hc1 := hubclient.NewHubClient()
	defer hc1.Disconnect()
	err = hc1.ConnectWithPassword(clientURL, testuser1, testpass1, authBundle.CaCert)
	require.NoError(t, err)

	// 2. request a new login token
	// use the authentication client to request a new token
	cl1 := client.NewClientAuthn(authBundle.ServiceID, hc1)
	userKey, _ := nkeys.CreateUser()
	userPub, _ := userKey.PublicKey()
	authToken1, err = cl1.NewToken(testuser1, testpass1, userPub)
	require.NoError(t, err)
	assert.NotEmpty(t, authToken1)

	// 3. login with the new token
	userSeed, _ := userKey.Seed()
	hc2 := hubclient.NewHubClient()
	cl2 := client.NewClientAuthn(authBundle.ServiceID, hc2)
	defer hc2.Disconnect()

	// create a new key pair using the returned token and private key
	authToken1Cred, err := jwt.FormatUserConfig(authToken1, userSeed)
	require.NoError(t, err)
	err = hc2.ConnectWithJWT(clientURL, authToken1Cred, authBundle.CaCert)
	require.NoError(t, err)
	prof2, err := cl2.GetProfile("")
	require.NoError(t, err)
	require.Equal(t, testuser1, prof2.ClientID)

	// 4. Obtain a refresh token using the new token
	authToken2, err = cl2.Refresh(authBundle.UserID, authToken1)
	assert.NoError(t, err)
	assert.NotEmpty(t, authToken2)

	// 5. login with the refresh token
	cl3 := hubclient.NewHubClient()
	authToken2Cred, err := jwt.FormatUserConfig(authToken2, userSeed)
	err = cl3.ConnectWithJWT(clientURL, authToken2Cred, authBundle.CaCert)
	require.NoError(t, err)
	prof3, err := client.NewClientAuthn(authBundle.ServiceID, cl3).GetProfile(authBundle.UserID)
	cl3.Disconnect()
	require.NoError(t, err)
	require.NotEmpty(t, prof3)
	require.Equal(t, testuser1, prof3.ClientID)

	// 6. login with a forged token should fail
	cl4 := hubclient.NewHubClient()
	appAcctPub, _ := authBundle.AppAccountNKey.PublicKey()
	fakeAcct, _ := nkeys.CreateAccount()
	forgedClaims := jwt.NewUserClaims(userPub)
	forgedClaims.Subject = userPub
	forgedClaims.Issuer = appAcctPub
	forgedJWT, err := forgedClaims.Encode(fakeAcct) // <- forged
	require.NoError(t, err)
	forgedCred, err := jwt.FormatUserConfig(forgedJWT, userSeed)

	err = cl4.ConnectWithJWT(clientURL, forgedCred, authBundle.CaCert)
	require.Error(t, err)

	slog.Info("--- TestLoginRefresh end")
}

func TestLoginFail(t *testing.T) {
	slog.Info("--- TestLoginFail start")
	var testuser1 = "testuser1"
	var testpass1 = "testpass1"

	mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add a user to test with
	time.Sleep(time.Second)
	err = mng.AddUser(testuser1, "user 1", testpass1)
	require.NoError(t, err)

	// login and get tokens
	hc := hubclient.NewHubClient()
	err = hc.ConnectWithPassword(clientURL, testuser1, "badpass", authBundle.CaCert)

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
	mng, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add a user to test with
	err = mng.AddUser(testuser1, "user 1", testpass1)
	require.NoError(t, err)

	// after authentication get/set profile and get password should succeed
	prof1, err := mng.GetClientProfile(testuser1)
	assert.NoError(t, err)
	assert.Equal(t, testuser1, prof1.ClientID)
	assert.Equal(t, "user 1", prof1.Name)

	// login and get profile as a user
	hc := hubclient.NewHubClient()
	err = hc.ConnectWithPassword(clientURL, testuser1, testpass1, authBundle.CaCert)
	require.NoError(t, err)
	cl := client.NewClientAuthn(authBundle.ServiceID, hc)

	prof2, err := cl.GetProfile(testuser1)
	require.NoError(t, err)
	assert.Equal(t, prof1.ClientID, prof2.ClientID)
	assert.Equal(t, prof1.Name, prof2.Name)
	slog.Info("--- TestGetProfile end")
}
