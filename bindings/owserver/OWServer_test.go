package owserver_test

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/owserver/config"
	"github.com/hiveot/hub/bindings/owserver/service"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/lib/things"
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
var owsConfig config.OWServerConfig
var owsSimulationFile string // simulation file
var ts *testenv.TestServer

const agentID = "owserver"
const device1ID = "2A000003BB170B28" // <-- from the simulation file

// TestMain run test server and use the project test folder as the home folder.
// All tests are run using the simulation file.
func TestMain(m *testing.M) {
	// setup environment
	var err error
	tempFolder = path.Join(os.TempDir(), "test-owserver")
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
	if err != nil {
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
	t.Log("--- TestStartStop ---")

	svc := service.NewOWServerBinding(&owsConfig)

	hc, _ := ts.AddConnectAgent(agentID)
	require.Equal(t, hubclient.Connected, hc.GetStatus().ConnectionStatus)
	defer hc.Disconnect()

	err := svc.Start(hc)
	require.NoError(t, err)
	// give heartbeat time to run
	time.Sleep(time.Millisecond * 1)
	svc.Stop()
}

func TestPoll(t *testing.T) {
	var tdCount atomic.Int32
	const userID = "user1"

	t.Log("--- TestPoll ---")
	hc, _ := ts.AddConnectAgent(agentID)
	defer hc.Disconnect()
	hc2, _ := ts.AddConnectUser(userID, authn.ClientRoleManager)
	defer hc2.Disconnect()
	svc := service.NewOWServerBinding(&owsConfig)

	// Count the number of received TD events
	err := hc2.Subscribe("", "")
	require.NoError(t, err)
	hc2.SetMessageHandler(func(ev *things.ThingMessage) (stat hubclient.DeliveryStatus) {
		slog.Info("received event", "id", ev.Key)
		if ev.Key == vocab.EventTypeProperties {
			var value map[string]interface{}
			err2 := ev.Decode(&value)
			assert.NoError(t, err2)
		} else {
			var value interface{}
			err2 := ev.Decode(&value)
			assert.NoError(t, err2)
		}
		tdCount.Add(1)
		return stat.Completed(ev, nil, nil)
	})
	assert.NoError(t, err)

	// start the service which publishes TDs
	err = svc.Start(hc)
	require.NoError(t, err)

	// give heartbeat a chance to run. stop will wait for it to complete
	time.Sleep(time.Millisecond * 1)
	svc.Stop()

	// the simulation file contains 3 things. The service is 1 Thing.
	assert.GreaterOrEqual(t, tdCount.Load(), int32(4))

	// get events from the outbox
	dThingID := things.MakeDigiTwinThingID(agentID, device1ID)
	events, err := digitwin.OutboxReadLatest(hc2, nil, vocab.MessageTypeEvent, "", dThingID)
	require.NoError(t, err)
	require.True(t, len(events) > 1)
}

func TestPollInvalidEDSAddress(t *testing.T) {
	t.Log("--- TestPollInvalidEDSAddress ---")

	hc, _ := ts.AddConnectAgent(agentID)
	defer hc.Disconnect()

	badConfig := owsConfig // copy
	badConfig.OWServerURL = "http://invalidAddress/"
	svc := service.NewOWServerBinding(&badConfig)

	err := svc.Start(hc)
	assert.NoError(t, err)
	// give heartbeat a chance to run. stop will wait for it to complete
	time.Sleep(time.Millisecond * 1)
	svc.Stop()

	_, err = svc.PollNodes()
	assert.Error(t, err)
}

func TestAction(t *testing.T) {
	t.Log("--- TestAction ---")
	const user1ID = "operator1"
	// node in test data
	var dThingID = things.MakeDigiTwinThingID(agentID, device1ID)
	var actionName = "RelayFunction" // the action attribute as defined by the device
	var actionValue = "1"

	hc, _ := ts.AddConnectAgent(agentID)
	defer hc.Disconnect()

	svc := service.NewOWServerBinding(&owsConfig)
	err := svc.Start(hc)
	require.NoError(t, err)
	defer svc.Stop()

	// give Start time to run
	time.Sleep(time.Millisecond * 10)

	// note that the simulation file doesn't support writes so this logs an error
	hc2, _ := ts.AddConnectUser(user1ID, authn.ClientRoleOperator)
	require.NoError(t, err)
	defer hc2.Disconnect()
	err = hc2.Rpc(dThingID, actionName, &actionValue, nil)
	// can't write to a simulation
	assert.Error(t, err)

	//time.Sleep(time.Second * 1) // debugging delay
}

func TestConfig(t *testing.T) {
	t.Log("--- TestConfig ---")
	const user1ID = "manager1"
	var configName = "LEDFunction"
	var configValue = ([]byte)("1")

	hc, _ := ts.AddConnectAgent(agentID)
	defer hc.Disconnect()

	svc := service.NewOWServerBinding(&owsConfig)
	err := svc.Start(hc)
	require.NoError(t, err)
	defer svc.Stop()

	// give heartbeat time to run
	time.Sleep(time.Millisecond * 10)

	// note that the simulation file doesn't support writes so this logs an error
	hc2, _ := ts.AddConnectUser(user1ID, authn.ClientRoleManager)
	defer hc2.Disconnect()
	dThingID := things.MakeDigiTwinThingID(agentID, device1ID)
	err = hc2.Rpc(dThingID, configName, &configValue, nil)
	// can't write to a simulation. How to test for real?
	assert.Error(t, err)

	//time.Sleep(time.Second*10)  // debugging delay
}
