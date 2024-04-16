package authn_test

import (
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn"
	"github.com/hiveot/hub/runtime/authn/authenticator"
	"github.com/hiveot/hub/runtime/authn/authnstore"
	"github.com/hiveot/hub/runtime/authn/service"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/logging"
)

var certBundle = certs.CreateTestCertBundle()
var testDir = path.Join(os.TempDir(), "test-authn")
var authnConfig authn.AuthnConfig
var defaultHash = authn.PWHASH_ARGON2id

// launch the authn client management service and return a client for using and managing it.
// the messaging server is already running (see TestMain)
func startTestAuthnClientService() (
	authnSvc *service.AuthnClientService, store api.IAuthnStore, stopFn func()) {

	// the password file to use
	passwordFile := path.Join(testDir, "test.passwd")
	authnStore := authnstore.NewAuthnFileStore(passwordFile, defaultHash)
	err := authnStore.Open()
	if err != nil {
		panic(err.Error())
	}

	// TODO: put this in a test environment
	_ = os.Remove(passwordFile)
	authnConfig = authn.NewAuthnConfig()
	authnConfig.Setup(testDir, testDir)
	authnConfig.PasswordFile = passwordFile
	authnConfig.AgentTokenValiditySec = 100
	authnConfig.Encryption = defaultHash

	// before being able to connect, the AuthService and its key must be known
	sessionAuth := authenticator.NewJWTAuthenticatorFromFile(
		authnStore, authnConfig.KeysDir, authnConfig.DefaultKeyType)
	authnClientSvc := service.NewAuthnClientService(authnStore, sessionAuth)
	return authnClientSvc, authnStore, func() {
		authnStore.Close()

		// let background tasks finish
		time.Sleep(time.Millisecond * 100)
	}
}

// TestMain creates a test environment
// Used for all test cases in this package
func TestMain(m *testing.M) {

	logging.SetLogging("info", "")

	res := m.Run()

	time.Sleep(time.Second)
	if res == 0 {
		_ = os.RemoveAll(testDir)
	}
	os.Exit(res)
}

// Start the authn service and list clients
func TestStartStopClientService(t *testing.T) {
	// this creates the admin user key
	svc, store, stopFn := startTestAuthnClientService()
	_ = svc
	_ = store
	defer stopFn()
}

func TestClientUpdatePubKey(t *testing.T) {
	var tu1ID = "tu1ID"

	svc, store, stopFn := startTestAuthnClientService()
	defer stopFn()

	// add user to test with. don't set the public key yet
	err := store.Add(tu1ID, api.ClientProfile{
		ClientID:    tu1ID,
		ClientType:  api.ClientTypeUser,
		DisplayName: "user1",
		PubKey:      "",
	})
	require.NoError(t, err)

	// update the public key
	kp := keys.NewKey(keys.KeyTypeECDSA)
	prof1, err := svc.GetProfile(tu1ID)
	assert.Equal(t, tu1ID, prof1.ClientID)
	require.NoError(t, err)
	err = svc.UpdatePubKey(tu1ID, kp.ExportPublic())
	assert.NoError(t, err)

	// check result
	prof, err := svc.GetProfile(tu1ID)
	require.NoError(t, err)
	assert.Equal(t, kp.ExportPublic(), prof.PubKey)
}

// Note: RefreshToken is only possible when using JWT.
func TestLoginRefresh(t *testing.T) {
	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"
	var tu1SessionID = "session1"
	var authToken1 string
	var authToken2 string

	svc, store, stopFn := startTestAuthnClientService()
	defer stopFn()

	// add user to test with
	tu1Key := keys.NewKey(keys.KeyTypeECDSA)
	tu1KeyPub := tu1Key.ExportPublic()

	err := store.Add(tu1ID, api.ClientProfile{
		ClientID:    tu1ID,
		ClientType:  api.ClientTypeUser,
		DisplayName: "testuser1",
		PubKey:      tu1KeyPub,
	})
	require.NoError(t, err)
	err = store.SetPassword(tu1ID, tu1Pass)
	require.NoError(t, err)

	authToken1, err = svc.Login(tu1ID, tu1Pass, tu1SessionID)
	require.NoError(t, err)

	cid2, sid2, err := svc.ValidateToken(authToken1)
	assert.Equal(t, tu1ID, cid2)
	assert.Equal(t, tu1SessionID, sid2)
	require.NoError(t, err)

	// RefreshToken the token
	authToken2, err = svc.RefreshToken(tu1ID, authToken1)
	require.NoError(t, err)
	require.NotEmpty(t, authToken2)

	// ValidateToken the new token
	cid3, sid3, err := svc.ValidateToken(authToken2)
	assert.Equal(t, tu1ID, cid3)
	assert.Equal(t, sid2, sid3)
	require.NoError(t, err)
}

func TestLoginRefreshFail(t *testing.T) {

	svc, store, stopFn := startTestAuthnClientService()
	_ = store
	defer stopFn()

	// RefreshToken the token non-existing
	_, err := svc.RefreshToken("badclientID", "badToken")
	require.Error(t, err)
}

func TestUpdatePassword(t *testing.T) {

	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	svc, store, stopFn := startTestAuthnClientService()
	defer stopFn()

	// add user to test with
	err := store.Add(tu1ID, api.ClientProfile{
		ClientID:    tu1ID,
		DisplayName: tu1Name,
		ClientType:  api.ClientTypeUser})
	require.NoError(t, err)
	err = store.SetPassword(tu1ID, "oldpass")
	require.NoError(t, err)

	// login should succeed
	_, err = svc.Login(tu1ID, "oldpass", "session1")
	require.NoError(t, err)

	// change password
	err = svc.UpdatePassword(tu1ID, "newpass")
	require.NoError(t, err)

	// login with old password should now fail
	//t.Log("an error is expected logging in with the old password")
	_, err = svc.Login(tu1ID, "oldpass", "")
	require.Error(t, err)

	// re-login with new password
	_, err = svc.Login(tu1ID, "newpass", "")
	require.NoError(t, err)
}

func TestUpdatePasswordFail(t *testing.T) {
	svc, store, stopFn := startTestAuthnClientService()
	_ = store
	defer stopFn()

	err := svc.UpdatePassword("badclientid", "newpass")
	assert.Error(t, err)
}
