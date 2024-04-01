package authn_test

import (
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/keys"
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
var authConfig *authn.AuthnConfig

// launch the authn service and return a client for using and managing it.
// the messaging server is already running (see TestMain)
func startTestAuthnService() (authnSvc *service.AuthnService, stopFn func(), err error) {
	// the password file to use
	passwordFile := path.Join(testDir, "test.passwd")

	// TODO: put this in a test environment
	_ = os.Remove(passwordFile)
	authConfig = authn.NewAuthnConfig()
	_ = authConfig.Setup(testDir, testDir)
	authConfig.PasswordFile = passwordFile
	authConfig.AgentTokenValiditySec = 100
	authConfig.Encryption = authn.PWHASH_BCRYPT // nats requires bcrypt

	authnSvc, err = service.StartAuthnService(authConfig, certBundle.CaCert)
	if err != nil {
		panic("cant start test authn service: " + err.Error())
	}
	return authnSvc, func() {
		authnSvc.Stop()

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
	t.Log("--- TestStartStop start")
	defer t.Log("--- TestStartStop end")

	// this creates the admin user key
	svc, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	defer stopFn()

	clList, err := svc.GetAllClients()
	require.NoError(t, err)
	// admin and launcher are 2 pre-existing clients
	assert.Equal(t, 2, len(clList))
}

// Create manage users
func TestAddRemoveClientsSuccess(t *testing.T) {
	t.Log("--- TestAddRemoveClientsSuccess start")
	defer t.Log("--- TestAddRemoveClientsSuccess stop")

	deviceID := "device1"
	deviceKP := keys.NewKey(keys.KeyTypeECDSA)
	deviceKeyPub := deviceKP.ExportPublic()
	serviceID := "service1"
	serviceKP := keys.NewKey(keys.KeyTypeECDSA)
	serviceKeyPub := serviceKP.ExportPublic()

	svc, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	defer stopFn()

	err = svc.AddClient(authn.ClientTypeUser, "user1", "user 1", "pass1", "")
	assert.NoError(t, err)
	// duplicate should update
	err = svc.AddClient(authn.ClientTypeUser, "user1", "user 1 updated", "pass1", "")
	assert.NoError(t, err)

	err = svc.AddClient(authn.ClientTypeUser, "user2", "user 2", "", "pass2")
	assert.NoError(t, err)
	err = svc.AddClient(authn.ClientTypeUser, "user3", "user 3", "", "pass2")
	assert.NoError(t, err)
	err = svc.AddClient(authn.ClientTypeUser, "user4", "user 4", "", "pass2")
	assert.NoError(t, err)

	err = svc.AddClient(authn.ClientTypeAgent, deviceID, "agent 1", deviceKeyPub, "")
	assert.NoError(t, err)

	err = svc.AddClient(authn.ClientTypeService, serviceID, "service 1", serviceKeyPub, "")
	assert.NoError(t, err)

	// update the server. users can connect and have unlimited access
	clList, err := svc.GetAllClients()
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

	clList, err = svc.GetAllClients()
	require.NoError(t, err)
	assert.Equal(t, 2+2, len(clList))

	err = svc.AddClient(authn.ClientTypeUser, "user1", "user 1", "", "pass1")
	assert.NoError(t, err)
	// a bad key is allowed
	err = svc.AddClient(authn.ClientTypeUser, "user2", "user 2", "badkey", "")
	assert.NoError(t, err)
}

// Create manage users
func TestAddRemoveClientsFail(t *testing.T) {
	t.Log("--- TestAddRemoveClientsFail start")
	defer t.Log("--- TestAddRemoveClientsFail stop")

	svc, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	defer stopFn()

	// missing clientID should fail
	err = svc.AddClient(authn.ClientTypeService, "", "user 1", "", "")
	assert.Error(t, err)

	// a bad key is not an error
	err = svc.AddClient(authn.ClientTypeUser, "user2", "user 2", "badkey", "")
	assert.NoError(t, err)

}

func TestUpdatePubKey(t *testing.T) {
	t.Log("--- TestUpdatePubKey start")
	defer t.Log("--- TestUpdatePubKey end")

	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"

	svc, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add user to test with. don't set the public key yet
	err = svc.AddClient(authn.ClientTypeUser, tu1ID, "user 2", "", tu1Pass)
	require.NoError(t, err)
	//
	token, err := svc.CreateSessionToken(tu1ID, "", 0)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// update the public key
	kp := keys.NewKey(keys.KeyTypeECDSA)
	prof1, err := svc.GetClient(tu1ID)
	require.NoError(t, err)
	prof1.PubKey = kp.ExportPublic()
	err = svc.UpdateClient(tu1ID, prof1)
	assert.NoError(t, err)

	// check result
	prof, err := svc.GetClient(tu1ID)
	require.NoError(t, err)
	assert.Equal(t, kp.ExportPublic(), prof.PubKey)
}

// Note: Refresh is only useful when using JWT. The nats nkey server ignores the token and uses nkeys
func TestLoginRefresh(t *testing.T) {
	t.Log("--- TestLoginRefresh start")
	defer t.Log("--- TestLoginRefresh end")

	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"
	var authToken1 string
	var authToken2 string

	svc, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add user to test with
	tu1Key := keys.NewKey(keys.KeyTypeECDSA)
	tu1KeyPub := tu1Key.ExportPublic()

	err = svc.AddClient(authn.ClientTypeUser,
		tu1ID, "testuser 1", tu1KeyPub, tu1Pass)
	require.NoError(t, err)
	authToken1, err = svc.Login(tu1ID, tu1Pass, "")
	require.NoError(t, err)

	err = svc.ValidateToken(tu1ID, authToken1)
	require.NoError(t, err)

	// Refresh the token
	authToken2, err = svc.RefreshToken(tu1ID, authToken1)
	require.NoError(t, err)
	require.NotEmpty(t, authToken2)

	// Validate the new token
	err = svc.ValidateToken(tu1ID, authToken2)
	require.NoError(t, err)
}

func TestUpdateProfile(t *testing.T) {
	t.Log("--- TestUpdateProfile start")
	defer t.Log("--- TestUpdateProfile end")
	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	svc, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add user to test with and connect
	err = svc.AddClient(authn.ClientTypeUser, tu1ID, tu1Name, "", "pass0")
	require.NoError(t, err)
	//tu1Key, _ := testServer.MsgServer.CreateKP()

	// update display name
	const newDisplayName = "new display name"
	prof1, err := svc.GetClient(tu1ID)
	require.NoError(t, err)
	prof1.DisplayName = newDisplayName
	err = svc.UpdateClient(tu1ID, prof1)
	assert.NoError(t, err)

	// verify
	prof2, err := svc.GetClient(tu1ID)
	require.NoError(t, err)
	assert.Equal(t, newDisplayName, prof2.DisplayName)
}

func TestUpdatePassword(t *testing.T) {
	t.Log("--- TestUpdate start")
	defer t.Log("--- TestUpdate end")
	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	svc, stopFn, err := startTestAuthnService()
	defer stopFn()
	require.NoError(t, err)

	// add user to test with
	err = svc.AddClient(authn.ClientTypeUser, tu1ID, tu1Name, "", "oldpass")
	require.NoError(t, err)

	// login should succeed
	_, err = svc.Login(tu1ID, "oldpass", "session1")
	require.NoError(t, err)

	// change password
	err = svc.UpdatePassword(tu1ID, "newpass")
	require.NoError(t, err)

	// login with old password should now fail
	t.Log("an error is expected logging in with the old password")
	_, err = svc.Login(tu1ID, "oldpass", "")
	require.Error(t, err)

	// re-login with new password
	_, err = svc.Login(tu1ID, "newpass", "")
	require.NoError(t, err)

}
