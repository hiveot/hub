package authn_test

import (
	"fmt"
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authn/service"
	"github.com/hiveot/hub/core/authn/service/unpwstore"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/nats-io/jwt/v2"
	"os"
	"path"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/logging"
)

var storeFolder string  // set in TestMain
var passwordFile string // set in TestMain

var tempFolder string

// var testuser1 = "testuser1"
var testpass1 = "secret11" // set at start
var testCerts testenv.TestCerts

// create a new authn service and set the password for testuser1
// containing a password for testuser1
func startTestAuthnService() (mngAuthn authn.IManageAuthn, clientAuthn authn.IClientAuthn, stopFn func(), err error) {
	_ = os.Remove(passwordFile)
	cfg := service.NewAuthnConfig(storeFolder)
	cfg.PasswordFile = passwordFile
	cfg.DeviceTokenValidity = 10
	pwStore := unpwstore.NewPasswordFileStore(passwordFile)
	svc := service.NewAuthnService(pwStore, testCerts.AccountNKey)
	err = svc.Start()
	if err == nil {
		err = svc.AddUser(testCerts.UserID, "test user 1", testpass1)
	}
	if err != nil {
		logrus.Panicf("cant start test authn service: %s", err)
	}

	return svc, svc, func() {
		_ = svc.Stop()
	}, err
}

// TestMain creates a test environment
// Used for all test cases in this package
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")

	// a working folder for the data
	tempFolder = path.Join(os.TempDir(), "test-authn")
	_ = os.MkdirAll(tempFolder, 0700)
	testCerts = testenv.CreateAuthBundle()

	// the password file to use
	passwordFile = path.Join(tempFolder, "test.passwd")

	res := m.Run()

	time.Sleep(time.Second)
	if res == 0 {
		_ = os.RemoveAll(tempFolder)
	}
	os.Exit(res)
}

// Create and verify a JWT token
func TestStartStop(t *testing.T) {
	mng, cl, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	assert.NotNil(t, mng)
	assert.NotNil(t, cl)
	time.Sleep(time.Millisecond * 10)
	stopFn()
}

// Create and verify a JWT token
func TestStartTwice(t *testing.T) {
	mng, cl, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	require.NotNil(t, mng)
	require.NotNil(t, cl)
	// todo

	stopFn()
}

// Create manage users
func TestManageUser(t *testing.T) {

	mng, _, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// expect the test user
	userList, err := mng.ListClients()
	assert.NoError(t, err)
	require.Equal(t, 1, len(userList))

	profile1 := userList[0]
	assert.Equal(t, testCerts.UserID, profile1.ClientID)

	// remove user
	err = mng.RemoveClient(testCerts.UserID)
	assert.NoError(t, err)
	userList, err = mng.ListClients()
	assert.NoError(t, err)
	require.Equal(t, 0, len(userList))

	// reset password adds the user again
	newPw := "newpass"
	err = mng.ResetPassword(testCerts.UserID, newPw)
	assert.NoError(t, err)
	assert.NotEmpty(t, newPw)
	userList, err = mng.ListClients()
	assert.NoError(t, err)
	require.Equal(t, 1, len(userList))

	// add existing user should fail
	err = mng.AddUser(testCerts.UserID, "", "")
	assert.Error(t, err)
}

func TestLoginRefresh(t *testing.T) {
	var authToken1 string
	var authToken2 string
	count := 100
	_, cl, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// login and get tokens
	t1 := time.Now()
	for i := 0; i < count; i++ {
		pubKey, _ := testCerts.UserNKey.PublicKey()
		authToken1, err = cl.NewToken(testCerts.UserID, testpass1, pubKey)
	}
	d1 := time.Now().Sub(t1)
	assert.NoError(t, err)
	assert.NotEmpty(t, authToken1)

	// refresh token
	t2 := time.Now()
	signedJWT, err := jwt.ParseDecoratedJWT([]byte(authToken1))
	for i := 0; i < count; i++ {
		authToken2, err = cl.Refresh(testCerts.UserID, signedJWT)
	}
	d2 := time.Now().Sub(t2)
	fmt.Printf("Time to login   %d times: %d msec\n", count, d1.Milliseconds())
	fmt.Printf("Time to refresh %d times: %d msec\n", count, d2.Milliseconds())
	assert.NoError(t, err)
	assert.NotEmpty(t, authToken2)

}

func TestLoginFail(t *testing.T) {
	_, cl, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// login and get tokens
	pubKey, _ := testCerts.UserNKey.PublicKey()
	authToken, err := cl.NewToken(testCerts.UserID, "badpass", pubKey)
	assert.Error(t, err)
	assert.Empty(t, authToken)
}

func TestProfile(t *testing.T) {
	mng, cl, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// after authentication get/set profile and get password should succeed
	pubKey, _ := testCerts.UserNKey.PublicKey()
	authToken, err := cl.NewToken(testCerts.UserID, testpass1, pubKey)
	assert.NoError(t, err)
	assert.NotEmpty(t, authToken)

	prof1, err := mng.GetProfile(testCerts.UserID)
	assert.NoError(t, err)
	assert.Equal(t, testCerts.UserID, prof1.ClientID)

	prof1.Name = "new name"
	err = cl.UpdateName(testCerts.UserID, prof1.Name)
	assert.NoError(t, err)
	err = cl.UpdatePassword(testCerts.UserID, "newpass")
	assert.NoError(t, err)

}
