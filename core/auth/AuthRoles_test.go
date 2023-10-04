package auth_test

import (
	authapi "github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/core/auth/authclient"
	"github.com/hiveot/hub/lib/hubcl"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Start the authn service and list clients
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
	_, user1Pub := testServer.MsgServer.CreateKP()
	_, err = mng.AddUser(user1ID, "nu 1", user1Pass, user1Pub, authapi.ClientRoleViewer)
	require.NoError(t, err)

	// admin user that can change roles
	adminKP, adminPub := testServer.MsgServer.CreateKP()
	token, err := mng.AddUser(adminUserID, "admin", "", adminPub, authapi.ClientRoleViewer)
	hc := hubcl.NewHubClient(serverURL, adminUserID, adminKP, testServer.CertBundle.CaCert, testServer.Core)
	err = hc.ConnectWithToken(token)
	require.NoError(t, err)
	defer hc.Disconnect()

	roleMng := authclient.NewAuthRolesClient(hc)

	err = roleMng.CreateRole(role1Name)
	require.NoError(t, err)

	err = roleMng.SetRole(user1ID, authapi.ClientRoleViewer)
	require.NoError(t, err)

	// user hasn't been added yet
	err = roleMng.SetRole("notaknownuser", authapi.ClientRoleAdmin)
	require.Error(t, err)

	err = roleMng.DeleteRole(role1Name)
	require.NoError(t, err)
}
