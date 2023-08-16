package testenv

import (
	"github.com/hiveot/hub/core/msgserver/natsserver"
	"github.com/hiveot/hub/lib/certs"
	"os"
	"path"
)

// StartNatsTestServer generate a test configuration and starts a NKeys based nats test server
// A new temporary storage directory is used.
func StartNatsTestServer() (
	clientURL string, server *natsserver.NatsNKeyServer, certBundle certs.TestCertBundle, config *natsserver.NatsServerConfig, err error) {

	certBundle = certs.CreateTestCertBundle()
	hubNatsServer := natsserver.NewNatsNKeyServer()

	serverCfg := &natsserver.NatsServerConfig{
		Port:   9990,
		CaCert: certBundle.CaCert,
		CaKey:  certBundle.CaKey,
		//ServerCert: certBundle.ServerCert, // auto generate
		Debug: true,
	}
	//
	tmpDir := path.Join(os.TempDir(), "nats-testserver")
	_ = os.RemoveAll(tmpDir)
	err = serverCfg.Setup("", tmpDir, false)
	if err == nil {
		clientURL, err = hubNatsServer.Start(serverCfg)
	}
	return clientURL, hubNatsServer, certBundle, serverCfg, err
}
