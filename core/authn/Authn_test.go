package authn_test

import (
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authn/client"
	"github.com/hiveot/hub/core/authn/service"
	"github.com/hiveot/hub/core/authn/service/unpwstore"
	"github.com/hiveot/hub/core/nats"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/nats-io/jwt/v2"
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

var hubServer *nats.HubNatsServer

// create a new authn service and set the password for testuser1
// containing a password for testuser1
func startTestAuthnService() (mngAuthn authn.IManageAuthn, clientAuthn authn.IClientAuthn, stopFn func(), err error) {

	// the password file to use
	passwordFile := path.Join(tempFolder, "test.passwd")

	// TODO: put this in a test environment
	_ = os.Remove(passwordFile)
	cfg := service.NewAuthnConfig(storeFolder)
	cfg.PasswordFile = passwordFile
	cfg.DeviceTokenValidity = 10

	pwStore := unpwstore.NewPasswordFileStore(passwordFile)
	// since it uses jwt auth, only the account key is needed to issue tokens
	svc := service.NewAuthnService(pwStore, authBundle.AppSigningNKey, authBundle.CaCert)
	err = svc.Start()
	if err != nil {
		logrus.Panicf("cant start test authn service: %s", err)
	}
	//
	// Create the authn service for authentication requests
	// This must be an app-account client as it subscribes to app authn messages
	// It can use in-proc connections to reduce latency
	// options: static nkey - added to the server
	hcSvc := hubclient.NewHubClient()
	////err = hcSvc.ConnectWithJWT(clientURL, authBundle.ServiceCreds, authBundle.CaCert)
	////err = hcSvc.ConnectWithCert(clientURL, authn.AuthnServiceName, authBundle.ServerCert, authBundle.CaCert)
	//err = hcSvc.ConnectWithNKey(clientURL, authBundle.ServiceID, authBundle.ServiceNKey, authBundle.CaCert)
	err = hcSvc.ConnectWithNKey(clientURL, authBundle.ServiceID, authBundle.ServiceNKey, authBundle.CaCert)
	if err != nil {
		panic(err)
	}
	natsSvr := service.NewAuthnNatsBinding(authBundle.AppSigningNKey, svc, hcSvc)
	natsSvr.Start()
	verifier := service.NewAuthnNatsVerify(svc)
	hubServer.SetAuthnVerifier(verifier.VerifyAuthnReq)
	//
	//// hook the callout authentication handler
	//err = hcSvc.Subscribe(server.AuthCalloutSubject, natsSvr.HandleCallOut)
	//if err != nil {
	//	panic(err)
	//}
	////--- temp

	// create an hub client for the authn client and management api's
	hcUser := hubclient.NewHubClient()
	//err = hcUser.ConnectWithJWT(clientURL, authBundle.UserCreds, authBundle.CaCert)
	err = hcUser.ConnectWithNKey(clientURL, authBundle.UserID, authBundle.UserNKey, authBundle.CaCert)
	//err = hcUser.ConnectWithPassword(clientURL, authBundle.UserID, "somepass", authBundle.CaCert)
	if err != nil {
		panic(err)
	}
	mngAuthn = client.NewManageAuthn("service1", hcUser)
	clientAuthn = client.NewClientAuthn("service1", hcUser)
	//
	return mngAuthn, clientAuthn, func() {
		_ = svc.Stop()
		natsSvr.Stop()
		hcSvc.Disconnect()
		hcUser.Disconnect()
		// let background tasks finish
		time.Sleep(time.Millisecond * 10)
	}, err
}

// TestMain creates a test environment
// Used for all test cases in this package
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")

	// a working folder for the data
	tempFolder = path.Join(os.TempDir(), "test-authn")
	_ = os.MkdirAll(tempFolder, 0700)
	authBundle = testenv.CreateTestAuthBundle()
	// run the test server
	hubServer = nats.NewHubNatsServer(
		"AppAccount",
		authBundle.AppAccountNKey,
		"/tmp/nats-server",
		"127.0.0.1", 9990,
		authBundle.ServerCert, authBundle.CaCert,
		nil)

	//cURL, err := testServer.Start("", 9999, authBundle.ServerCert, authBundle.CaCert)
	//testServer := testenv.NewTestCalloutServer(
	//	authBundle.AppAccountNKey,
	//	authBundle.SystemSigningNKey,
	//	authBundle.SystemUserNKey,
	//	authBundle.ServerCert, authBundle.CaCert)

	cURL, err := hubServer.Start()
	clientURL = cURL
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
	mng, cl, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	assert.NotNil(t, mng)
	assert.NotNil(t, cl)
	time.Sleep(time.Millisecond * 10)
	stopFn()
	slog.Info("--- TestStartStop end")
}

func TestLoginWithPassword(t *testing.T) {
	const testuser2 = "user2"
	const testpass2 = "pass2"
	slog.Info("--- TestLoginWithPassword start")
	mng, _, stopFn, err := startTestAuthnService()
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

// Create manage users
func TestMultiRequests(t *testing.T) {
	slog.Info("--- TestMultiRequests start")
	mng, _, stopFn, err := startTestAuthnService()
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

	mng, _, stopFn, err = startTestAuthnService()
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

	mng, cl, stopFn, err := startTestAuthnService()
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

	// manages can change password of the remaining user
	newPw := "newpass"
	err = cl.UpdatePassword(testuser2, newPw)
	require.NoError(t, err)

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
	mng, _, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add users to test with
	err = mng.AddUser(testuser1, "testuser 1", testpass1)
	require.NoError(t, err)
	err = mng.AddUser(authBundle.UserID, "user one", "")
	require.NoError(t, err)

	// login and get profile
	hc1 := hubclient.NewHubClient()
	cl1 := client.NewClientAuthn(authBundle.ServiceID, hc1)
	defer hc1.Disconnect()
	err = hc1.ConnectWithPassword(clientURL, testuser1, testpass1, authBundle.CaCert)
	require.NoError(t, err)
	prof1, err := cl1.GetProfile("")
	require.NoError(t, err)
	require.NotEmpty(t, prof1)

	// request a new login token
	userPub, _ := authBundle.UserNKey.PublicKey()
	userSeed, _ := authBundle.UserNKey.Seed()
	authToken1, err = cl1.NewToken(testuser1, testpass1, userPub)
	assert.NoError(t, err)
	assert.NotEmpty(t, authToken1)

	// login with the new token
	hc2 := hubclient.NewHubClient()
	cl2 := client.NewClientAuthn(authBundle.ServiceID, hc2)
	defer hc2.Disconnect()
	authToken1Cred, err := jwt.FormatUserConfig(authToken1, userSeed)
	_ = authToken1Cred
	require.NoError(t, err)
	err = hc2.ConnectWithJWT(clientURL, authBundle.UserCreds, authBundle.CaCert)
	//err = hc2.ConnectWithJWT(clientURL, authToken1Cred, authBundle.CaCert)
	require.NoError(t, err)
	prof2, err := cl2.GetProfile("")
	require.NoError(t, err)
	require.NotEmpty(t, prof2)

	// refresh token
	signedJWT, err := jwt.ParseDecoratedJWT([]byte(authToken1))
	authToken2, err = cl1.Refresh(authBundle.UserID, signedJWT)
	assert.NoError(t, err)
	assert.NotEmpty(t, authToken2)

	//--- just testing, remove after
	// ??? WHY DOES THIS WORK?
	//hc5 := hubclient.NewHubClient()
	//err = hc5.ConnectWithJWT(clientURL, authBundle.DeviceCreds, authBundle.CaCert)
	//require.NoError(t, err)
	//at5b,err := authBundle.DeviceJWT
	//authToken3, err := cl.Refresh(authBundle.UserID, authToken2)
	//require.NoError(t, err)
	//hc5.Disconnect()

	//--- just testing end

	// login with new token and get profile
	//hc2 = hubclient.NewHubClient()
	//defer hc2.Disconnect()
	//seed2, _ := authBundle.UserNKey.Seed()
	//authToken2Cred, err := jwt.FormatUserConfig(authToken2, seed2)
	//require.NoError(t, err)
	//err = hc2.ConnectWithJWT(clientURL, authToken2Cred, authBundle.CaCert)
	//require.NoError(t, err)
	//cl2 := client.NewClientAuthn(authBundle.ServiceID, hc2)
	//prof2, err := cl2.GetProfile(authBundle.UserID)
	//
	//require.NoError(t, err)
	//require.NotEmpty(t, prof2)

	slog.Info("--- TestLoginRefresh end")
}

func TestLoginFail(t *testing.T) {
	slog.Info("--- TestLoginFail start")
	var testuser1 = "testuser1"
	var testpass1 = "testpass1"

	mng, _, stopFn, err := startTestAuthnService()
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
	mng, _, stopFn, err := startTestAuthnService()
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
