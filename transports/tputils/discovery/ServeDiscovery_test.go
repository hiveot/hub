package discovery_test

import (
	"fmt"
	discovery2 "github.com/hiveot/hub/transports/tputils/discovery"
	"github.com/hiveot/hub/transports/tputils/net"
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
	testServiceAddress := net.GetOutboundIP("").String()

	discoServer, err := discovery2.ServeDiscovery(
		testServiceID, testServiceName, testServiceAddress, testServicePort, params)

	assert.NoError(t, err)
	assert.NotNil(t, discoServer)

	// Test if it is discovered
	address, port, discoParams, records, err := discovery2.DiscoverService(
		testServiceName, time.Second, true)
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

	discoServer, err := discovery2.ServeDiscovery(
		testServiceID, testServiceName, testServiceAddress, testServicePort, nil)

	assert.NoError(t, err)
	assert.NotNil(t, discoServer)

	// Test if it is discovered
	discoAddress, discoPort, _, records, err := discovery2.DiscoverService(
		testServiceName, time.Second, true)
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
	address := net.GetOutboundIP("").String()
	_, err := discovery2.ServeDiscovery(
		serviceID, testServiceName, address, badPort, nil)

	assert.Error(t, err)
}

func TestNoInstanceID(t *testing.T) {
	serviceID := "serviceID"
	address := net.GetOutboundIP("").String()

	_, err := discovery2.ServeDiscovery(
		"", testServiceName, address, testServicePort, nil)
	assert.Error(t, err) // missing instance name

	_, err = discovery2.ServeDiscovery(
		serviceID, "", address, testServicePort, nil)
	assert.Error(t, err) // missing service name
}

func TestDiscoverNotFound(t *testing.T) {
	instanceID := "idprov-test-id"
	serviceName := "idprov-test"
	address := net.GetOutboundIP("").String()

	discoServer, err := discovery2.ServeDiscovery(
		instanceID, serviceName, address, testServicePort, nil)

	assert.NoError(t, err)

	// Test if it is discovered
	discoAddress, discoPort, _, records, err := discovery2.DiscoverService(
		"wrongname", time.Second, true)
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

	discoServer, err := discovery2.ServeDiscovery(
		instanceID, testServiceName, "notanipaddress", testServicePort, nil)

	assert.Error(t, err)
	assert.Nil(t, discoServer)
}

func TestExternalAddress(t *testing.T) {
	instanceID := "idprov-test-id"

	discoServer, err := discovery2.ServeDiscovery(
		instanceID, testServiceName, "1.2.3.4", testServicePort, nil)

	// expect a warning
	assert.NoError(t, err)
	time.Sleep(time.Millisecond) // prevent race error in discovery.server
	discoServer.Shutdown()
}

func TestDNSSDScan(t *testing.T) {

	records, err := discovery2.DnsSDScan("", time.Second*2, false)
	fmt.Printf("Found %d records in scan", len(records))

	assert.NoError(t, err)
	assert.Greater(t, len(records), 0, "No DNS records found")
}
