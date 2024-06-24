package authn_test

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/lib/hubclient/embedded"
	"github.com/hiveot/hub/lib/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// NOTE: this uses default settings from Authn_test.go

// Create manage users
func TestAddRemoveClientsSuccess(t *testing.T) {
	deviceID := "device1"
	deviceKP := keys.NewKey(keys.KeyTypeECDSA)
	deviceKeyPub := deviceKP.ExportPublic()
	serviceID := "service1"
	serviceKP := keys.NewKey(keys.KeyTypeECDSA)
	serviceKeyPub := serviceKP.ExportPublic()

	svc, adminHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	hc := embedded.NewEmbeddedClient(serviceID, adminHandler)

	err := authn.AdminAddConsumer(hc, "user1", "user 1", "pass1")
	assert.NoError(t, err)
	// duplicate should update
	err = authn.AdminAddConsumer(hc, "user1", "user 1 updated", "pass1")
	assert.NoError(t, err)

	err = authn.AdminAddConsumer(hc, "user2", "user 2", "pass2")
	assert.NoError(t, err)
	err = authn.AdminAddConsumer(hc, "user3", "user 3", "pass2")
	assert.NoError(t, err)
	err = authn.AdminAddConsumer(hc, "user4", "user 4", "pass2")
	assert.NoError(t, err)

	_, err = authn.AdminAddAgent(hc, deviceID, "agent 1", deviceKeyPub)
	assert.NoError(t, err)

	_, err = authn.AdminAddService(hc, serviceID, "service 1", serviceKeyPub)
	assert.NoError(t, err)

	// update the server. users can connect and have unlimited access
	profiles, err := authn.AdminGetProfiles(hc)
	require.NoError(t, err)
	assert.Equal(t, 6+3, len(profiles))

	err = authn.AdminRemoveClient(hc, "user1")
	assert.NoError(t, err)
	err = authn.AdminRemoveClient(hc, "user1") // remove is idempotent
	assert.NoError(t, err)
	err = authn.AdminRemoveClient(hc, "user2")
	assert.NoError(t, err)
	err = authn.AdminRemoveClient(hc, deviceID)
	assert.NoError(t, err)
	err = authn.AdminRemoveClient(hc, serviceID)
	assert.NoError(t, err)

	profiles, err = authn.AdminGetProfiles(hc)
	require.NoError(t, err)
	assert.Equal(t, 2+3, len(profiles))

	clEntries := svc.AdminSvc.GetEntries()
	assert.Equal(t, 2+3, len(clEntries))

	err = authn.AdminAddConsumer(hc, "user1", "user 1", "pass1")
	assert.NoError(t, err)
}

// Create manage users
func TestAddRemoveClientsFail(t *testing.T) {
	const adminID = "administrator-1"
	_, adminHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	hc := embedded.NewEmbeddedClient(adminID, adminHandler)

	// missing clientID should fail
	_, err := authn.AdminAddService(hc, "", "user 1", "")
	assert.Error(t, err)

	// a bad key is not an error
	err = authn.AdminAddConsumer(hc, "user2", "user 2", "badkey")
	assert.NoError(t, err)
}

func TestUpdateClientPassword(t *testing.T) {
	var tu1ID = "tu1ID"
	var tuPass1 = "tuPass1"
	var tuPass2 = "tuPass2"
	const adminID = "administrator-1"

	svc, adminHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	hc := embedded.NewEmbeddedClient(adminID, adminHandler)

	err := authn.AdminAddConsumer(hc, tu1ID, "user 1", tuPass1)
	require.NoError(t, err)

	err = svc.SessionAuth.ValidatePassword(tu1ID, tuPass1)
	require.NoError(t, err)

	err = authn.AdminSetClientPassword(hc, tu1ID, tuPass2)
	require.NoError(t, err)

	err = svc.SessionAuth.ValidatePassword(tu1ID, tuPass1)
	require.Error(t, err)

	err = svc.SessionAuth.ValidatePassword(tu1ID, tuPass2)
	require.NoError(t, err)
}

func TestUpdatePubKey(t *testing.T) {
	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"
	const adminID = "administrator-1"

	svc, adminHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	hc := embedded.NewEmbeddedClient(adminID, adminHandler)

	// add user to test with. don't set the public key yet
	err := authn.AdminAddConsumer(hc, tu1ID, "user 2", tu1Pass)
	require.NoError(t, err)
	//
	token := svc.SessionAuth.CreateSessionToken(tu1ID, "", 0)
	require.NotEmpty(t, token)

	// update the public key
	kp := keys.NewKey(keys.KeyTypeECDSA)
	profile, err := authn.AdminGetClientProfile(hc, tu1ID)
	require.NoError(t, err)
	profile.PubKey = kp.ExportPublic()
	err = authn.AdminUpdateClientProfile(hc, profile)
	assert.NoError(t, err)

	// check result
	profile2, err := authn.AdminGetClientProfile(hc, tu1ID)
	require.NoError(t, err)
	assert.Equal(t, kp.ExportPublic(), profile2.PubKey)
}

func TestNewAuthToken(t *testing.T) {
	var tu1ID = "ag1ID"
	var tu1Name = "agent 1"

	const adminID = "administrator-1"
	svc, adminHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	hc := embedded.NewEmbeddedClient(adminID, adminHandler)

	// add agent to test with and connect
	_, err := authn.AdminAddAgent(hc, tu1ID, tu1Name, "")
	require.NoError(t, err)

	// get a new token
	token, err := authn.AdminNewAuthToken(hc, tu1ID)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// login with new token
	clientID, _, err := svc.SessionAuth.ValidateToken(token)
	require.NoError(t, err)
	require.Equal(t, tu1ID, clientID)

}

func TestUpdateProfile(t *testing.T) {
	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	const adminID = "administrator-1"
	_, adminHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	hc := embedded.NewEmbeddedClient(adminID, adminHandler)

	// add user to test with and connect
	err := authn.AdminAddConsumer(hc, tu1ID, tu1Name, "pass0")
	require.NoError(t, err)
	//tu1Key, _ := testServer.MsgServer.CreateKP()

	// update display name
	const newDisplayName = "new display name"
	profile, err := authn.AdminGetClientProfile(hc, tu1ID)
	require.NoError(t, err)
	profile.DisplayName = newDisplayName
	err = authn.AdminUpdateClientProfile(hc, profile)
	assert.NoError(t, err)

	// verify
	profile2, err := authn.AdminGetClientProfile(hc, tu1ID)

	require.NoError(t, err)
	assert.Equal(t, newDisplayName, profile2.DisplayName)
}

func TestUpdateProfileFail(t *testing.T) {
	const adminID = "administrator-1"

	_, adminHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	hc := embedded.NewEmbeddedClient(adminID, adminHandler)

	err := authn.AdminUpdateClientProfile(hc, authn.ClientProfile{ClientID: "badclient"})
	assert.Error(t, err)
}

func TestBadAdminCommand(t *testing.T) {
	const adminID = "administrator-1"

	_, adminHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	cl := embedded.NewEmbeddedClient(adminID, adminHandler)
	err := cl.Rpc(authn.AdminServiceID, "badmethod", nil, nil)
	require.Error(t, err)

}
