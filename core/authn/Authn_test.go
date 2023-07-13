package authn_test

import (
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authn/client"
	"github.com/hiveot/hub/core/authn/service"
	"github.com/hiveot/hub/core/authn/service/unpwstore"
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

var testCerts testenv.TestAuthBundle

// clientURL is set by the testmain
var clientURL string

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
	svc := service.NewAuthnService(pwStore, testCerts.AppAccountNKey)
	err = svc.Start()
	if err != nil {
		logrus.Panicf("cant start test authn service: %s", err)
	}

	// to use nats as an RPC, connect as a service
	hcSvc := hubclient.NewHubClientNats()
	err = hcSvc.ConnectWithJWT(clientURL, testCerts.ServiceCreds, testCerts.CaCert)
	//err = hcSvc.ConnectWithCert(clientURL, authn.AuthnServiceName, testCerts.ServerCert, testCerts.CaCert)
	//err = hcSvc.ConnectWithNKey(clientURL, testCerts.ServiceNKey, testCerts.CaCert)
	if err != nil {
		panic(err)
	}

	natsSvr := service.NewAuthnNatsServer(svc, hcSvc)
	natsSvr.Start()

	// use the client API with admin credentials to take the place of the direct service API
	hcUser := hubclient.NewHubClientNats()
	err = hcUser.ConnectWithJWT(clientURL, testCerts.UserCreds, testCerts.CaCert)
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
		time.Sleep(time.Second)
	}, err
}

// TestMain creates a test environment
// Used for all test cases in this package
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")

	// a working folder for the data
	tempFolder = path.Join(os.TempDir(), "test-authn")
	_ = os.MkdirAll(tempFolder, 0700)
	testCerts = testenv.CreateTestAuthBundle()
	// run the test server
	testServer := testenv.NewTestServer(testCerts.ServerCert, testCerts.CaCert)
	// oddly enough the err is not assigned if it is an existing variable
	cURL, err := testServer.Start(testCerts)
	clientURL = cURL
	if err != nil {
		panic(err)
	}

	res := m.Run()

	testServer.Stop()
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

	// add a user to test with
	err = mng.AddUser(testuser1, "user 1", testpass1)
	require.NoError(t, err)
	// login and get tokens
	hc := hubclient.NewHubClientNats()
	err = hc.ConnectWithPassword(clientURL, testuser1, testpass1, testCerts.CaCert)
	require.NoError(t, err)
	cl := client.NewClientAuthn("service1", hc)
	pubKey, _ := testCerts.UserNKey.PublicKey()
	authToken1, err = cl.NewToken(testuser1, testpass1, pubKey)

	assert.NoError(t, err)
	assert.NotEmpty(t, authToken1)

	// refresh token
	signedJWT, err := jwt.ParseDecoratedJWT([]byte(authToken1))
	authToken2, err = cl.Refresh(testCerts.UserID, signedJWT)
	assert.NoError(t, err)
	assert.NotEmpty(t, authToken2)
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
	hc := hubclient.NewHubClientNats()
	err = hc.ConnectWithPassword(clientURL, testuser1, "badpass", testCerts.CaCert)

	//pubKey, _ := testCerts.UserNKey.PublicKey()
	//authToken, err := cl.NewToken(testCerts.UserID, "badpass", pubKey)
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
	hc := hubclient.NewHubClientNats()
	//err = hc.ConnectWithPassword(clientURL, testuser1, testpass1, testCerts.CaCert)
	err = hc.ConnectUnauthenticated(clientURL, testCerts.CaCert)
	require.NoError(t, err)
	cl := client.NewClientAuthn("service1", hc)

	prof2, err := cl.GetProfile(testuser1)
	require.NoError(t, err)
	assert.Equal(t, prof1.ClientID, prof2.ClientID)
	assert.Equal(t, prof1.Name, prof2.Name)
	slog.Info("--- TestGetProfile end")
}
