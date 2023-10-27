package internal_test

import (
	"os"
	"testing"
	"time"

	"github.com/iotdomain/iotdomain-go/messaging"
	"github.com/iotdomain/iotdomain-go/nodes"
	"github.com/iotdomain/iotdomain-go/publisher"
	"github.com/iotdomain/iotdomain-go/types"
	"github.com/iotdomain/isy99/internal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Use simulation files
const deckLightsID = "15 2D A 1"
const appID = "isy99"

// For testing, IsyAPI.isyRequest simulates reading isy from file using the path:
//  gatewayaddress[7:]/restpath.xml, where restpath is the isy REST api path and
// .xml is appended   path of the simulation test files with
// For example reading the isy gateway: ../test/rest/config.xml
var appConfig = &internal.IsyAppConfig{GatewayAddress: "file://../test"}

var testConfigFolder = "../test"
var nodesFile = testConfigFolder + "/isy99-nodes.json"
var messengerConfig = &messaging.MessengerConfig{Domain: "test"}

// Read ISY device and check if more than 1 node is returned. A minimum of 1 is expected if the device is online with
// an additional node for each connected node.
func TestReadIsyGateway(t *testing.T) {
	os.Remove(nodesFile)
	_, err := publisher.NewAppPublisher(appID, testConfigFolder, appConfig, "", false)
	assert.NoError(t, err)

	isyAPI := internal.NewIsyAPI(appConfig.GatewayAddress, appConfig.LoginName, appConfig.Password)
	// use a simulation file
	isyDevice, err := isyAPI.ReadIsyGateway()
	assert.NoError(t, err)
	assert.NotEmptyf(t, isyDevice.Configuration.AppVersion, "Expected an application version")

	// use a simulation file
	isyNodes, err := isyAPI.ReadIsyNodes()
	if assert.NoError(t, err) {
		assert.True(t, len(isyNodes.Nodes) > 5, "Expected 6 ISY nodes. Got fewer.")
	}
	//  error case - error reading gateway
	isyAPI = internal.NewIsyAPI("localhost", appConfig.LoginName, appConfig.Password)
	isyDevice, err = isyAPI.ReadIsyGateway()
	assert.Error(t, err)
}
func TestReadIsyStatus(t *testing.T) {
	isyAPI := internal.NewIsyAPI(appConfig.GatewayAddress, appConfig.LoginName, appConfig.Password)
	// use a simulation file
	status, err := isyAPI.ReadIsyStatus()
	assert.NoError(t, err)
	assert.NotNil(t, status)

	// error case - gateway doesn't exist
	isyAPI = internal.NewIsyAPI("localhost", appConfig.LoginName, appConfig.Password)
	_, err = isyAPI.ReadIsyStatus()
	assert.Error(t, err)

	// error case - simulation file doesn't exist
	isyAPI = internal.NewIsyAPI("file://doesn't exist", appConfig.LoginName, appConfig.Password)
	_, err = isyAPI.ReadIsyStatus()
	assert.Error(t, err)

}
func TestReadWriteIsyDevice(t *testing.T) {
	// error case - write to non existing gateway
	isyAPI := internal.NewIsyAPI(appConfig.GatewayAddress, appConfig.LoginName, appConfig.Password)
	err := isyAPI.WriteOnOff(deckLightsID, false)
	assert.NoError(t, err)

	// error case - write to non existing gateway
	isyAPI = internal.NewIsyAPI("localhost", appConfig.LoginName, appConfig.Password)
	err = isyAPI.WriteOnOff(deckLightsID, false)
	assert.Error(t, err)
}

func TestIsyAppGateway(t *testing.T) {
	os.Remove(nodesFile)
	pub, err := publisher.NewAppPublisher(appID, testConfigFolder, appConfig, "", false)
	assert.NoError(t, err)

	// appconfig, read from test/isy99.yaml, contains simulated gateway file
	app := internal.NewIsyApp(appConfig, pub)
	pub.Start()
	gwNodeID, err := app.ReadGateway()
	assert.NoError(t, err)
	assert.NotEmpty(t, gwNodeID)

	// error case - use real url
	appConfig.GatewayAddress = "localhost"
	app = internal.NewIsyApp(appConfig, pub)
	gwNodeID, err = app.ReadGateway()
	assert.Error(t, err)

	// error case - update devices, should not panic
	app.UpdateDevices()

	pub.Stop()
}

func TestIsyAppConfig(t *testing.T) {
	os.Remove(nodesFile)
	pub, err := publisher.NewAppPublisher(appID, testConfigFolder, appConfig, "", false)
	assert.NoError(t, err)
	app := internal.NewIsyApp(appConfig, pub)
	pub.Start()

	gwNodeID, err := app.ReadGateway()
	gwNodeAddr := nodes.MakeNodeConfigureAddress(pub.Domain(), pub.PublisherID(), gwNodeID)
	pub.PublishNodeConfigure(gwNodeAddr, types.NodeAttrMap{types.NodeAttrName: "test"})

	name := pub.GetNodeAttr(gwNodeID, types.NodeAttrName)
	assert.Equal(t, "test", name)
	pub.Stop()
}

func TestIsyAppPoll(t *testing.T) {
	os.Remove(nodesFile)
	pub, err := publisher.NewAppPublisher(appID, testConfigFolder, appConfig, "", false)
	assert.NoError(t, err)

	app := internal.NewIsyApp(appConfig, pub)
	pub.Start()
	assert.NoError(t, err)
	app.Poll(pub)
	time.Sleep(3 * time.Second)
	pub.Stop()
}

// This simulates the switch
func TestSwitch(t *testing.T) {
	os.Remove(nodesFile)
	pub, err := publisher.NewAppPublisher(appID, testConfigFolder, appConfig, "", false)
	assert.NoError(t, err)

	app := internal.NewIsyApp(appConfig, pub)
	app.SetupGatewayNode(pub)

	// FIXME: load isy nodes from file
	pub.Start()
	assert.NoError(t, err)
	app.Poll(pub)
	// some time to publish stuff
	time.Sleep(2 * time.Second)

	// throw a switch
	deckSwitch := pub.GetNodeByHWID(deckLightsID)
	require.NotNilf(t, deckSwitch, "Switch %s not found", deckLightsID)

	switchInput := pub.GetInputByNodeHWID(deckSwitch.HWID, types.InputTypeSwitch, types.DefaultInputInstance)
	require.NotNil(t, switchInput, "Input of switch node not found on address %s", deckSwitch.Address)
	// switchInput := deckSwitch.GetInput(types.InputTypeSwitch)

	logrus.Infof("TestSwitch: --- Switching deck switch %s OFF", deckSwitch.Address)

	pub.PublishSetInput(switchInput.Address, "false")
	assert.NoError(t, err)
	time.Sleep(2 * time.Second)

	// fetch result
	switchOutput := pub.GetOutputByNodeHWID(deckLightsID, types.OutputTypeSwitch, types.DefaultOutputInstance)
	// switchOutput := deckSwitch.GetOutput(types.InputTypeSwitch)
	require.NotNilf(t, switchOutput, "Output switch of node %s not found", deckLightsID)

	outputValue := pub.GetOutputValueByNodeHWID(switchOutput.NodeHWID, types.OutputTypeSwitch, types.DefaultOutputInstance)
	assert.Equal(t, "false", outputValue.Value)

	logrus.Infof("TestSwitch: --- Switching deck switch %s ON", deckSwitch.Address)
	require.NotNil(t, switchInput, "Input switch of node %s not found", deckSwitch.NodeID)
	pub.PublishSetInput(switchInput.Address, "true")

	time.Sleep(3 * time.Second)
	outputValue = pub.GetOutputValueByNodeHWID(switchOutput.NodeHWID, types.OutputTypeSwitch, types.DefaultOutputInstance)
	assert.Equal(t, "true", outputValue.Value)

	// be nice and turn the light back off
	pub.PublishSetInput(switchInput.Address, "false")

	pub.Stop()

}

func TestStartStop(t *testing.T) {
	pub, err := publisher.NewAppPublisher(appID, testConfigFolder, appConfig, "", false)
	assert.NoError(t, err)

	// app := NewIsyApp(appConfig, pub)

	pub.Start()
	time.Sleep(time.Second * 10)
	pub.Stop()
}
