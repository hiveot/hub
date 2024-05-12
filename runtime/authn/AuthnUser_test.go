package authn_test

import (
	"github.com/hiveot/hub/lib/hubclient/embedded"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn/authnclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestClientUpdatePubKey(t *testing.T) {
	var tu1ID = "tu1ID"

	svc, userHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	// wrap service in message de/encoders
	ecl := embedded.NewEmbeddedClient(tu1ID, userHandler)
	userCl := authnclient.NewAuthnUserClient(ecl)

	// add user to test with. don't set the public key yet
	err := svc.AdminSvc.AddClient(
		api.ClientTypeUser, tu1ID, tu1ID, "user1", "")
	require.NoError(t, err)

	// update the public key
	kp := keys.NewKey(keys.KeyTypeECDSA)
	prof1, err := svc.UserSvc.GetProfile(tu1ID)
	assert.Equal(t, tu1ID, prof1.ClientID)
	require.NoError(t, err)
	err = userCl.UpdatePubKey(tu1ID, kp.ExportPublic())
	assert.NoError(t, err)

	// check result
	prof, err := userCl.GetProfile()
	require.NoError(t, err)
	assert.Equal(t, tu1ID, prof.ClientID)
	assert.Equal(t, kp.ExportPublic(), prof.PubKey)
}

// Note: RefreshToken is only possible when using JWT.
func TestLoginRefresh(t *testing.T) {
	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"
	var authToken1 string
	var authToken2 string

	svc, userHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	// create the client that connects directly to the user service
	ecl := embedded.NewEmbeddedClient(tu1ID, userHandler)
	userCl := authnclient.NewAuthnUserClient(ecl)

	// add user to test with
	tu1Key := keys.NewKey(keys.KeyTypeECDSA)
	tu1KeyPub := tu1Key.ExportPublic()
	err := svc.AdminSvc.AddClient(
		api.ClientTypeUser, tu1ID, "testuser1", tu1KeyPub, "")
	require.NoError(t, err)

	err = userCl.UpdatePassword(tu1Pass)
	require.NoError(t, err)

	// FIXME: how to provide a sessionID??
	authToken1, err = userCl.Login(tu1ID, tu1Pass)
	require.NoError(t, err)

	cid2, sid2, err := svc.SessionAuth.ValidateToken(authToken1)
	assert.Equal(t, tu1ID, cid2)
	assert.NotEmpty(t, sid2)
	require.NoError(t, err)

	// RefreshToken the token
	authToken2, err = userCl.RefreshToken(authToken1)
	require.NoError(t, err)
	require.NotEmpty(t, authToken2)

	// ValidateToken the new token
	cid3, sid3, err := svc.SessionAuth.ValidateToken(authToken2)
	assert.Equal(t, tu1ID, cid3)
	assert.Equal(t, sid2, sid3)
	require.NoError(t, err)
}

func TestLoginRefreshFail(t *testing.T) {
	var tu1ID = "testuser1"

	_, userHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	// create the client that connects directly to the user service
	ecl := embedded.NewEmbeddedClient(tu1ID, userHandler)
	userCl := authnclient.NewAuthnUserClient(ecl)

	// RefreshToken the token non-existing
	token, err := userCl.RefreshToken("badToken")
	_ = token
	require.Error(t, err)
}

func TestUpdatePassword(t *testing.T) {

	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	svc, userHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	// create the client that connects directly to the user service
	ecl := embedded.NewEmbeddedClient(tu1ID, userHandler)
	userCl := authnclient.NewAuthnUserClient(ecl)

	// add user to test with
	err := svc.AdminSvc.AddClient(
		api.ClientTypeUser, tu1ID, tu1Name, "", "oldpass")
	require.NoError(t, err)

	// login should succeed
	_, err = userCl.Login(tu1ID, "oldpass")
	require.NoError(t, err)

	// change password
	err = userCl.UpdatePassword("newpass")
	require.NoError(t, err)

	// login with old password should now fail
	//t.Log("an error is expected logging in with the old password")
	_, err = userCl.Login(tu1ID, "oldpass")
	require.Error(t, err)

	// re-login with new password
	_, err = userCl.Login(tu1ID, "newpass")
	require.NoError(t, err)
}

func TestUpdatePasswordFail(t *testing.T) {
	var tu1ID = "tu1ID"
	_, userHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	// create the client that connects directly to the user service
	ecl := embedded.NewEmbeddedClient(tu1ID, userHandler)
	userCl := authnclient.NewAuthnUserClient(ecl)

	err := userCl.UpdatePassword("newpass")
	assert.Error(t, err)
}

func TestUpdateName(t *testing.T) {

	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"
	var tu2Name = "test user 1"

	svc, userHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	// create the client that connects directly to the user service
	ecl := embedded.NewEmbeddedClient(tu1ID, userHandler)
	userCl := authnclient.NewAuthnUserClient(ecl)

	// add user to test with
	err := svc.AdminSvc.AddClient(
		api.ClientTypeUser, tu1ID, tu1Name, "", "oldpass")
	require.NoError(t, err)

	prof1, err := userCl.GetProfile()
	require.NoError(t, err)
	assert.Equal(t, tu1Name, prof1.DisplayName)

	err = userCl.UpdateName(tu2Name)
	require.NoError(t, err)
	prof2, err := userCl.GetProfile()
	require.NoError(t, err)

	assert.Equal(t, tu2Name, prof2.DisplayName)
}

func TestBadUserCommand(t *testing.T) {
	_, userHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	ecl := embedded.NewEmbeddedClient("client1", userHandler)
	stat, err := ecl.Rpc(nil, api.AuthnUserThingID, "badmethod", nil, nil)
	_ = stat
	require.Error(t, err)

}
