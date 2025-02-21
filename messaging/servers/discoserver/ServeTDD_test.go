package discoserver_test

import (
	"fmt"
	"github.com/hiveot/hub/messaging/clients/discovery"
	"github.com/hiveot/hub/messaging/servers/discoserver"
	"github.com/hiveot/hub/messaging/tputils/net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serviceID is the service publishing the record, thing or directory
const testServiceID = "hiveot-test"

// the service name is normally 'wot'.
const testServiceName = "wot-test"
const testServicePath = "/discovery/path"
const testServicePort = 9999

func TestDNSSDScan(t *testing.T) {

	records, err := discovery.DnsSDScan("", "", time.Second*2, false)
	fmt.Printf("Found %d records in scan", len(records))

	assert.NoError(t, err)
	assert.Greater(t, len(records), 0, "No DNS records found")
}

// Test the discovery client and server
func TestDiscover(t *testing.T) {
	testServiceAddress := net.GetOutboundIP("").String()
	endpoints := map[string]string{"wss": "wss://localhost/wssendpoint"}

	tddURL := fmt.Sprintf("https://%s:%d%s", testServiceAddress, testServicePort, testServicePath)
	discoServer, err := discoserver.ServeTDDiscovery(
		testServiceID, testServiceName, tddURL, endpoints)

	assert.NoError(t, err)
	assert.NotNil(t, discoServer)

	// Test if it is discovered
	records := discovery.DiscoverTDD(testServiceID, testServiceName, time.Second, false)
	require.NotEmpty(t, records)
	rec0 := records[0]
	assert.Equal(t, testServiceID, rec0.Instance)
	assert.Equal(t, testServiceAddress, rec0.Addr)
	assert.Equal(t, testServicePath, rec0.TD)
	assert.Equal(t, true, rec0.IsDirectory)

	time.Sleep(time.Millisecond) // prevent race error in discovery.server
	discoServer.Shutdown()
}

func TestDiscoViaDomainName(t *testing.T) {
	testServiceAddress := "localhost"

	discoServer, err := discoserver.ServeDnsSD(
		testServiceID, testServiceName, "",
		testServiceAddress, testServicePort, nil)

	assert.NoError(t, err)
	assert.NotNil(t, discoServer)

	// Test if it is discovered
	records := discovery.DiscoverTDD("", testServiceName, time.Second, true)
	require.NoError(t, err)
	require.True(t, len(records) > 0)
	rec0 := records[0]
	assert.Equal(t, "127.0.0.1", rec0.Addr)
	//assert.True(t, strings.HasPrefix(rec0.HostName, testServiceAddress))
	//assert.Equal(t, testServicePort, discoPort)

	time.Sleep(time.Millisecond) // prevent race error in discovery.server
	discoServer.Shutdown()
}

func TestDiscoverBadPort(t *testing.T) {
	serviceID := "idprov-test"
	badPort := 0
	address := net.GetOutboundIP("").String()
	_, err := discoserver.ServeDnsSD(
		serviceID, testServiceName, "", address, badPort, nil)

	assert.Error(t, err)
}

func TestNoInstanceID(t *testing.T) {
	serviceID := "serviceID"
	address := net.GetOutboundIP("").String()

	_, err := discoserver.ServeDnsSD(
		"", testServiceName, "",
		address, testServicePort, nil)
	assert.Error(t, err) // missing instance name

	_, err = discoserver.ServeDnsSD(
		serviceID, "", "", address, testServicePort, nil)
	assert.Error(t, err) // missing service name
}

func TestDiscoverNotFound(t *testing.T) {
	instanceID := "idprov-test-id"
	serviceName := "idprov-test"
	address := net.GetOutboundIP("").String()

	discoServer, err := discoserver.ServeDnsSD(
		instanceID, serviceName, "",
		address, testServicePort, nil)

	assert.NoError(t, err)

	// Test if it is discovered
	records := discovery.DiscoverTDD("", testServiceName, time.Second, true)
	_ = records
	assert.Equal(t, 0, len(records))

	time.Sleep(time.Millisecond) // prevent race error in discovery.server
	discoServer.Shutdown()
}

func TestBadAddress(t *testing.T) {
	instanceID := "idprov-test-id"

	discoServer, err := discoserver.ServeDnsSD(
		instanceID, testServiceName, "", "notanipaddress", testServicePort, nil)

	assert.Error(t, err)
	assert.Nil(t, discoServer)
}

func TestExternalAddress(t *testing.T) {
	instanceID := "idprov-test-id"

	discoServer, err := discoserver.ServeDnsSD(
		instanceID, testServiceName, "", "1.2.3.4", testServicePort, nil)

	// expect a warning
	assert.NoError(t, err)
	time.Sleep(time.Millisecond) // prevent race error in discovery.server
	discoServer.Shutdown()
}
