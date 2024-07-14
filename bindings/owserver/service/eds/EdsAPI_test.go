package eds_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/bindings/owserver/service/eds"
)

// simulation file for testing without OWServer gateway
const owserverSimulation = "../../docs/owserver-simulation.xml"

// TestDiscover requires a live OWServer
func TestDiscover(t *testing.T) {
	addr, err := eds.Discover(3)
	assert.NoError(t, err)
	assert.NotEmpty(t, addr, "EDS OWserver V2 not found")
}

// Read EDS test data from file
func TestReadEdsFromFile(t *testing.T) {
	address := "file://" + owserverSimulation
	rootNode, err := eds.ReadEds(address, "", "")
	assert.NoError(t, err)
	require.NotNil(t, rootNode, "Expected root node")
	assert.True(t, len(rootNode.Nodes) == 20, "Expected 20 parameters and nested")
}

// Read EDS test data from file
func TestReadEdsFromInvalidFile(t *testing.T) {
	// error case, unknown file
	address := "file://../doesnotexist.xml"
	rootNode, err := eds.ReadEds(address, "", "")
	assert.Error(t, err)
	assert.Nil(t, rootNode, "Did not expect root node")
}

// Read EDS gateway and check if more than 1 node is returned.
// A minimum of 1 is expected if the device is online with an additional node
// for each connected node.
// NOTE: This requires a live EDS gateway on the 'edsAddress'
func TestReadEdsFromGW(t *testing.T) {

	// NOTE: This requires a live discoverable OWServer
	edsAddress, err := eds.Discover(3)
	require.NoError(t, err, "Live OWServer not found")

	rootNode, err := eds.ReadEds(edsAddress, "", "")
	assert.NoError(t, err, "Failed reading EDS gateway")
	require.NotNil(t, rootNode, "Expected root node")
	assert.GreaterOrEqual(t, len(rootNode.Nodes), 3, "Expected at least 3 nodes")
}

func TestReadEdsFromInvalidAddress(t *testing.T) {
	// error case - bad hub
	// error case, unknown file
	address := "doesnoteexist"
	rootNode, err := eds.ReadEds(address, "", "")
	assert.Error(t, err)
	assert.Nil(t, rootNode)
}

// Parse the nodes xml file and test for correct results
func TestParseNodeFile(t *testing.T) {
	address := "file://" + owserverSimulation

	rootNode, err := eds.ReadEds(address, "", "")
	require.NoError(t, err)
	require.NotNil(t, rootNode)

	// The test file has hub parameters and 3 connected nodes
	deviceNodes := eds.ParseOneWireNodes(rootNode, 0, true)
	assert.Lenf(t, deviceNodes, 4, "Expected 4 nodes")
}

// TestPollValues reads the EDS and extracts property values of each node
func TestPollValues(t *testing.T) {
	address := "file://" + owserverSimulation
	edsAPI := eds.NewEdsAPI(address, "", "")

	nodes, err := edsAPI.PollNodes()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodes)
}

// there is nothing to write to so make it fail
func TestWriteDataFail(t *testing.T) {
	address := "file://" + owserverSimulation
	edsAPI := eds.NewEdsAPI(address, "", "")

	err := edsAPI.WriteData("badRomID", "temp", "")
	assert.Error(t, err)
}
