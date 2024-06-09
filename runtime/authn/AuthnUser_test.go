package authn_test

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/lib/hubclient/embedded"
	"github.com/hiveot/hub/lib/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestClientUpdatePubKey(t *testing.T) {
	var tu1ID = "tu1ID"

	svc, userHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	ehc := embedded.NewEmbeddedClient(tu1ID, userHandler)
	// PROBLEM: this client connects directly to the agent with the digitwin thingID.
	// The agent however expects the native thingID.
	// Option1: agent removes the digitwin prefix if used.
	// Option2: client accepts agentID to use
	//   this allows connecting to different agents instead of hardcoding one.
	//   not an issue for auth though.
	userCl := authn.NewUserClient(ehc)

	// add user to test with. don't set the public key yet
	err := svc.AdminSvc.AddConsumer("",
		authn.AdminAddConsumerArgs{tu1ID, tu1ID, "user1"})
	profile, err := svc.AdminSvc.GetClientProfile("", tu1ID)
	require.NoError(t, err)
	assert.Equal(t, tu1ID, profile.ClientID)
	assert.Equal(t, tu1ID, profile.DisplayName)
	assert.NotEmpty(t, profile.Updated)

	// update the public key
	kp := keys.NewKey(keys.KeyTypeECDSA)
	profile2, err := svc.UserSvc.GetProfile(tu1ID)
	assert.Equal(t, tu1ID, profile2.ClientID)
	require.NoError(t, err)
	err = userCl.UpdatePubKey(kp.ExportPublic())
	assert.NoError(t, err)

	// check result
	profile3, err := userCl.GetProfile()
	require.NoError(t, err)
	assert.Equal(t, tu1ID, profile3.ClientID)
	assert.Equal(t, kp.ExportPublic(), profile3.PubKey)
}

// Note: RefreshToken is only possible when using JWT.
func TestLoginRefresh(t *testing.T) {
	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"

	svc, userHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	// create the client that connects directly to the user service
	ecl := embedded.NewEmbeddedClient(tu1ID, userHandler)
	userCl := authn.NewUserClient(ecl)

	// add user to test with
	err := svc.AdminSvc.AddConsumer(tu1ID, authn.AdminAddConsumerArgs{tu1ID, "testuser1", ""})
	require.NoError(t, err)

	err = userCl.UpdatePassword(tu1Pass)
	require.NoError(t, err)

	// FIXME: how to provide a sessionID??
	resp, err := userCl.Login(authn.UserLoginArgs{tu1ID, tu1Pass})
	require.NoError(t, err)

	cid2, sid2, err := svc.SessionAuth.ValidateToken(resp.Token)
	assert.Equal(t, tu1ID, cid2)
	assert.NotEmpty(t, sid2)
	require.NoError(t, err)

	// RefreshToken the token
	token2, err := userCl.RefreshToken(authn.UserRefreshTokenArgs{tu1ID, resp.Token})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Token)

	// ValidateToken the new token
	cid3, sid3, err := svc.SessionAuth.ValidateToken(token2)
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
	userCl := authn.NewUserClient(ecl)

	// RefreshToken the token non-existing
	resp, err := userCl.RefreshToken(authn.UserRefreshTokenArgs{tu1ID, "badToken"})
	_ = resp
	require.Error(t, err)
}

func TestUpdatePassword(t *testing.T) {

	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	svc, userHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	// create the client that connects directly to the user service
	ecl := embedded.NewEmbeddedClient(tu1ID, userHandler)
	userCl := authn.NewUserClient(ecl)

	// add user to test with
	err := svc.AdminSvc.AddConsumer(tu1ID, authn.AdminAddConsumerArgs{tu1ID, tu1Name, "oldpass"})
	require.NoError(t, err)

	// login should succeed
	_, err = userCl.Login(authn.UserLoginArgs{tu1ID, "oldpass"})
	require.NoError(t, err)

	// change password
	err = userCl.UpdatePassword("newpass")
	require.NoError(t, err)

	// login with old password should now fail
	//t.Log("an error is expected logging in with the old password")
	_, err = userCl.Login(authn.UserLoginArgs{tu1ID, "oldpass"})
	require.Error(t, err)

	// re-login with new password
	_, err = userCl.Login(authn.UserLoginArgs{tu1ID, "newpass"})
	require.NoError(t, err)
}

func TestUpdatePasswordFail(t *testing.T) {
	var tu1ID = "tu1ID"
	_, userHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	// create the client that connects directly to the user service
	ecl := embedded.NewEmbeddedClient(tu1ID, userHandler)
	userCl := authn.NewUserClient(ecl)

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
	userCl := authn.NewUserClient(ecl)

	// add user to test with
	err := svc.AdminSvc.AddConsumer(tu1ID, authn.AdminAddConsumerArgs{tu1ID, tu1Name, "oldpass"})
	require.NoError(t, err)

	profile, err := userCl.GetProfile()
	require.NoError(t, err)
	assert.Equal(t, tu1Name, profile.DisplayName)

	err = userCl.UpdateName(tu2Name)
	require.NoError(t, err)
	profile2, err := userCl.GetProfile()
	require.NoError(t, err)

	assert.Equal(t, tu2Name, profile2.DisplayName)
}

func TestBadUserCommand(t *testing.T) {
	_, userHandler, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()
	ecl := embedded.NewEmbeddedClient("client1", userHandler)
	err := ecl.Rpc(authn.UserServiceID, "badmethod", nil, nil)
	require.Error(t, err)

}
