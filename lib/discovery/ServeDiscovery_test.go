package discovery_test

import (
	"fmt"
	"github.com/hiveot/hub/lib/discovery"
	"github.com/hiveot/hub/lib/utils"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testServiceID = "discovery-test"
const testServiceName = "test-service"
const testServicePath = "/discovery/path"
const testServicePort = 9999

// Test the discovery client and server
func TestDiscover(t *testing.T) {
	params := map[string]string{"path": testServicePath}
	testServiceAddress := utils.GetOutboundIP("").String()

	discoServer, err := discovery.ServeDiscovery(
		testServiceID, testServiceName, testServiceAddress, testServicePort, params)

	assert.NoError(t, err)
	assert.NotNil(t, discoServer)

	// Test if it is discovered
	address, port, discoParams, records, err := discovery.DiscoverService(testServiceName, 1)
	require.NoError(t, err)
	rec0 := records[0]
	assert.Equal(t, testServiceID, rec0.Instance)
	assert.Equal(t, testServiceAddress, address)
	assert.Equal(t, testServicePort, port)
	assert.Equal(t, testServicePath, discoParams["path"])

	time.Sleep(time.Millisecond) // prevent race error in discovery.server
	discoServer.Shutdown()
}

func TestDiscoViaDomainName(t *testing.T) {
	testServiceAddress := "localhost"

	discoServer, err := discovery.ServeDiscovery(
		testServiceID, testServiceName, testServiceAddress, testServicePort, nil)

	assert.NoError(t, err)
	assert.NotNil(t, discoServer)

	// Test if it is discovered
	discoAddress, discoPort, _, records, err := discovery.DiscoverService(testServiceName, time.Second)
	require.NoError(t, err)
	require.True(t, len(records) > 0)
	rec0 := records[0]
	assert.Equal(t, "127.0.0.1", discoAddress)
	assert.True(t, strings.HasPrefix(rec0.HostName, testServiceAddress))
	assert.Equal(t, testServicePort, discoPort)

	time.Sleep(time.Millisecond) // prevent race error in discovery.server
	discoServer.Shutdown()
}

func TestDiscoverBadPort(t *testing.T) {
	serviceID := "idprov-test"
	badPort := 0
	address := utils.GetOutboundIP("").String()
	_, err := discovery.ServeDiscovery(
		serviceID, testServiceName, address, badPort, nil)

	assert.Error(t, err)
}

func TestNoInstanceID(t *testing.T) {
	serviceID := "serviceID"
	address := utils.GetOutboundIP("").String()

	_, err := discovery.ServeDiscovery(
		"", testServiceName, address, testServicePort, nil)
	assert.Error(t, err) // missing instance name

	_, err = discovery.ServeDiscovery(
		serviceID, "", address, testServicePort, nil)
	assert.Error(t, err) // missing service name
}

func TestDiscoverNotFound(t *testing.T) {
	instanceID := "idprov-test-id"
	serviceName := "idprov-test"
	address := utils.GetOutboundIP("").String()

	discoServer, err := discovery.ServeDiscovery(
		instanceID, serviceName, address, testServicePort, nil)

	assert.NoError(t, err)

	// Test if it is discovered
	discoAddress, discoPort, _, records, err := discovery.DiscoverService("wrongname", 1)
	_ = discoAddress
	_ = discoPort
	_ = records
	assert.Error(t, err)

	time.Sleep(time.Millisecond) // prevent race error in discovery.server
	discoServer.Shutdown()
	assert.Error(t, err)
}

func TestBadAddress(t *testing.T) {
	instanceID := "idprov-test-id"

	discoServer, err := discovery.ServeDiscovery(
		instanceID, testServiceName, "notanipaddress", testServicePort, nil)

	assert.Error(t, err)
	assert.Nil(t, discoServer)
}

func TestExternalAddress(t *testing.T) {
	instanceID := "idprov-test-id"

	discoServer, err := discovery.ServeDiscovery(
		instanceID, testServiceName, "1.2.3.4", testServicePort, nil)

	// expect a warning
	assert.NoError(t, err)
	time.Sleep(time.Millisecond) // prevent race error in discovery.server
	discoServer.Shutdown()
}

func TestDNSSDScan(t *testing.T) {

	records, err := discovery.DnsSDScan("", 2)
	fmt.Printf("Found %d records in scan", len(records))

	assert.NoError(t, err)
	assert.Greater(t, len(records), 0, "No DNS records found")
}
