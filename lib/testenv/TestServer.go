package testenv

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/httpclient"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime"
	"github.com/hiveot/hub/runtime/api"
	"log/slog"
	"math/rand"
	"os"
	"path"
)

// TestDir is the default test directory
var TestDir = path.Join(os.TempDir(), "hiveot-test")

var testTDs = []struct {
	ID         string
	Title      string
	DeviceType string
	NrEvents   int
	NrProps    int
	NrActions  int
}{
	{ID: "thing-1", Title: "Environmental Sensor",
		DeviceType: vocab.ThingSensorEnvironment, NrEvents: 1, NrProps: 1, NrActions: 0},
	{ID: "thing-2", Title: "Light Switch",
		DeviceType: vocab.ThingActuatorLight, NrEvents: 2, NrProps: 2, NrActions: 1},
	{ID: "thing-3", Title: "Power meter",
		DeviceType: vocab.ThingMeterElectric, NrEvents: 3, NrProps: 3, NrActions: 1},
	{ID: "thing-4", Title: "Multisensor",
		DeviceType: vocab.ThingSensorMulti, NrEvents: 4, NrProps: 4, NrActions: 2},
	{ID: "thing-5", Title: "Alarm",
		DeviceType: vocab.ThingActuatorAlarm, NrEvents: 2, NrProps: 2, NrActions: 2},
}

var PropTypes = []string{vocab.PropDeviceMake, vocab.PropDeviceModel,
	vocab.PropDeviceDescription, vocab.PropDeviceFirmwareVersion, vocab.PropLocationCity}
var EventTypes = []string{vocab.PropElectricCurrent, vocab.PropElectricVoltage,
	vocab.PropElectricPower, vocab.PropEnvTemperature, vocab.PropEnvPressure}
var ActionTypes = []string{vocab.ActionDimmer, vocab.ActionSwitch,
	vocab.ActionSwitchToggle, vocab.ActionValveOpen, vocab.ActionValveClose}

// TestServer for testing application services.
// Usage: run NewTestServer() followed by Start(clean)
type TestServer struct {
	Port    int
	Certs   certs.TestCertBundle
	TestDir string
	//
	Config  *runtime.RuntimeConfig
	AppEnv  plugin.AppEnvironment
	Runtime *runtime.Runtime
}

// AddConnectUser creates a new test user with the given role,
// and returns a hub client and a new session token.
// In case of error this panics.
func (test *TestServer) AddConnectUser(
	clientID string, clientRole string) (cl hubclient.IHubClient, token string) {

	password := clientID
	err := test.Runtime.AuthnSvc.AdminSvc.AddUser(clientID, clientID, password)
	if err == nil {
		err = test.Runtime.AuthzSvc.SetClientRole(clientID, clientRole)
	}
	if err != nil {
		panic("Failed adding client:" + err.Error())
	}

	hostPort := fmt.Sprintf("localhost:%d", test.Port)
	cl = httpclient.NewHttpSSEClient(hostPort, clientID, test.Certs.CaCert)
	token, err = cl.ConnectWithPassword(password)

	if err != nil {
		panic("Failed connect with password:" + err.Error())
	}
	return cl, token
}

// AddConnectAgent creates a new agent test client.
// Agents use non-session tokens and survive a server restart.
// This returns the agent's connection token.
//
// clientType can be one of ClientTypeAgent or ClientTypeService
func (test *TestServer) AddConnectAgent(
	clientType api.ClientType, clientID string) (
	cl hubclient.IHubClient, token string) {

	token, err := test.Runtime.AuthnSvc.AdminSvc.AddAgent(
		clientType, clientID, clientID, "")
	if err == nil {
		err = test.Runtime.AuthzSvc.SetClientRole(clientID, api.ClientRoleAgent)
	}
	if err != nil {
		panic("Failed adding client:" + err.Error())
	}

	hostPort := fmt.Sprintf("localhost:%d", test.Port)
	cl = httpclient.NewHttpSSEClient(hostPort, clientID, test.Certs.CaCert)
	_, err = cl.ConnectWithToken(token)
	if err != nil {
		panic("Failed connecting using token. ClientID=" + clientID)
	}

	return cl, token
}

// AddTD adds a test TD document using the embedded connection to the runtime
// if td is nil then a random TD will be added
func (test *TestServer) AddTD(agentID string, td *things.TD) *things.TD {
	if td == nil {
		i := rand.Intn(99882)
		td = test.CreateTestTD(i)
	}
	tdJSON, _ := json.Marshal(td)
	ag := test.Runtime.TransportsMgr.GetEmbedded().NewClient(agentID)
	err := ag.PubEvent(td.ID, vocab.EventTypeTD, tdJSON)
	if err != nil {
		slog.Error("Failed adding TD")
	}
	return td
}

// AddTDs adds a bunch of test TD documents
func (test *TestServer) AddTDs(agentID string, count int) {
	for i := 0; i < count; i++ {
		_ = test.AddTD(agentID, nil)
	}
}

// CreateTestTD returns a test TD.
//
// The first 10 are predefined and always the same. A higher number generates at random.
// i is the index.
func (test *TestServer) CreateTestTD(i int) (td *things.TD) {
	tdi := testTDs[0]
	if i < len(testTDs) {
		tdi = testTDs[i]
	} else {
		tdi.ID = fmt.Sprintf("thing-%d", rand.Intn(99823))
	}

	td = things.NewTD(tdi.ID, tdi.Title, tdi.DeviceType)
	// add random properties
	for n := 0; n < tdi.NrProps; n++ {
		td.AddProperty(fmt.Sprintf("prop-%d", n), PropTypes[n], "title-"+PropTypes[n], vocab.WoTDataTypeString)
	}
	// add random events
	for n := 0; n < tdi.NrEvents; n++ {
		td.AddProperty(fmt.Sprintf("prop-%d", n), EventTypes[n], "title-"+EventTypes[n], vocab.WoTDataTypeString)
	}
	// add random actions
	for n := 0; n < tdi.NrActions; n++ {
		td.AddProperty(fmt.Sprintf("prop-%d", n), ActionTypes[n], "title-"+ActionTypes[n], vocab.WoTDataTypeString)
	}

	return td
}

// Stop the test server
func (test *TestServer) Stop() {
	test.Runtime.Stop()
}

// Start the test server.
// This panics if something goes wrong.
func (test *TestServer) Start(clean bool) {
	//logging.SetLogging("info", "")

	if clean {
		_ = os.RemoveAll(test.TestDir)
	}
	test.AppEnv = plugin.GetAppEnvironment(test.TestDir, false)
	//
	test.Config.Transports.HttpsTransport.Port = test.Port
	test.Config.CaCert = test.Certs.CaCert
	test.Config.CaKey = test.Certs.CaKey
	test.Config.ServerKey = test.Certs.ServerKey
	test.Config.ServerCert = test.Certs.ServerCert
	err := test.Config.Setup(&test.AppEnv)
	if err != nil {
		panic("unable to setup test server config")
	}
	test.Runtime = runtime.NewRuntime(test.Config)
	err = test.Runtime.Start(&test.AppEnv)
	if err != nil {
		panic("unable to start test server runtime")
	}
}

func NewTestServer() *TestServer {
	srv := TestServer{
		Port:    9444,
		TestDir: TestDir,
		Certs:   certs.CreateTestCertBundle(),
		Config:  runtime.NewRuntimeConfig(),
	}

	return &srv
}

// StartTestServer creates and starts the test server
// This panics if start fails.
func StartTestServer(clean bool) *TestServer {
	srv := NewTestServer()
	srv.Start(clean)
	return srv
}
