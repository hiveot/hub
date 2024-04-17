package authn_test

import (
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn/authnclient"
	"github.com/hiveot/hub/runtime/protocols/direct"
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

	svc, adminHandler, _, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	mt := direct.NewDirectTransport(serviceID, adminHandler)
	adminCl := authnclient.NewAuthnAdminClient(mt)

	err := adminCl.AddClient(api.ClientTypeUser, "user1", "user 1", "pass1", "")
	assert.NoError(t, err)
	// duplicate should update
	err = adminCl.AddClient(api.ClientTypeUser, "user1", "user 1 updated", "pass1", "")
	assert.NoError(t, err)

	err = adminCl.AddClient(api.ClientTypeUser, "user2", "user 2", "", "pass2")
	assert.NoError(t, err)
	err = adminCl.AddClient(api.ClientTypeUser, "user3", "user 3", "", "pass2")
	assert.NoError(t, err)
	err = adminCl.AddClient(api.ClientTypeUser, "user4", "user 4", "", "pass2")
	assert.NoError(t, err)

	err = adminCl.AddClient(api.ClientTypeAgent, deviceID, "agent 1", deviceKeyPub, "")
	assert.NoError(t, err)

	err = adminCl.AddClient(api.ClientTypeService, serviceID, "service 1", serviceKeyPub, "")
	assert.NoError(t, err)

	// update the server. users can connect and have unlimited access
	clList, err := adminCl.GetProfiles()
	require.NoError(t, err)
	assert.Equal(t, 6+2, len(clList))

	err = adminCl.RemoveClient("user1")
	assert.NoError(t, err)
	err = adminCl.RemoveClient("user1") // remove is idempotent
	assert.NoError(t, err)
	err = adminCl.RemoveClient("user2")
	assert.NoError(t, err)
	err = adminCl.RemoveClient(deviceID)
	assert.NoError(t, err)
	err = adminCl.RemoveClient(serviceID)
	assert.NoError(t, err)

	clList, err = adminCl.GetProfiles()
	require.NoError(t, err)
	assert.Equal(t, 2+2, len(clList))

	clEntries := svc.AdminSvc.GetEntries()
	assert.Equal(t, 2+2, len(clEntries))

	err = adminCl.AddClient(api.ClientTypeUser, "user1", "user 1", "", "pass1")
	assert.NoError(t, err)
	// a bad key is allowed
	err = adminCl.AddClient(api.ClientTypeUser, "user2", "user 2", "badkey", "")
	assert.NoError(t, err)
}

// Create manage users
func TestAddRemoveClientsFail(t *testing.T) {
	const adminID = "administrator-1"
	_, adminHandler, _, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	mt := direct.NewDirectTransport(adminID, adminHandler)
	adminCl := authnclient.NewAuthnAdminClient(mt)

	// missing clientID should fail
	err := adminCl.AddClient(api.ClientTypeService, "", "user 1", "", "")
	assert.Error(t, err)

	// a bad key is not an error
	err = adminCl.AddClient(api.ClientTypeUser, "user2", "user 2", "badkey", "")
	assert.NoError(t, err)
}

func TestUpdateClientPassword(t *testing.T) {
	var tu1ID = "tu1ID"
	var tuPass1 = "tuPass1"
	var tuPass2 = "tuPass2"
	const adminID = "administrator-1"

	svc, adminHandler, _, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	mt := direct.NewDirectTransport(adminID, adminHandler)
	adminCl := authnclient.NewAuthnAdminClient(mt)

	err := adminCl.AddClient(
		api.ClientTypeUser, tu1ID, "user 1", "", tuPass1)
	require.NoError(t, err)

	token, err := svc.SessionAuth.Login(tu1ID, tuPass1, "session1")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	adminCl.UpdateClientPassword(tu1ID, tuPass2)

	token, err = svc.SessionAuth.Login(tu1ID, tuPass1, "session1")
	require.Error(t, err)
	require.Empty(t, token)

	token, err = svc.SessionAuth.Login(tu1ID, tuPass2, "session1")
	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestUpdatePubKey(t *testing.T) {
	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"
	const adminID = "administrator-1"

	svc, adminHandler, _, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	mt := direct.NewDirectTransport(adminID, adminHandler)
	adminCl := authnclient.NewAuthnAdminClient(mt)

	// add user to test with. don't set the public key yet
	err := adminCl.AddClient(api.ClientTypeUser, tu1ID, "user 2", "", tu1Pass)
	require.NoError(t, err)
	//
	token := svc.SessionAuth.CreateSessionToken(tu1ID, "", 0)
	require.NotEmpty(t, token)

	// update the public key
	kp := keys.NewKey(keys.KeyTypeECDSA)
	prof1, err := adminCl.GetClientProfile(tu1ID)
	require.NoError(t, err)
	prof1.PubKey = kp.ExportPublic()
	err = adminCl.UpdateClientProfile(prof1)
	assert.NoError(t, err)

	// check result
	prof, err := adminCl.GetClientProfile(tu1ID)
	require.NoError(t, err)
	assert.Equal(t, kp.ExportPublic(), prof.PubKey)
}

func TestUpdateProfile(t *testing.T) {
	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	const adminID = "administrator-1"
	_, adminHandler, _, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	mt := direct.NewDirectTransport(adminID, adminHandler)
	adminCl := authnclient.NewAuthnAdminClient(mt)

	// add user to test with and connect
	err := adminCl.AddClient(api.ClientTypeUser, tu1ID, tu1Name, "", "pass0")
	require.NoError(t, err)
	//tu1Key, _ := testServer.MsgServer.CreateKP()

	// update display name
	const newDisplayName = "new display name"
	prof1, err := adminCl.GetClientProfile(tu1ID)
	require.NoError(t, err)
	prof1.DisplayName = newDisplayName
	err = adminCl.UpdateClientProfile(prof1)
	assert.NoError(t, err)

	// verify
	prof2, err := adminCl.GetClientProfile(tu1ID)
	require.NoError(t, err)
	assert.Equal(t, newDisplayName, prof2.DisplayName)
}

func TestUpdateProfileFail(t *testing.T) {
	const adminID = "administrator-1"

	_, adminHandler, _, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	mt := direct.NewDirectTransport(adminID, adminHandler)
	adminCl := authnclient.NewAuthnAdminClient(mt)

	err := adminCl.UpdateClientProfile(api.ClientProfile{ClientID: "badclient"})
	assert.Error(t, err)
}

func TestBadAdminCommand(t *testing.T) {
	const adminID = "administrator-1"

	_, adminHandler, _, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	mt := direct.NewDirectTransport(adminID, adminHandler)
	err := mt(api.AuthnAdminThingID, "badmethod", nil, nil)
	require.Error(t, err)

}
