package authn_test

import (
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
)

// NOTE: this uses default settings from AuthnClient_test.go

// launch the authn service and return a client for using and managing it.
// the messaging server is already running (see TestMain)
func startTestAuthnAdminService(testHash string) (
	authnAdminSvc *service.AuthnAdminService, sessionAuth api.IAuthenticator, stopFn func(), err error) {

	// the password file to use
	passwordFile := path.Join(testDir, "test.passwd")

	// TODO: put this in a test environment
	_ = os.Remove(passwordFile)
	authnConfig = authn.NewAuthnConfig()
	authnConfig.Setup(testDir, testDir)
	authnConfig.PasswordFile = passwordFile
	authnConfig.AgentTokenValiditySec = 100
	authnConfig.Encryption = testHash

	authnStore := authnstore.NewAuthnFileStore(
		authnConfig.PasswordFile, authnConfig.Encryption)
	err = authnStore.Open()
	if err != nil {
		panic("Error opening password store:" + err.Error())
	}
	sessionAuth = authenticator.NewJWTAuthenticatorFromFile(
		authnStore, authnConfig.KeysDir, authnConfig.DefaultKeyType)
	authnAdminSvc = service.NewAuthnAdminService(&authnConfig, authnStore, sessionAuth)
	err = authnAdminSvc.Start()
	if err != nil {
		panic("Error starting authn admin service:" + err.Error())
	}
	_ = sessionAuth
	return authnAdminSvc, sessionAuth, func() {
		authnAdminSvc.Stop()
		authnStore.Close()

		// let background tasks finish
		time.Sleep(time.Millisecond * 100)
	}, err
}

// Start the authn service and list clients
func TestStartStop(t *testing.T) {
	// this creates the admin user key
	svc, _, stopFn, err := startTestAuthnAdminService(defaultHash)
	require.NoError(t, err)
	defer stopFn()

	clList, err := svc.GetProfiles()
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

	svc, _, stopFn, err := startTestAuthnAdminService(defaultHash)
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
	clList, err := svc.GetProfiles()
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

	clList, err = svc.GetProfiles()
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
	svc, _, stopFn, err := startTestAuthnAdminService(defaultHash)
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

	svc, sessionAuth, stopFn, err := startTestAuthnAdminService(defaultHash)
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
	prof1, err := svc.GetClientProfile(tu1ID)
	require.NoError(t, err)
	prof1.PubKey = kp.ExportPublic()
	err = svc.UpdateClientProfile(prof1)
	assert.NoError(t, err)

	// check result
	prof, err := svc.GetClientProfile(tu1ID)
	require.NoError(t, err)
	assert.Equal(t, kp.ExportPublic(), prof.PubKey)
}

func TestUpdateProfile(t *testing.T) {
	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	svc, sessionAuth, stopFn, err := startTestAuthnAdminService(defaultHash)
	_ = sessionAuth
	defer stopFn()
	require.NoError(t, err)

	// add user to test with and connect
	err = svc.AddClient(api.ClientTypeUser, tu1ID, tu1Name, "", "pass0")
	require.NoError(t, err)
	//tu1Key, _ := testServer.MsgServer.CreateKP()

	// update display name
	const newDisplayName = "new display name"
	prof1, err := svc.GetClientProfile(tu1ID)
	require.NoError(t, err)
	prof1.DisplayName = newDisplayName
	err = svc.UpdateClientProfile(prof1)
	assert.NoError(t, err)

	// verify
	prof2, err := svc.GetClientProfile(tu1ID)
	require.NoError(t, err)
	assert.Equal(t, newDisplayName, prof2.DisplayName)
}

func TestUpdateProfileFail(t *testing.T) {
	svc, _, stopFn, err := startTestAuthnAdminService(defaultHash)
	defer stopFn()
	require.NoError(t, err)
	err = svc.UpdateClientProfile(api.ClientProfile{ClientID: "badclient"})
	assert.Error(t, err)
}
