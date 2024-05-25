package authz_test

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient/embedded"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn/authnstore"
	"github.com/hiveot/hub/runtime/authz"
	"github.com/hiveot/hub/runtime/authz/authzagent"
	"github.com/hiveot/hub/runtime/authz/authzclient"
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
	svc.Stop()
}

// Test Get/Set role
func TestSetRole(t *testing.T) {
	const client1ID = "client1"
	const client1Role = api.ClientRoleAgent
	// start the authz server
	cfg := authz.NewAuthzConfig()
	authnStore := authnstore.NewAuthnFileStore(passwordFile, "")
	svc := authz.NewAuthzService(&cfg, authnStore)
	err := svc.Start()
	require.NoError(t, err)
	defer svc.Stop()

	// add the user whose role to set
	err = authnStore.Add(client1ID, api.ClientProfile{
		ClientID:    client1ID,
		ClientType:  api.ClientTypeUser,
		DisplayName: "user 1",
	})
	assert.NoError(t, err)

	// connect the client marshaller to the server agent

	handler, _ := authzagent.StartAuthzAgent(svc, nil)
	ecl := embedded.NewEmbeddedClient(client1ID, handler.HandleMessage)
	authzCl := authzclient.NewAuthzClient(ecl)

	// set the role
	err = authzCl.SetClientRole(client1ID, client1Role)
	require.NoError(t, err)

	// get the role
	role, err := authzCl.GetClientRole(client1ID)
	require.NoError(t, err)
	require.Equal(t, client1Role, role)
}

func TestHasPermission(t *testing.T) {
	const client1ID = "client1"
	const client1Role = api.ClientRoleAgent
	cfg := authz.NewAuthzConfig()
	cfg.Setup(testDir)
	authnStore := authnstore.NewAuthnFileStore(passwordFile, "")
	svc := authz.NewAuthzService(&cfg, authnStore)
	err := svc.Start()
	require.NoError(t, err)
	defer svc.Stop()

	err = authnStore.Add(client1ID, api.ClientProfile{ClientID: client1ID, ClientType: api.ClientTypeUser})
	require.NoError(t, err)
	err = svc.SetClientRole(client1ID, client1Role)
	assert.NoError(t, err)
	hasperm := svc.HasPermission(client1ID, vocab.MessageTypeEvent, true)
	assert.True(t, hasperm)
	//
	hasperm = svc.CanPubAction(client1ID)
	assert.False(t, hasperm)
	hasperm = svc.CanPubEvent(client1ID)
	assert.True(t, hasperm)
	hasperm = svc.CanSubEvent(client1ID)
	assert.False(t, hasperm)
	hasperm = svc.CanSubAction(client1ID)
	assert.True(t, hasperm)
}
