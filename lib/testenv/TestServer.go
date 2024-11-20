package testenv

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/sseclient"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/runtime"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"math/rand"
	"os"
	"path"
	"time"
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
		DeviceType: vocab.ThingSensorEnvironment, NrEvents: 1, NrProps: 1, NrActions: 3},
	{ID: "thing-2", Title: "Light Switch",
		DeviceType: vocab.ThingActuatorLight, NrEvents: 2, NrProps: 2, NrActions: 0},
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
	clientID string, clientRole authz.ClientRole) (cl hubclient.IConsumerClient, token string) {

	password := clientID
	err := test.Runtime.AuthnSvc.AdminSvc.AddConsumer(clientID,
		authn.AdminAddConsumerArgs{clientID, "my name", password})
	if err == nil {
		err = test.Runtime.AuthzSvc.SetClientRole(clientID,
			authz.AdminSetClientRoleArgs{clientID, clientRole})
	}
	if err != nil {
		panic("Failed adding client:" + err.Error())
	}

	hostPort := fmt.Sprintf("localhost:%d", test.Port)
	cl = sseclient.NewHttpSSEClient(hostPort, clientID, nil, test.Certs.CaCert, time.Minute)
	token, err = cl.ConnectWithPassword(password)

	if err != nil {
		panic("Failed connect with password:" + err.Error())
	}
	return cl, token
}

// AddConnectAgent creates a new agent test client.
// Agents use non-session tokens and survive a server restart.
// This returns the agent's connection token.
func (test *TestServer) AddConnectAgent(
	agentID string) (cl hubclient.IAgentClient, token string) {

	token, err := test.Runtime.AuthnSvc.AdminSvc.AddAgent(agentID,
		authn.AdminAddAgentArgs{agentID, "agent name", ""})

	if err == nil {
		err = test.Runtime.AuthzSvc.SetClientRole(agentID,
			authz.AdminSetClientRoleArgs{agentID, authz.ClientRoleAgent})
	}
	if err != nil {
		panic("AddConnectAgent: Failed adding client:" + err.Error())
	}

	hostPort := fmt.Sprintf("localhost:%d", test.Port)
	cl = sseclient.NewHttpSSEClient(hostPort, agentID, nil, test.Certs.CaCert, time.Minute)
	_, err = cl.ConnectWithToken(token)
	if err != nil {
		panic("AddConnectAgent: Failed connecting using token. SenderID=" + agentID)
	}

	return cl, token
}

// AddConnectService creates a new service test client.
// Services are agents and use non-session tokens and survive a server restart.
// This returns the service's connection token.
//
// clientType can be one of ClientTypeAgent or ClientTypeService
func (test *TestServer) AddConnectService(serviceID string) (
	cl hubclient.IAgentClient, token string) {

	token, err := test.Runtime.AuthnSvc.AdminSvc.AddService(serviceID,
		authn.AdminAddServiceArgs{serviceID, "service name", ""})
	if err == nil {
		err = test.Runtime.AuthzSvc.SetClientRole(serviceID,
			authz.AdminSetClientRoleArgs{serviceID, authz.ClientRoleService})
	}
	if err != nil {
		panic("AddConnectService: Failed adding client:" + err.Error())
	}

	hostPort := fmt.Sprintf("localhost:%d", test.Port)
	cl = sseclient.NewHttpSSEClient(hostPort, serviceID, nil, test.Certs.CaCert, time.Minute)
	_, err = cl.ConnectWithToken(token)
	if err != nil {
		panic("AddConnectService: Failed connecting using token. serviceID=" + serviceID)
	}

	return cl, token
}

// AddTD adds a test TD document to the runtime
// if td is nil then a random TD will be added
func (test *TestServer) AddTD(agentID string, td *tdd.TD) *tdd.TD {
	if td == nil {
		i := rand.Intn(99882)
		td = test.CreateTestTD(i)
	}
	tdJSON, _ := json.Marshal(td)
	err := test.Runtime.DigitwinSvc.DirSvc.UpdateTD(agentID, string(tdJSON))
	//ag := test.Runtime.TransportsMgr.GetEmbedded().NewClient(agentID)
	//err := ag.PubEvent(td.ID, vocab.EventNameTD, string(tdJSON))
	if err != nil {
		slog.Error("AddTD: Failed adding TD")
	}
	return td
}

// AddTDs adds a bunch of test TD documents
func (test *TestServer) AddTDs(agentID string, count int) {
	for i := 0; i < count; i++ {
		_ = test.AddTD(agentID, nil)
	}
}

// CreateTestTD returns a test TD with ID "thing-{i}", and a variable
// number of properties, events and actions.
//
//	properties are named "prop-{i}
//	events are named "event-{i}
//	actions are named "action-{i}
//
// The first 10 are predefined and always the same. A higher number generates at random.
// i is the index.
func (test *TestServer) CreateTestTD(i int) (td *tdd.TD) {
	tdi := testTDs[0]
	if i < len(testTDs) {
		tdi = testTDs[i]
	} else {
		tdi.ID = fmt.Sprintf("thing-%d", rand.Intn(99823))
	}

	td = tdd.NewTD(tdi.ID, tdi.Title, tdi.DeviceType)
	// add random properties
	for n := 0; n < tdi.NrProps; n++ {
		propName := fmt.Sprintf("prop-%d", n)
		td.AddProperty(propName, "title-"+PropTypes[n], "", vocab.WoTDataTypeString).
			SetAtType(PropTypes[n])
	}
	// add random events
	for n := 0; n < tdi.NrEvents; n++ {
		evName := fmt.Sprintf("event-%d", n)
		td.AddEvent(evName, "title-"+EventTypes[n], "",
			&tdd.DataSchema{Type: vocab.WoTDataTypeString}).
			SetAtType(EventTypes[n])
	}
	// add random actions
	for n := 0; n < tdi.NrActions; n++ {
		actionName := fmt.Sprintf("action-%d", n)
		td.AddAction(actionName, "title-"+ActionTypes[n], "",
			&tdd.DataSchema{Type: vocab.WoTDataTypeString},
		).SetAtType(ActionTypes[n])
	}

	return td
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
	//err := test.Config.Setup(&test.AppEnv)
	//if err != nil {
	//	panic("unable to setup test server config")
	//}
	test.Runtime = runtime.NewRuntime(test.Config)
	err := test.Runtime.Start(&test.AppEnv)
	if err != nil {
		panic("unable to start test server runtime: " + err.Error())
	}
}

// Stop the test server
func (test *TestServer) Stop() {
	test.Runtime.Stop()
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
