package testenv

import (
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/lib/certs"
)

// StartTestServer creates a NATS or MQTT test server depending on the requested type
// core is either "nats", or "mqtt" (default)
// applyTestClients registers the test client with the server for direct usage.
//
// Note that test clients will be replaced when the 'auth' core service is started,
// so only use it instead of 'auth'.
func StartTestServer(core string, applyTestClients bool) (
	serverURL string,
	msgServer msgserver.IMsgServer,
	certBundle certs.TestCertBundle,
	err error) {

	certBundle = certs.CreateTestCertBundle()
	if core == "nats" {

		serverURL, msgServer, certBundle, _, err = StartNatsTestServer(applyTestClients, false)

	} else {
		serverURL, msgServer, certBundle, err = StartMqttTestServer(applyTestClients)

	}
	return serverURL, msgServer, certBundle, err
}
