package authz_test

import (
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/runtime/authz"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
)

const authzFileName = "testauthzroles.json"

var authzFilePath string

var tempFolder string

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	tempFolder = path.Join(os.TempDir(), "hiveot-authz-test")
	_ = os.MkdirAll(tempFolder, 0700)

	authzFilePath = path.Join(tempFolder, authzFileName)
	_ = os.Remove(authzFilePath)

	res := m.Run()
	if res == 0 {
		_ = os.RemoveAll(tempFolder)
	}
	os.Exit(res)
}

// Test creating and deleting custom roles
func TestCRUDRole(t *testing.T) {
	const user1ID = "newUser"
	const user1Pass = "user1pass"
	const role1Name = "role1"
	const adminUserID = "admin"
	t.Log("--- TestGetRole start")
	defer t.Log("--- TestGetRole end")

	cfg := authz.NewAuthzConfig()
	svc := authz.NewAuthzService(cfg)
	err := svc.Start()
	require.NoError(t, err)
	defer svc.Stop()

	err = svc.CreateRole(role1Name)
	require.NoError(t, err)

	err = svc.DeleteRole(role1Name)
	require.NoError(t, err)
}

// Test creating and deleting custom roles
func TestVerifyPermissions(t *testing.T) {
	const user1ID = "newUser"
	const user1Pass = "user1pass"
	const role1Name = "role1"
	const adminUserID = "admin"
	t.Log("--- TestGetRole start")
	defer t.Log("--- TestGetRole end")

	cfg := authz.NewAuthzConfig()
	svc := authz.NewAuthzService(cfg)
	err := svc.Start()
	require.NoError(t, err)
	defer svc.Stop()

	// verify default permissions
	hasperm = svc.VerifyPermissions(role1Name)
	require.True(t, hasperm)

}
