package authn_test

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/lib/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestClientUpdatePubKey(t *testing.T) {
	var user1ID = "user1ID"

	svc, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()

	// add user to test with. don't set the public key yet
	err := svc.AdminSvc.AddConsumer("",
		authn.AdminAddConsumerArgs{user1ID, user1ID, "user1"})
	profile, err := svc.AdminSvc.GetClientProfile("", user1ID)
	require.NoError(t, err)
	assert.Equal(t, user1ID, profile.ClientID)
	assert.Equal(t, user1ID, profile.DisplayName)
	assert.NotEmpty(t, profile.Updated)

	// update the public key
	kp := keys.NewKey(keys.KeyTypeECDSA)
	profile2, err := svc.UserSvc.GetProfile(user1ID)
	assert.Equal(t, user1ID, profile2.ClientID)
	require.NoError(t, err)
	err = svc.UserSvc.UpdatePubKey(user1ID, kp.ExportPublic())
	assert.NoError(t, err)

	// check result
	profile3, err := svc.UserSvc.GetProfile(user1ID)
	require.NoError(t, err)
	assert.Equal(t, user1ID, profile3.ClientID)
	assert.Equal(t, kp.ExportPublic(), profile3.PubKey)
}

// Note: RefreshToken is only possible when using JWT.
func TestLoginRefresh(t *testing.T) {
	var user1ID = "user1ID"
	var tu1Pass = "tu1Pass"

	svc, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()

	// add user to test with
	err := svc.AdminSvc.AddConsumer(
		user1ID, authn.AdminAddConsumerArgs{user1ID, "testuser1", ""})
	require.NoError(t, err)

	err = svc.UserSvc.UpdatePassword(user1ID, tu1Pass)
	require.NoError(t, err)

	token1, err := svc.UserSvc.Login(user1ID, authn.UserLoginArgs{user1ID, tu1Pass})
	require.NoError(t, err)

	cid2, sid2, err := svc.SessionAuth.ValidateToken(token1)
	assert.Equal(t, user1ID, cid2)
	assert.NotEmpty(t, sid2)
	require.NoError(t, err)

	// RefreshToken the token
	token2, err := svc.UserSvc.RefreshToken(user1ID, token1)
	require.NoError(t, err)
	require.NotEmpty(t, token2)

	// ValidateToken the new token
	cid3, sid3, err := svc.SessionAuth.ValidateToken(token2)
	assert.Equal(t, user1ID, cid3)
	assert.Equal(t, sid2, sid3)
	require.NoError(t, err)
}

func TestLoginRefreshFail(t *testing.T) {
	var user1ID = "testuser1"

	svc, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()

	// RefreshToken the token non-existing
	resp, err := svc.UserSvc.RefreshToken(user1ID, "badToken")
	_ = resp
	require.Error(t, err)
}

func TestUpdatePassword(t *testing.T) {

	var user1ID = "user1ID"
	var tu1Name = "test user 1"

	svc, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()

	// add user to test with
	err := svc.AdminSvc.AddConsumer(user1ID, authn.AdminAddConsumerArgs{user1ID, tu1Name, "oldpass"})
	require.NoError(t, err)

	// login should succeed
	_, err = svc.UserSvc.Login(user1ID, authn.UserLoginArgs{user1ID, "oldpass"})
	require.NoError(t, err)

	// change password
	err = svc.UserSvc.UpdatePassword(user1ID, "newpass")
	require.NoError(t, err)

	// login with old password should now fail
	//t.Log("an error is expected logging in with the old password")
	_, err = svc.UserSvc.Login(user1ID, authn.UserLoginArgs{user1ID, "oldpass"})
	require.Error(t, err)

	// re-login with new password
	_, err = svc.UserSvc.Login(user1ID, authn.UserLoginArgs{user1ID, "newpass"})
	require.NoError(t, err)
}

func TestUpdatePasswordFail(t *testing.T) {
	var user1ID = "user1ID"
	svc, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()

	err := svc.UserSvc.UpdatePassword(user1ID, "newpass")
	assert.Error(t, err)
}

func TestUpdateName(t *testing.T) {

	var user1ID = "user1ID"
	var tu1Name = "test user 1"
	var tu2Name = "test user 1"

	svc, stopFn := startTestAuthnService(defaultHash)
	defer stopFn()

	// add user to test with
	err := svc.AdminSvc.AddConsumer(user1ID, authn.AdminAddConsumerArgs{user1ID, tu1Name, "oldpass"})
	require.NoError(t, err)

	profile, err := svc.UserSvc.GetProfile(user1ID)
	require.NoError(t, err)
	assert.Equal(t, tu1Name, profile.DisplayName)

	err = svc.UserSvc.UpdateName(user1ID, tu2Name)
	require.NoError(t, err)
	profile2, err := svc.UserSvc.GetProfile(user1ID)
	require.NoError(t, err)

	assert.Equal(t, tu2Name, profile2.DisplayName)
}
