package testenv

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/core/natsmsgserver"
	"github.com/hiveot/hub/core/natsmsgserver/callouthook"
	"github.com/hiveot/hub/core/natsmsgserver/service"
	"github.com/hiveot/hub/lib/certs"
	"os"
	"path"
)

// StartNatsTestServer generate a test configuration and starts a NKeys based nats test server
// A new temporary storage directory is used.
func StartNatsTestServer(withCallout bool) (
	clientURL string,
	hubNatsServer *service.NatsMsgServer,
	certBundle certs.TestCertBundle,
	config *natsmsgserver.NatsServerConfig, err error) {

	certBundle = certs.CreateTestCertBundle()

	serverCfg := &natsmsgserver.NatsServerConfig{
		Port:   9990,
		CaCert: certBundle.CaCert,
		CaKey:  certBundle.CaKey,
		//ServerCert: certBundle.ServerCert, // auto generate
		//Debug: true,
	}
	//
	tmpDir := path.Join(os.TempDir(), "nats-testserver")
	_ = os.RemoveAll(tmpDir)
	err = serverCfg.Setup(tmpDir, tmpDir, false)
	if err == nil {
		hubNatsServer = service.NewNatsMsgServer(serverCfg, auth.DefaultRolePermissions)
		clientURL, err = hubNatsServer.Start()
	}
	if err == nil && withCallout {

		// use the callout server to enable for JWT
		_, err = callouthook.EnableNatsCalloutHook(hubNatsServer)
		if err != nil {
			panic(err)
		}
	}
	return clientURL, hubNatsServer, certBundle, serverCfg, err
}
