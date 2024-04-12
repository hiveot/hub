package authn_test

import (
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn"
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
var defaultHash = api.PWHASH_ARGON2id

// launch the authn service and return a client for using and managing it.
// the messaging server is already running (see TestMain)
func startTestAuthnService(testHash string) (
	authnSvc *service.AuthnService, sessionAuth api.IAuthenticator, stopFn func(), err error) {

	// the password file to use
	passwordFile := path.Join(testDir, "test.passwd")

	// TODO: put this in a test environment
	_ = os.Remove(passwordFile)
	authnConfig = authn.NewAuthnConfig()
	authnConfig.Setup(testDir, testDir)
	authnConfig.PasswordFile = passwordFile
	authnConfig.AgentTokenValiditySec = 100
	authnConfig.Encryption = testHash

	authnSvc, authnStore, sessionAuth, err := service.StartAuthnService(&authnConfig, certBundle.CaCert)
	_ = sessionAuth
	return authnSvc, sessionAuth, func() {
		authnSvc.Stop()
		authnStore.Close()

		// let background tasks finish
		time.Sleep(time.Millisecond * 100)
	}, err
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
func TestStartStop(t *testing.T) {
	// this creates the admin user key
	svc, _, stopFn, err := startTestAuthnService(defaultHash)
	require.NoError(t, err)
	defer stopFn()

	clList, err := svc.GetAllProfiles()
	require.NoError(t, err)
	// admin and launcher are 2 pre-existing clients
	assert.Equal(t, 2, len(clList))
}

// Create manage users
func TestAddRemoveClientsSuccess(t *testing.T) {
	deviceID := "device1"
	deviceKP := keys.NewKey(keys.KeyTypeECDSA)
	deviceKeyPub := deviceKP.ExportPublic()
	serviceID := "service1"
	serviceKP := keys.NewKey(keys.KeyTypeECDSA)
	serviceKeyPub := serviceKP.ExportPublic()

	svc, _, stopFn, err := startTestAuthnService(defaultHash)
	require.NoError(t, err)
	defer stopFn()

	err = svc.AddClient(api.ClientTypeUser, "user1", "user 1", "pass1", "")
	assert.NoError(t, err)
	// duplicate should update
	err = svc.AddClient(api.ClientTypeUser, "user1", "user 1 updated", "pass1", "")
	assert.NoError(t, err)

	err = svc.AddClient(api.ClientTypeUser, "user2", "user 2", "", "pass2")
	assert.NoError(t, err)
	err = svc.AddClient(api.ClientTypeUser, "user3", "user 3", "", "pass2")
	assert.NoError(t, err)
	err = svc.AddClient(api.ClientTypeUser, "user4", "user 4", "", "pass2")
	assert.NoError(t, err)

	err = svc.AddClient(api.ClientTypeAgent, deviceID, "agent 1", deviceKeyPub, "")
	assert.NoError(t, err)

	err = svc.AddClient(api.ClientTypeService, serviceID, "service 1", serviceKeyPub, "")
	assert.NoError(t, err)

	// update the server. users can connect and have unlimited access
	clList, err := svc.GetAllProfiles()
	require.NoError(t, err)
	assert.Equal(t, 6+2, len(clList))

	err = svc.RemoveClient("user1")
	assert.NoError(t, err)
	err = svc.RemoveClient("user1") // remove is idempotent
	assert.NoError(t, err)
	err = svc.RemoveClient("user2")
	assert.NoError(t, err)
	err = svc.RemoveClient(deviceID)
	assert.NoError(t, err)
	err = svc.RemoveClient(serviceID)
	assert.NoError(t, err)

	clList, err = svc.GetAllProfiles()
	require.NoError(t, err)
	assert.Equal(t, 2+2, len(clList))

	clEntries := svc.GetEntries()
	assert.Equal(t, 2+2, len(clEntries))

	err = svc.AddClient(api.ClientTypeUser, "user1", "user 1", "", "pass1")
	assert.NoError(t, err)
	// a bad key is allowed
	err = svc.AddClient(api.ClientTypeUser, "user2", "user 2", "badkey", "")
	assert.NoError(t, err)
}

// Create manage users
func TestAddRemoveClientsFail(t *testing.T) {
	svc, _, stopFn, err := startTestAuthnService(defaultHash)
	require.NoError(t, err)
	defer stopFn()

	// missing clientID should fail
	err = svc.AddClient(api.ClientTypeService, "", "user 1", "", "")
	assert.Error(t, err)

	// a bad key is not an error
	err = svc.AddClient(api.ClientTypeUser, "user2", "user 2", "badkey", "")
	assert.NoError(t, err)

	//
}

func TestUpdatePubKey(t *testing.T) {
	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"

	svc, sessionAuth, stopFn, err := startTestAuthnService(defaultHash)
	defer stopFn()
	require.NoError(t, err)

	// add user to test with. don't set the public key yet
	err = svc.AddClient(api.ClientTypeUser, tu1ID, "user 2", "", tu1Pass)
	require.NoError(t, err)
	//
	token, err := sessionAuth.CreateSessionToken(tu1ID, "", 0)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// update the public key
	kp := keys.NewKey(keys.KeyTypeECDSA)
	prof1, err := svc.GetProfile(tu1ID)
	require.NoError(t, err)
	prof1.PubKey = kp.ExportPublic()
	err = svc.UpdateClient(tu1ID, prof1)
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
	var authToken1 string
	var authToken2 string

	svc, sessionAuth, stopFn, err := startTestAuthnService(defaultHash)
	defer stopFn()
	require.NoError(t, err)

	// add user to test with
	tu1Key := keys.NewKey(keys.KeyTypeECDSA)
	tu1KeyPub := tu1Key.ExportPublic()

	err = svc.AddClient(api.ClientTypeUser,
		tu1ID, "testuser 1", tu1KeyPub, tu1Pass)
	require.NoError(t, err)
	authToken1, err = svc.Login(tu1ID, tu1Pass, "")
	require.NoError(t, err)

	cid2, sid2, err := sessionAuth.ValidateToken(authToken1)
	assert.Equal(t, tu1ID, cid2)
	assert.NotEmpty(t, sid2)
	require.NoError(t, err)

	// RefreshToken the token
	authToken2, err = sessionAuth.RefreshToken(tu1ID, authToken1, 100)
	require.NoError(t, err)
	require.NotEmpty(t, authToken2)

	// ValidateToken the new token
	cid3, sid3, err := sessionAuth.ValidateToken(authToken2)
	assert.Equal(t, tu1ID, cid3)
	assert.Equal(t, sid2, sid3)
	require.NoError(t, err)
}

func TestLoginRefreshFail(t *testing.T) {

	_, sessionAuth, stopFn, err := startTestAuthnService(defaultHash)
	defer stopFn()
	require.NoError(t, err)

	// RefreshToken the token non-existing
	_, err = sessionAuth.RefreshToken("badclientID", "badToken", 10)
	require.Error(t, err)
}

func TestUpdateProfile(t *testing.T) {
	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	svc, sessionAuth, stopFn, err := startTestAuthnService(defaultHash)
	_ = sessionAuth
	defer stopFn()
	require.NoError(t, err)

	// add user to test with and connect
	err = svc.AddClient(api.ClientTypeUser, tu1ID, tu1Name, "", "pass0")
	require.NoError(t, err)
	//tu1Key, _ := testServer.MsgServer.CreateKP()

	// update display name
	const newDisplayName = "new display name"
	prof1, err := svc.GetProfile(tu1ID)
	require.NoError(t, err)
	prof1.DisplayName = newDisplayName
	err = svc.UpdateClient(tu1ID, prof1)
	assert.NoError(t, err)

	// verify
	prof2, err := svc.GetProfile(tu1ID)
	require.NoError(t, err)
	assert.Equal(t, newDisplayName, prof2.DisplayName)
}

func TestUpdateProfileFail(t *testing.T) {
	svc, _, stopFn, err := startTestAuthnService(defaultHash)
	defer stopFn()
	require.NoError(t, err)
	err = svc.UpdateClient("badclient", api.ClientProfile{})
	assert.Error(t, err)
}

func TestUpdatePasswordForHashes(t *testing.T) {
	const badhash = "badhhash"
	var hashes = []string{
		api.PWHASH_BCRYPT, api.PWHASH_ARGON2id, badhash}
	for _, testHash := range hashes {
		var testName = "test-" + testHash
		t.Run(testName, func(t *testing.T) {
			var tu1ID = "tu1ID"
			var tu1Name = "test user 1"

			svc, _, stopFn, err := startTestAuthnService(testHash)
			defer stopFn()
			if testHash == badhash {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			// add user to test with
			err = svc.AddClient(api.ClientTypeUser, tu1ID, tu1Name, "", "oldpass")
			if testHash == badhash {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// login should succeed
			_, err = svc.Login(tu1ID, "oldpass", "session1")
			if testHash == badhash {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// change password
			err = svc.UpdatePassword(tu1ID, "newpass")
			if testHash == badhash {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// login with old password should now fail
			//t.Log("an error is expected logging in with the old password")
			_, err = svc.Login(tu1ID, "oldpass", "")
			require.Error(t, err)

			// re-login with new password
			_, err = svc.Login(tu1ID, "newpass", "")
			if testHash == badhash {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpdatePasswordFail(t *testing.T) {
	svc, _, stopFn, err := startTestAuthnService(defaultHash)
	defer stopFn()
	require.NoError(t, err)

	err = svc.UpdatePassword("badclientid", "newpass")
	assert.Error(t, err)
}
