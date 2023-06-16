package authn_test

import (
	"fmt"
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authn/config"
	"github.com/hiveot/hub/core/authn/service"
	"os"
	"path"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/logging"
)

var passwordFile string // set in TestMain

var tempFolder string
var testuser1 = "testuser1"
var testpass1 = "secret11" // set at start

// create a new authn service and set the password for testuser1
// containing a password for testuser1
func startTestAuthnService() (authSvc authn.IAuthnService, stopFn func(), err error) {
	_ = os.Remove(passwordFile)
	cfg := config.AuthnConfig{
		PasswordFile:            passwordFile,
		AccessTokenValiditySec:  10,
		RefreshTokenValiditySec: 120,
	}
	svc := service.NewAuthnService(cfg)
	err = svc.Start()
	if err == nil {
		testpass1, err = svc.AddUser(testuser1, "")
	}
	if err != nil {
		logrus.Panicf("cant start test authn service: %s", err)
	}

	return svc, func() {
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
	srv, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	assert.NotNil(t, srv)
	time.Sleep(time.Millisecond * 10)
	stopFn()
}

// Create and verify a JWT token
func TestStartTwice(t *testing.T) {
	svc, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	require.NotNil(t, svc)

	stopFn()
}

// Create manage users
func TestManageUser(t *testing.T) {

	svc, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// expect the test user
	userList, err := svc.ListUsers()
	assert.NoError(t, err)
	require.Equal(t, 1, len(userList))

	profile1 := userList[0]
	assert.Equal(t, testuser1, profile1.LoginID)

	// remove user
	err = svc.RemoveUser(testuser1)
	assert.NoError(t, err)
	userList, err = svc.ListUsers()
	assert.NoError(t, err)
	require.Equal(t, 0, len(userList))

	// reset password adds the user again
	newpw, err := svc.ResetPassword(testuser1, "")
	assert.NoError(t, err)
	assert.NotEmpty(t, newpw)
	userList, err = svc.ListUsers()
	assert.NoError(t, err)
	require.Equal(t, 1, len(userList))

	// add existing user should fail
	_, err = svc.AddUser(testuser1, "")
	assert.Error(t, err)

}

func TestLoginRefreshLogout(t *testing.T) {
	var at1 string
	var rt1 string
	var at2 string
	var rt2 string
	count := 100
	svc, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// login and get tokens
	t1 := time.Now()
	for i := 0; i < count; i++ {
		at1, rt1, err = svc.Login(testuser1, testpass1)
	}
	d1 := time.Now().Sub(t1)
	assert.NoError(t, err)
	assert.NotEmpty(t, at1)
	assert.NotEmpty(t, rt1)

	// refresh token
	t2 := time.Now()
	for i := 0; i < count; i++ {
		at2, rt2, err = svc.Refresh(testuser1, rt1)
	}
	d2 := time.Now().Sub(t2)
	fmt.Printf("Time to login   %d times: %d msec\n", count, d1.Milliseconds())
	fmt.Printf("Time to refresh %d times: %d msec\n", count, d2.Milliseconds())
	assert.NoError(t, err)
	assert.NotEmpty(t, at2)
	assert.NotEmpty(t, rt2)

	// logout
	err = svc.Logout(testuser1, rt2)
	assert.NoError(t, err)
	// second logout should not give an error
	err = svc.Logout(testuser1, rt2)
	assert.NoError(t, err)
}

func TestLoginFail(t *testing.T) {
	svc, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// login and get tokens
	accessToken, refreshToken, err := svc.Login(testuser1, "badpass")
	assert.Error(t, err)
	assert.Empty(t, accessToken)
	assert.Empty(t, refreshToken)
}

func TestProfile(t *testing.T) {
	svc, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// after authentication get/set profile and get password should succeed
	at, rt, err := svc.Login(testuser1, testpass1)
	assert.NoError(t, err)
	assert.NotEmpty(t, at)
	assert.NotEmpty(t, rt)

	prof1, err := svc.GetProfile(testuser1)
	assert.NoError(t, err)
	assert.Equal(t, testuser1, prof1.LoginID)

	prof1.Name = "new name"
	err = svc.SetProfile(prof1)
	assert.NoError(t, err)
	err = svc.SetPassword(testuser1, "newpass")
	assert.NoError(t, err)

}
