package testenv

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/runtime"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients"
	"github.com/hiveot/hub/transports/messaging"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"math/rand"
	"os"
	"path"
	"time"
)

// TestDir is the default test directory
var TestDir = path.Join(os.TempDir(), "hiveot-test")

const TestHttpsPort = 9444
const TestMqttTcpPort = 9883
const TestMqttWssPort = 9884

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
	Certs          certs.TestCertBundle
	TestDir        string
	ConnectTimeout time.Duration
	//
	Config  *runtime.RuntimeConfig
	AppEnv  plugin.AppEnvironment
	Runtime *runtime.Runtime
	// which protocol each client uses
	AgentProtocol    string
	ServiceProtocol  string
	ConsumerProtocol string
}

// GetAgentConnection returns a hub connection for an agent and protocol.
// This sets 'getForm' to the handler provided by the protocol server. For testing only.
func (test *TestServer) GetAgentConnection(agentID string, protocolName string) (transports.IClientConnection, *messaging.Agent) {
	connectURL := test.Runtime.TransportsMgr.GetConnectURL(protocolName)

	server := test.Runtime.TransportsMgr.GetServer(protocolName)
	cc, _ := clients.NewClient(connectURL, agentID, test.Certs.CaCert, server.GetForm, test.ConnectTimeout)
	agent := messaging.NewAgent(cc, nil, nil, nil, test.ConnectTimeout)
	return cc, agent
}

// GetConsumerConnection returns a hub connection for a consumer and protocol.
// This sets 'getForm' to the handler provided by the protocol server. For testing only.
func (test *TestServer) GetConsumerConnection(
	clientID string, protocolName string) (transports.IClientConnection, *messaging.Consumer) {

	getForm := func(op string, thingID string, name string) *td.Form {
		return test.Runtime.GetForm(op, protocolName)
	}

	connectURL := test.Runtime.TransportsMgr.GetConnectURL(protocolName)

	cc, _ := clients.NewClient(connectURL, clientID, test.Certs.CaCert, getForm, test.ConnectTimeout)
	consumer := messaging.NewConsumer(cc, nil, nil, test.ConnectTimeout)
	return cc, consumer
}

// AddConnectConsumer creates a new test user with the given role and returns a client, consumer and a new session token.
// In case of error this panics.
func (test *TestServer) AddConnectConsumer(
	clientID string, clientRole authz.ClientRole) (consumer *messaging.Consumer, token string) {

	password := clientID
	err := test.Runtime.AuthnSvc.AdminSvc.AddConsumer(clientID,
		authn.AdminAddConsumerArgs{ClientID: clientID, DisplayName: "my name", Password: password})
	if err == nil {
		err = test.Runtime.AuthzSvc.SetClientRole(clientID,
			authz.AdminSetClientRoleArgs{ClientID: clientID, Role: clientRole})
	}
	if err != nil {
		panic("Failed adding client:" + err.Error())
	}

	cc, consumer := test.GetConsumerConnection(clientID, test.ConsumerProtocol)
	token, err = cc.ConnectWithPassword(password)

	if err != nil {
		panic("Failed connect with password:" + err.Error())
	}
	return consumer, token
}

// AddConnectAgent creates a new agent test client.
// Agents use non-session tokens and survive a server restart.
// This returns the agent's connection token.
func (test *TestServer) AddConnectAgent(agentID string) (agent *messaging.Agent, token string) {

	token, err := test.Runtime.AuthnSvc.AdminSvc.AddAgent(agentID,
		authn.AdminAddAgentArgs{ClientID: agentID, DisplayName: "agent name"})

	if err == nil {
		err = test.Runtime.AuthzSvc.SetClientRole(agentID,
			authz.AdminSetClientRoleArgs{ClientID: agentID, Role: authz.ClientRoleAgent})
	}
	if err != nil {
		panic("AddConnectAgent: Failed adding client:" + err.Error())
	}
	cc, agent := test.GetAgentConnection(agentID, test.AgentProtocol)

	err = cc.ConnectWithToken(token)
	if err != nil {
		err = fmt.Errorf("AddConnectAgent: Failed connecting using token. "+
			"SenderID='%s': %s", agentID, err.Error())
		panic(err)
	}

	return agent, token
}

// AddConnectService creates a new service test client.
// Services are agents and use non-session tokens and survive a server restart.
// This returns the service's connection token.
//
// This sets the getForm handler of the client to the protocol server. (for testing)
//
// clientType can be one of ClientTypeAgent or ClientTypeService
func (test *TestServer) AddConnectService(serviceID string) (ag *messaging.Agent, token string) {

	// 1: register the service
	token, err := test.Runtime.AuthnSvc.AdminSvc.AddService(serviceID,
		authn.AdminAddServiceArgs{ClientID: serviceID, DisplayName: "service name"})
	if err == nil {
		err = test.Runtime.AuthzSvc.SetClientRole(serviceID,
			authz.AdminSetClientRoleArgs{ClientID: serviceID, Role: authz.ClientRoleService})
	}
	if err != nil {
		panic("AddConnectService: Failed adding client:" + err.Error())
	}
	// 2: create and connect a client. The getForm is needed for agents to call other services.
	connectURL := test.Runtime.TransportsMgr.GetConnectURL(test.ServiceProtocol)
	server := test.Runtime.TransportsMgr.GetServer(test.ServiceProtocol)
	getForm := server.GetForm
	cc, err := clients.NewClient(connectURL, serviceID, test.Certs.CaCert, getForm, test.ConnectTimeout)
	if err == nil {
		err = cc.ConnectWithToken(token)
		ag = messaging.NewAgent(cc, nil, nil, nil, 0)
	}
	if err != nil {
		panic("AddConnectService: Failed connecting using token. serviceID=" + serviceID)
	}

	return ag, token
}

// AddTD adds a test TD document to the runtime
// if td is nil then a random TD will be added
func (test *TestServer) AddTD(agentID string, td *td.TD) *td.TD {
	if td == nil {
		i := rand.Intn(99882)
		td = test.CreateTestTD(i)
	}
	tdJSON, _ := jsoniter.MarshalToString(td)
	err := test.Runtime.DigitwinSvc.DirSvc.UpdateTD(agentID, tdJSON)
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
func (test *TestServer) CreateTestTD(i int) (tdi *td.TD) {
	ttd := testTDs[0]
	if i < len(testTDs) {
		ttd = testTDs[i]
	} else {
		ttd.ID = fmt.Sprintf("thing-%d", rand.Intn(99823))
	}

	tdi = td.NewTD(ttd.ID, ttd.Title, ttd.DeviceType)
	// add random properties
	for n := 0; n < ttd.NrProps; n++ {
		propName := fmt.Sprintf("prop-%d", n)
		tdi.AddProperty(propName, "title-"+PropTypes[n], "", vocab.WoTDataTypeString).
			SetAtType(PropTypes[n])
	}
	// add random events
	for n := 0; n < ttd.NrEvents; n++ {
		evName := fmt.Sprintf("event-%d", n)
		tdi.AddEvent(evName, "title-"+EventTypes[n], "",
			&td.DataSchema{Type: vocab.WoTDataTypeString}).
			SetAtType(EventTypes[n])
	}
	// add random actions
	for n := 0; n < ttd.NrActions; n++ {
		actionName := fmt.Sprintf("action-%d", n)
		tdi.AddAction(actionName, "title-"+ActionTypes[n], "",
			&td.DataSchema{Type: vocab.WoTDataTypeString},
		).SetAtType(ActionTypes[n])
	}

	return tdi
}

// GetForm returns the form for the given operation and transport protocol binding
func (test *TestServer) GetForm(op string, thingID string, name string) *td.Form {

	// test default test server consumer protocol (only consumers need forms)
	srv := test.Runtime.TransportsMgr.GetServer(test.ConsumerProtocol)
	return srv.GetForm(op, thingID, name)
}

// GetServerURL returns the default connection URL to use for the given client type
func (test *TestServer) GetServerURL(clientType authn.ClientType) string {
	if clientType == authn.ClientTypeService {
		return test.Runtime.GetConnectURL(test.ServiceProtocol)
	} else if clientType == authn.ClientTypeAgent {
		return test.Runtime.GetConnectURL(test.AgentProtocol)
	}
	return test.Runtime.GetConnectURL(test.ConsumerProtocol)
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
	test.Config.ProtocolConfig.HttpHost = "localhost"
	test.Config.ProtocolConfig.HttpsPort = TestHttpsPort
	test.Config.ProtocolConfig.MqttHost = "localhost"
	test.Config.ProtocolConfig.MqttTcpPort = TestMqttTcpPort
	test.Config.ProtocolConfig.MqttWssPort = TestMqttWssPort
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
		TestDir: TestDir,
		Certs:   certs.CreateTestCertBundle(),
		Config:  runtime.NewRuntimeConfig(),
		// change these for running all tests with different protocols
		AgentProtocol:    transports.ProtocolTypeHiveotWSS,
		ServiceProtocol:  transports.ProtocolTypeHiveotWSS,
		ConsumerProtocol: transports.ProtocolTypeHiveotWSS,
		ConnectTimeout:   time.Second * 120, // testing extra long
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
