package testenv

import (
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/core/auth/authservice"
	"github.com/hiveot/hub/lib/certs"
	"os"
	"path"
)

// StartTestServer creates a NATS or MQTT test server depending on the requested type
// core is either "nats", or "mqtt" (default)
//
//	startAuth also starts the auth service.
//	applyTestClients instead of using auth, registers test client accounts directly with the server.
//
// Note that test clients will be replaced when the 'auth' core service is started,
// so only use it instead of 'auth'.
func StartTestServer(core string, startAuth bool, applyTestClients bool) (
	serverURL string,
	msgServer msgserver.IMsgServer,
	certBundle certs.TestCertBundle,
	stopFn func(),
	err error) {

	var authSvc *authservice.AuthService
	certBundle = certs.CreateTestCertBundle()
	if core == "nats" {
		serverURL, msgServer, certBundle, _, err = StartNatsTestServer(applyTestClients, false)
	} else {
		serverURL, msgServer, certBundle, err = StartMqttTestServer(applyTestClients)
	}
	if startAuth {
		var testDir = path.Join(os.TempDir(), "test-authn")
		authConfig := auth.AuthConfig{}
		_ = authConfig.Setup(testDir, testDir)
		authSvc, err = authservice.StartAuthService(authConfig, msgServer)
	}
	return serverURL, msgServer, certBundle, func() {
		if authSvc != nil {
			authSvc.Stop()
		}
		if msgServer != nil {
			msgServer.Stop()
		}
	}, err
}
