package runtime_test

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime"
	"github.com/hiveot/hub/runtime/api"
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
var rtConfig = runtime.NewRuntimeConfig()
var certsBundle = certs.CreateTestCertBundle()

// start the runtime
func startRuntime() *runtime.Runtime {
	r := runtime.NewRuntime(rtConfig)
	err := r.Start(&rtEnv)
	if err != nil {
		panic("Failed to start runtime:" + err.Error())
	}
	return r
}

// Add a client and connect with its password
func addConnectClient(r *runtime.Runtime, clientType api.ClientType, clientID string) (cl *tlsclient.TLSClient, token string) {
	password := "pass1"
	err := r.AuthnSvc.AdminSvc.AddClient(clientType, clientID, clientID, "", password)
	if err != nil {
		panic("Failed adding client:" + err.Error())
	}

	addr := fmt.Sprintf("localhost:%d", TestPort)
	cl = tlsclient.NewTLSClient(addr, certsBundle.CaCert, time.Second*120)
	token, err = cl.ConnectWithPassword(clientID, password)
	if err != nil {
		panic("Failed connect with password:" + err.Error())
	}
	return cl, token
}

// create a TD with the given thingID
func createTD(thingID string) *things.TD {
	td := things.NewTD(thingID, "title-"+thingID, vocab.ThingSensor)
	return td
}

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
	r := startRuntime()
	r.Stop()
	time.Sleep(time.Millisecond * 100)
}

func TestLogin(t *testing.T) {
	const senderID = "sender1"
	const password = "pass1"

	r := startRuntime()
	cl, _ := addConnectClient(r, api.ClientTypeUser, senderID)
	cl.Close()
	r.Stop()
	time.Sleep(time.Millisecond * 100)
}

func TestAddRemoveTD(t *testing.T) {
	t.Log("--- TestAddRemoveTD start ---")
	defer t.Log("--- TestAddRemoveTD end ---")

	const agentID = "agent1"
	const userID = "user1"
	const thing1ID = "thing1"

	r := startRuntime()
	defer r.Stop()
	ag, _ := addConnectClient(r, api.ClientTypeAgent, agentID)
	defer ag.Close()
	cl, _ := addConnectClient(r, api.ClientTypeUser, userID)
	defer cl.Close()

	td1 := createTD(thing1ID)
	params := map[string]string{"thingID": thing1ID}
	addr := utils.Substitute(vocab.PostThingPath, params)
	_, err := ag.Post(addr, td1)
	assert.NoError(t, err)

	// Get returns a serialized TD object
	addr = utils.Substitute(vocab.GetThingPath, params)
	td2Doc, err := cl.Get(addr)
	//time.Sleep(time.Second * 30)  // otherwise timeout during debugging. Is there a better way?
	require.NoError(t, err)
	td2 := things.TD{}
	err = json.Unmarshal(td2Doc, &td2)
	require.NoError(t, err)

	addr = fmt.Sprintf("/things/%s", thing1ID)
	_, err = cl.Delete(addr)
	require.NoError(t, err)

	// after removal, getTD should return nil
	addr = fmt.Sprintf("/things/%s", thing1ID)
	td3, err := cl.Get(addr)
	assert.Empty(t, td3)
	assert.Error(t, err)
}

func TestHttpsPubValueEvent(t *testing.T) {
	const thingID = "thing1"
	const key1 = "key1"
	const senderID = "sender1"
	const data = "Hello world"

	r := startRuntime()
	cl, _ := addConnectClient(r, api.ClientTypeUser, senderID)

	// publish an event
	msg := things.NewThingMessage(vocab.MessageTypeEvent, thingID, key1, []byte(data), senderID)

	// TODO: use a client from the library. path needs to match the server
	//evPath := fmt.Sprintf("/event/%s/%s", thingID, key1)
	vars := map[string]string{"thingID": thingID, "key": key1}
	eventPath := utils.Substitute(vocab.PostEventPath, vars)

	_, err := cl.Post(eventPath, msg.Data)
	assert.NoError(t, err)

	props, err := r.DigiTwinSvc.Values.ReadEvents(thingID, nil, "")
	require.NoError(t, err)
	require.NotEmpty(t, props)
	assert.Equal(t, msg.Data, props[key1].Data)

	// the event must be in the store
	r.Stop()
}

func TestHttpsPutProperties(t *testing.T) {
	const thingID = "thing1"
	const agentID = "agent1"

	r := startRuntime()
	cl, token := addConnectClient(r, api.ClientTypeAgent, agentID)
	require.NotEmpty(t, token)
	vars := map[string]string{"thingID": thingID, "key": vocab.EventTypeProperties}
	pubEventPath := utils.Substitute(vocab.PostEventPath, vars)
	props := map[string]string{
		"prop1": "val1",
		"prop2": "val2",
	}
	data, _ := json.Marshal(props)
	_, err := cl.Post(pubEventPath, data)
	assert.NoError(t, err)

	readPropsPath := utils.Substitute(vocab.GetPropertiesPath, vars)
	data, err = cl.Get(readPropsPath)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
}
