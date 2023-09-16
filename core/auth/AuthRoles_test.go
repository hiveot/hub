package auth_test

import (
	authapi "github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/core/auth/authclient"
	"github.com/hiveot/hub/lib/testenv"
	"golang.org/x/exp/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Start the authn service and list clients
func TestCRUDRole(t *testing.T) {
	const role1Name = "role1"
	slog.Info("--- TestGetRole start")
	defer slog.Info("--- TestGetRole end")

	svc, mng, stopFn, err := startTestAuthnService()
	_ = svc
	require.NoError(t, err)
	defer stopFn()
	time.Sleep(time.Millisecond * 10)

	_, err = mng.AddUser(testenv.TestUser1ID, "u 1", testenv.TestUser1Pass, "", authapi.ClientRoleViewer)
	require.NoError(t, err)

	hc, err := msgServer.ConnectInProc(testenv.TestUser1ID)
	roleMng := authclient.NewAuthRolesClient(hc)

	err = roleMng.CreateRole(role1Name)
	require.NoError(t, err)

	err = roleMng.SetRole(testenv.TestUser1ID, authapi.ClientRoleViewer)
	require.NoError(t, err)

	// admin user hasn't been added yet
	err = roleMng.SetRole(testenv.TestAdminUserID, authapi.ClientRoleAdmin)
	require.Error(t, err)

	err = roleMng.DeleteRole(role1Name)
	require.NoError(t, err)
}
