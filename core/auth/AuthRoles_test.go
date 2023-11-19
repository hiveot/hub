package auth_test

import (
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/auth/authclient"
	"github.com/hiveot/hub/lib/hubclient"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test creating and deleting custom roles
func TestCRUDRole(t *testing.T) {
	const user1ID = "newUser"
	const user1Pass = "user1pass"
	const role1Name = "role1"
	const adminUserID = "admin"
	t.Log("--- TestGetRole start")
	defer t.Log("--- TestGetRole end")

	svc, mng, stopFn, err := startTestAuthnService()
	_ = svc
	require.NoError(t, err)
	defer stopFn()
	serverURL, _, _ := testServer.MsgServer.GetServerURLs()
	time.Sleep(time.Millisecond * 10)

	// create the user whose roles to test
	user1KP := testServer.MsgServer.CreateKeyPair()
	user1Pub := user1KP.ExportPublic()
	_, err = mng.AddUser(user1ID, "nu 1", user1Pass, user1Pub, authapi.ClientRoleViewer)
	require.NoError(t, err)

	// admin user that can change roles
	hc := hubclient.NewHubClient(serverURL, adminUserID, testServer.CertBundle.CaCert, testServer.Core)
	hc.SetRetryConnect(false)
	adminKP := hc.CreateKeyPair()
	adminPub := adminKP.ExportPublic()
	token, err := mng.AddUser(adminUserID, "admin", "", adminPub, authapi.ClientRoleAdmin)
	err = hc.ConnectWithToken(adminKP, token)
	require.NoError(t, err)
	defer hc.Disconnect()

	roleMng := authclient.NewRolesClient(hc)

	err = roleMng.CreateRole(role1Name)
	require.NoError(t, err)

	err = roleMng.DeleteRole(role1Name)
	require.NoError(t, err)
}
