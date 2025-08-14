package owserver_test

import (
	"fmt"
	"github.com/hiveot/hub/bindings/owserver/config"
	"github.com/hiveot/hub/bindings/owserver/service"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/tputils"
	authz "github.com/hiveot/hub/runtime/authz/api"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/logging"
)

// TODO: switch for testing with real owserver

var tempFolder string
var storePath string
var owsConfig config.OWServerConfig
var owsSimulationFile string // simulation file
var ts *testenv.TestServer

const agentID = "owserver"

// const device1ID = "2A000003BB170B28" // <-- from the simulation file
const device1ID = "C100100000267C7E" // <-- from the simulation file

// TestMain run test server and use the project test folder as the home folder.
// All tests are run using the simulation file.
func TestMain(m *testing.M) {
	// setup environment
	var err error
	tempFolder = path.Join(os.TempDir(), "test-owserver")
	storePath = path.Join(tempFolder, "owserver-state")
	cwd, _ := os.Getwd()
	homeFolder := path.Join(cwd, "./docs")
	owsSimulationFile = "file://" + path.Join(homeFolder, "owserver-simulation.xml")
	// uncomment the next line to discover and test with a real owserver
	//owsSimulationFile = ""
	logging.SetLogging("info", "")

	owsConfig = *config.NewConfig()
	owsConfig.OWServerURL = owsSimulationFile
	//
	ts = testenv.StartTestServer(true)
	if ts == nil {
		panic("unable to start test server: " + err.Error())
	}
	result := m.Run()
	time.Sleep(time.Millisecond * 10)
	println("stopping testserver")

	ts.Stop()
	if result == 0 {
		_ = os.RemoveAll(tempFolder)
	}
	os.Exit(result)
}

func TestStartStop(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))

	svc := service.NewOWServerBinding(storePath, &owsConfig)

	ag, _, _ := ts.AddConnectAgent(agentID)
	//connected := ag.IsConnected()
	//require.Equal(t, true, connected)
	defer ag.Disconnect()

	err := svc.Start(ag)
	require.NoError(t, err)
	// give heartbeat time to run
	time.Sleep(time.Millisecond * 1)
	svc.Stop()
}

func TestPoll(t *testing.T) {
	var tdCount atomic.Int32
	const userID = "user1"

	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	ag1, _, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()
	co1, _, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer co1.Disconnect()
	svc := service.NewOWServerBinding(storePath, &owsConfig)

	// Count the number of received TD events
	err := co1.ObserveProperty("", "")
	err = co1.Subscribe("", "")
	require.NoError(t, err)
	co1.SetNotificationHandler(func(msg *messaging.NotificationMessage) {
		slog.Info("received notification", "operation", msg.Operation, "id", msg.Name)
		var value interface{}
		err2 := tputils.DecodeAsObject(msg.Data, &value)
		assert.NoError(t, err2)

		tdCount.Add(1)
	})
	assert.NoError(t, err)

	// start the service which publishes TDs
	err = svc.Start(ag1)
	require.NoError(t, err)

	// give heartbeat a chance to run. stop will wait for it to complete
	time.Sleep(time.Millisecond * 100)
	svc.Stop()

	// the simulation file contains 3 things. The service is 1 Thing.
	assert.GreaterOrEqual(t, tdCount.Load(), int32(4))

	// get events from the digitwin
	dThingID := td.MakeDigiTwinThingID(agentID, device1ID)
	events, err := digitwin.ThingValuesReadAllEvents(co1, dThingID)
	require.NoError(t, err)
	// this thing has 5 sensors and 4 alarm events
	require.Equal(t, 9, len(events))
}

func TestPollInvalidEDSAddress(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))

	hc, _, _ := ts.AddConnectAgent(agentID)
	defer hc.Disconnect()

	badConfig := owsConfig // copy
	badConfig.OWServerURL = "http://invalidAddress/"
	svc := service.NewOWServerBinding(storePath, &badConfig)

	err := svc.Start(hc)
	assert.NoError(t, err)
	// give heartbeat a chance to run. stop will wait for it to complete
	time.Sleep(time.Millisecond * 1)
	svc.Stop()

	_, err = svc.PollNodes()
	assert.Error(t, err)
}

func TestAction(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	const user1ID = "operator1"
	// node in test data
	var dThingID = td.MakeDigiTwinThingID(agentID, device1ID)
	var actionName = "RelayFunction" // the action attribute as defined by the device
	var actionValue = "1"

	ag1, _, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()

	svc := service.NewOWServerBinding(storePath, &owsConfig)
	err := svc.Start(ag1)
	require.NoError(t, err)
	defer svc.Stop()

	// give Start time to run
	time.Sleep(time.Millisecond * 10)

	// note that the simulation file doesn't support writes so this logs an error
	co1, _, _ := ts.AddConnectConsumer(user1ID, authz.ClientRoleOperator)
	require.NoError(t, err)
	defer co1.Disconnect()
	err = co1.WriteProperty(dThingID, actionName, &actionValue, true)

	//err = co1.SendRequest(dThingID, actionName, &actionValue, nil)
	// can't write to a simulation
	assert.Error(t, err)

	//time.Sleep(time.Second * 1) // debugging delay
}

func TestConfig(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	const user1ID = "manager1"
	var configName = "LEDState"
	var configValue = "1"

	ag1, _, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()

	svc := service.NewOWServerBinding(storePath, &owsConfig)
	err := svc.Start(ag1)
	require.NoError(t, err)
	defer svc.Stop()

	// give heartbeat time to run
	time.Sleep(time.Millisecond * 10)

	// note that the simulation file doesn't support writes so this logs an error
	co1, _, _ := ts.AddConnectConsumer(user1ID, authz.ClientRoleManager)
	defer co1.Disconnect()
	dThingID := td.MakeDigiTwinThingID(agentID, device1ID)
	err = co1.WriteProperty(dThingID, configName, &configValue, true)

	// can't write to a simulation file. Write should fail.
	assert.Error(t, err)

	//time.Sleep(time.Second*10)  // debugging delay
}
