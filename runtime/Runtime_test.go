package runtime

import (
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/runtime/authn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
	"time"
)

const TestPort = 9444

var testDir = path.Join(os.TempDir(), "test-runtime")
var rtEnv = plugin.GetAppEnvironment(testDir, false)
var rtConfig = NewRuntimeConfig()
var certsBundle = certs.CreateTestCertBundle()

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	rtConfig.Protocols.HttpsBinding.Port = TestPort
	rtConfig.CaCert = certsBundle.CaCert
	rtConfig.CaKey = certsBundle.CaKey
	rtConfig.ServerKey = certsBundle.ServerKey
	rtConfig.ServerCert = certsBundle.ServerCert
	err := rtConfig.Setup(&rtEnv)
	if err != nil {
		panic("setup runtime config failed: " + err.Error())
	}

	_ = os.RemoveAll(testDir)

	res := m.Run()
	if res == 0 {
		_ = os.RemoveAll(testDir)
	}
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	r := NewRuntime(rtConfig)
	err := r.Start(&rtEnv)
	require.NoError(t, err)
	r.Stop()
	time.Sleep(time.Millisecond * 100)
}

func TestLogin(t *testing.T) {
	const senderID = "sender1"
	const password = "pass1"

	r := NewRuntime(rtConfig)
	err := r.Start(&rtEnv)
	require.NoError(t, err)
	err = r.authnSvc.AddClient(authn.ClientTypeUser, senderID, senderID, "", password)
	require.NoError(t, err)
	cl := tlsclient.NewTLSClient(fmt.Sprintf("localhost:%d", TestPort), certsBundle.CaCert)
	_, err = cl.ConnectWithPassword(senderID, password)
	assert.NoError(t, err)
	r.Stop()
	time.Sleep(time.Millisecond * 100)
}

func TestHttpsPubEvent(t *testing.T) {
	const thingID = "thing1"
	const key1 = "key1"
	const senderID = "sender1"
	const password = "pass1"
	const data = "Hello world"

	r := NewRuntime(rtConfig)
	err := r.Start(&rtEnv)
	require.NoError(t, err)
	err = r.authnSvc.AddClient(authn.ClientTypeUser, senderID, senderID, "", password)
	require.NoError(t, err)

	// publish an event
	msg := things.NewThingMessage(vocab.MessageTypeEvent, thingID, key1, []byte(data), senderID)
	cl := tlsclient.NewTLSClient(fmt.Sprintf("localhost:%d", TestPort), certsBundle.CaCert)
	assert.Equal(t, certsBundle.CaCert, rtConfig.CaCert)

	token, err := cl.ConnectWithPassword(senderID, password)
	_ = token
	//err = cl.ConnectWithClientCert(certsBundle.ClientCert)
	assert.NoError(t, err)
	// TODO: use a client from the library. path needs to match the server
	evPath := fmt.Sprintf("/event/%s/%s", thingID, key1)
	_, err = cl.Post(evPath, msg)
	assert.NoError(t, err)

	// the event must be in the store
	r.Stop()
}
