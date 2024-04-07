package authz_test

import (
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/runtime/authn"
	"github.com/hiveot/hub/runtime/authn/authnstore"
	"github.com/hiveot/hub/runtime/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
)

var testDir = path.Join(os.TempDir(), "test-authz")
var passwordFile = path.Join(testDir, "test.passwd")

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")

	_ = os.RemoveAll(testDir)

	res := m.Run()
	if res == 0 {
		_ = os.RemoveAll(testDir)
	}
	os.Exit(res)
}

// Test starting and stopping authorization service
func TestStartStop(t *testing.T) {
	cfg := authz.NewAuthzConfig()
	svc := authz.NewAuthzService(&cfg, nil)
	err := svc.Start()
	require.NoError(t, err)
	defer svc.Stop()
}

// Test Get/Set role
func TestSetRole(t *testing.T) {
	const client1ID = "client1"
	const client1Role = authz.ClientRoleAgent
	cfg := authz.NewAuthzConfig()
	authnStore := authnstore.NewAuthnFileStore(passwordFile, "")
	svc := authz.NewAuthzService(&cfg, authnStore)
	err := svc.Start()
	require.NoError(t, err)
	defer svc.Stop()

	svc.SetRole(client1ID, client1Role)
}

func TestHasPermission(t *testing.T) {
	const client1ID = "client1"
	const client1Role = authz.ClientRoleAgent
	cfg := authz.NewAuthzConfig()
	cfg.Setup(testDir)
	authnStore := authnstore.NewAuthnFileStore(passwordFile, "")
	svc := authz.NewAuthzService(&cfg, authnStore)
	err := svc.Start()
	require.NoError(t, err)
	defer svc.Stop()

	err = authnStore.Add(client1ID, authn.ClientProfile{ClientID: client1ID, ClientType: authn.ClientTypeUser})
	require.NoError(t, err)
	err = svc.SetRole(client1ID, client1Role)
	assert.NoError(t, err)
	hasperm := svc.HasPermission(client1ID, vocab.MessageTypeEvent, true)
	assert.True(t, hasperm)
	//
	hasperm = svc.CanPubAction(client1ID)
	assert.False(t, hasperm)
	hasperm = svc.CanPubEvent(client1ID)
	assert.True(t, hasperm)
	hasperm = svc.CanPubRPC(client1ID, "someservice", "someinterface")
	assert.False(t, hasperm)
	hasperm = svc.CanSubEvent(client1ID)
	assert.False(t, hasperm)
	hasperm = svc.CanSubAction(client1ID)
	assert.True(t, hasperm)
	hasperm = svc.CanSubRPC(client1ID)
	assert.False(t, hasperm)
}
