package isyapi_test

import (
	"github.com/hiveot/hub/bindings/isy99x/service/isyapi"
	"github.com/stretchr/testify/assert"
	"testing"
)

// Use simulation files
const deckLightsID = "15 2D A 1"

// var simFile = "file://bindings/isy99x/test"
var simFile = "file://../../test"

// TestDiscover requires a live ISY99x gateway
func TestDiscover(t *testing.T) {
	addr, err := isyapi.Discover(3)
	assert.NoError(t, err)
	assert.NotEmpty(t, addr, "ISY99x gateway not found")
}

// Read simulation file from ISY device and check if more than 1 node is returned.
// A minimum of 1 is expected if the device is online with an additional node for each connected node.
func TestReadIsyGateway(t *testing.T) {

	isyAddr := simFile
	loginName := ""
	password := ""

	isyAPI := isyapi.NewIsyAPI(isyAddr, loginName, password)
	// use a simulation file
	isyDevice, err := isyAPI.ReadIsyGateway()
	assert.NoError(t, err)
	assert.NotEmptyf(t, isyDevice.Configuration.AppVersion, "Expected an application version")

	// use a simulation file
	isyNodes, err := isyAPI.ReadIsyNodes()
	if assert.NoError(t, err) {
		assert.True(t, len(isyNodes.Nodes) > 5, "Expected 6 ISY nodes. Got fewer.")
	}
}

// Read from ISY when not having a connection
func TestReadFromInvalidAddress(t *testing.T) {

	isyAddr := "doesnoteexist"
	loginName := ""
	password := ""
	//  error case - error reading gateway
	isyAPI := isyapi.NewIsyAPI(isyAddr, loginName, password)
	isyDevice, err := isyAPI.ReadIsyGateway()
	_ = isyDevice
	assert.Error(t, err)
}

func TestReadIsyStatus(t *testing.T) {
	isyAddr := simFile
	loginName := ""
	password := ""

	isyAPI := isyapi.NewIsyAPI(isyAddr, loginName, password)

	// use a simulation file
	status, err := isyAPI.ReadIsyStatus()
	assert.NoError(t, err)
	assert.NotNil(t, status)
}

func TestReadIsyStatusBadAddr(t *testing.T) {
	isyAddr := "not an address"
	loginName := ""
	password := ""

	isyAPI := isyapi.NewIsyAPI(isyAddr, loginName, password)

	// error case - gateway doesn't exist
	isyAPI = isyapi.NewIsyAPI(isyAddr, loginName, password)
	_, err := isyAPI.ReadIsyStatus()
	assert.Error(t, err)

	// error case - simulation file doesn't exist
	isyAPI = isyapi.NewIsyAPI("file://doesn't exist", loginName, password)
	_, err = isyAPI.ReadIsyStatus()
	assert.Error(t, err)

}

func TestReadWriteIsySimFile(t *testing.T) {
	isyAddr := simFile
	loginName := ""
	password := ""

	// use simulation file
	isyAPI := isyapi.NewIsyAPI(isyAddr, loginName, password)
	err := isyAPI.WriteOnOff(deckLightsID, false)
	assert.NoError(t, err)
}

func TestReadWriteIsyNoGateway(t *testing.T) {
	isyAddr := "nogateway"
	loginName := ""
	password := ""

	// error case - write to non existing gateway
	isyAPI := isyapi.NewIsyAPI(isyAddr, loginName, password)
	err := isyAPI.WriteOnOff(deckLightsID, false)
	assert.Error(t, err)

}
