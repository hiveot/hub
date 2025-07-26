package isy99x_test

import (
	"fmt"
	"github.com/hiveot/hub/bindings/isy99x/config"
	"github.com/hiveot/hub/bindings/isy99x/service"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// For testing, IsyGW.isyRequest simulates reading isy from file using the path:
//
//	gatewayaddress[7:]/restpath.xml, where restpath is the isy REST api path and
//
// .xml is appended   path of the simulation test files with
// For example reading the isy gateway: ../test/rest/config.xml
var appConfig = &config.Isy99xConfig{}

var testConfigFolder = "../test"
var nodesFile = testConfigFolder + "/isy99-nodes.json"

const agentID = "isy99x"

// set in TestMain
var tempDir = path.Join(os.TempDir(), "test-isy99x")
var ts *testenv.TestServer

// TestMain run test server and use the project test folder as the home folder.
// All tests are run using the simulation file.
func TestMain(m *testing.M) {
	// setup environment

	cwd, _ := os.Getwd()
	simulationRoot := "file://" + path.Join(cwd, "test")
	logging.SetLogging("info", "")

	appConfig.IsyAddress = simulationRoot

	//
	ts = testenv.StartTestServer(true)
	result := m.Run()
	time.Sleep(time.Second)

	ts.Stop()
	if result == 0 {
		_ = os.RemoveAll(tempDir)
	}

	os.Exit(result)
}

func TestStartStop(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	_ = os.Remove(nodesFile)

	// appconfig, read from test/isy99.yaml, contains simulated gateway file
	hc, _ := ts.AddConnectService(agentID)
	defer hc.Disconnect()

	svc := service.NewIsyBinding(appConfig)
	err := svc.Start(hc)
	require.NoError(t, err)
	defer svc.Stop()

	err = svc.IsyGW.ReadIsyThings()
	require.NoError(t, err)

	time.Sleep(time.Second)
	devices := svc.IsyGW.GetIsyThings()
	assert.True(t, len(devices) > 5, "Expected 6 ISY nodes. Got fewer.")

	time.Sleep(time.Millisecond)
}

func TestBadAddress(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	_ = os.Remove(nodesFile)

	hc, _ := ts.AddConnectService(agentID)
	defer hc.Disconnect()

	// error case - use real url
	badConfig := *appConfig
	badConfig.IsyAddress = "localhost"
	svc := service.NewIsyBinding(&badConfig)
	err := svc.Start(hc)
	assert.NoError(t, err)
	defer svc.Stop()

	err = svc.IsyGW.ReadIsyThings()
	assert.Error(t, err)
	time.Sleep(time.Millisecond * 100)
}

func TestIsyAppPoll(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	_ = os.Remove(nodesFile)
	// appconfig, read from test/isy99.yaml, contains simulated gateway file
	hc, _ := ts.AddConnectService(agentID)
	defer hc.Disconnect()

	svc := service.NewIsyBinding(appConfig)
	err := svc.Start(hc)
	require.NoError(t, err)
	defer svc.Stop()

	err = svc.PublishTDs()
	assert.NoError(t, err)
	time.Sleep(2 * time.Second)
}

// This simulates the switch
func TestSwitch(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	const deckThingLightsID = "13 57 73 1" // from simulation file
	const name = "value"

	_ = os.Remove(nodesFile)
	// appconfig, read from test/isy99.yaml, contains simulated gateway file
	hc, _ := ts.AddConnectService(agentID)
	defer hc.Disconnect()

	svc := service.NewIsyBinding(appConfig)
	err := svc.Start(hc)
	require.NoError(t, err)
	defer svc.Stop()

	err = svc.PublishTDs()
	assert.NoError(t, err)
	// some time to publish stuff
	time.Sleep(1 * time.Second)
	//
	//// throw a switch
	////cl := hc.HandleActionFlow("isy", nodeID, vocab.VocabValue, "true")
	//svc.SwitchOnOff(deckThingLightsID, name, "true")
	//deckSwitch := svc.GetNode(deckThingLightsID)
	//require.NotNilf(t, deckSwitch, "Switch %s not found", deckLightsID)
	//
	//switchInput := svc.GetInputByNodeHWID(deckSwitch.HWID, types.InputTypeSwitch, types.DefaultInputInstance)
	//require.NotNil(t, switchInput, "Input of switch node not found on address %s", deckSwitch.Address)
	//// switchInput := deckSwitch.GetInput(types.InputTypeSwitch)
	//
	//t.Logf("TestSwitch: --- Switching deck switch %s OFF", deckSwitch.Address)
	//
	//svc.PublishSetInput(switchInput.Address, "false")
	//assert.NoError(t, err)
	//time.Sleep(2 * time.Second)
	//
	//// fetch result
	//switchOutput := pub.GetOutputByNodeHWID(deckLightsID, types.OutputTypeSwitch, types.DefaultOutputInstance)
	//// switchOutput := deckSwitch.GetOutput(types.InputTypeSwitch)
	//require.NotNilf(t, switchOutput, "Value switch of node %s not found", deckLightsID)
	//
	//outputValue := pub.GetOutputValueByNodeHWID(switchOutput.NodeHWID, types.OutputTypeSwitch, types.DefaultOutputInstance)
	//assert.Equal(t, "false", outputValue.Value)
	//
	//t.Logf("TestSwitch: --- Switching deck switch %s ON", deckSwitch.Address)
	//require.NotNil(t, switchInput, "Input switch of node %s not found", deckSwitch.NodeID)
	//svc.PublishSetInput(switchInput.Address, "true")
	//
	//time.Sleep(3 * time.Second)
	//outputValue = pub.GetOutputValueByNodeHWID(switchOutput.NodeHWID, types.OutputTypeSwitch, types.DefaultOutputInstance)
	//assert.Equal(t, "true", outputValue.Value)
	//
	//// be nice and turn the light back off
	//svc.PublishSetInput(switchInput.Address, "false")

	svc.Stop()

}
