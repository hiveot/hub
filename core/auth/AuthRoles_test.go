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

	_, mng, stopFn, err := startTestAuthnService()
	require.NoError(t, err)
	defer stopFn()
	time.Sleep(time.Millisecond * 10)

	_, err = mng.AddUser(testenv.TestUser1ID, "u 1", testenv.TestUser1Pass, "", authapi.ClientRoleViewer)
	require.NoError(t, err)

	hc, err := msgServer.ConnectInProc("test")
	roles := authclient.NewAuthRolesClient(hc)

	err = roles.CreateRole(role1Name)
	require.NoError(t, err)

	err = roles.SetRole(testenv.TestUser1ID, authapi.ClientRoleViewer)
	require.NoError(t, err)

	// user2 hasn't been added yet
	err = roles.SetRole(testenv.TestUser2ID, authapi.ClientRoleViewer)
	require.Error(t, err)

	err = roles.DeleteRole(role1Name)
	require.NoError(t, err)
}
