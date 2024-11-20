package service_test

import (
	"github.com/hiveot/hub/api/go/authn"
	authz2 "github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/runtime/authn/authnstore"
	"github.com/hiveot/hub/runtime/authz"
	"github.com/hiveot/hub/runtime/authz/service"
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
	svc := service.NewAuthzService(&cfg, nil)
	err := svc.Start()
	require.NoError(t, err)
	svc.Stop()
}

// Test Get/Set role
func TestSetRole(t *testing.T) {
	const client1ID = "client1"
	const client1Role = authz2.ClientRoleAgent
	// start the authz server
	cfg := authz.NewAuthzConfig()
	authnStore := authnstore.NewAuthnFileStore(passwordFile, "")
	svc := service.NewAuthzService(&cfg, authnStore)
	err := svc.Start()
	require.NoError(t, err)
	defer svc.Stop()

	// add the user whose role to set
	err = authnStore.Add(client1ID, authn.ClientProfile{
		ClientID:    client1ID,
		ClientType:  authn.ClientTypeConsumer,
		DisplayName: "user 1",
	})
	assert.NoError(t, err)

	// connect the client marshaller to the server agent

	//handler, _ := service.StartAuthzAgent(svc, nil)
	//hc := embedded.NewEmbeddedClient(client1ID, handler.HandleMessage)

	// set the role
	err = svc.SetClientRole(client1ID, authz2.AdminSetClientRoleArgs{
		ClientID: client1ID,
		Role:     client1Role,
	})
	//err = authz2.AdminSetClientRole(hc, client1ID, client1Role)
	//require.NoError(t, err)

	// get the role
	role, err := svc.GetClientRole(client1ID, client1ID)
	//role, err := authz2.AdminGetClientRole(hc, client1ID)
	require.NoError(t, err)
	require.Equal(t, client1Role, role)
}

func TestHasPermission(t *testing.T) {
	const operatorID = "operator-1"
	const client1Role = authz2.ClientRoleOperator
	const thingID = ""
	const key = ""
	cfg := authz.NewAuthzConfig()
	cfg.Setup(testDir)
	authnStore := authnstore.NewAuthnFileStore(passwordFile, "")
	svc := service.NewAuthzService(&cfg, authnStore)
	err := svc.Start()
	require.NoError(t, err)
	defer svc.Stop()

	err = authnStore.Add(operatorID,
		authn.ClientProfile{ClientID: operatorID, ClientType: authn.ClientTypeConsumer})
	require.NoError(t, err)
	err = svc.SetClientRole(operatorID, authz2.AdminSetClientRoleArgs{operatorID, client1Role})
	assert.NoError(t, err)
	// consumers have permission to publish actions and write-property requests
	msg := hubclient.NewThingMessage(vocab.OpInvokeAction, thingID, key, "", operatorID)
	hasPerm := svc.HasPermission(msg.SenderID, msg.Operation, msg.ThingID)
	assert.True(t, hasPerm)

	// operators cannot publish property values
	msg = hubclient.NewThingMessage(vocab.HTOpUpdateProperty, thingID, key, "", operatorID)
	hasPerm = svc.HasPermission(msg.SenderID, msg.Operation, msg.ThingID)
	assert.False(t, hasPerm)
}
