package testenv

import (
	"github.com/hiveot/hub/core/msgserver/natsserver"
	"github.com/hiveot/hub/lib/certs"
)

// StartNatsTestServer generate a test configuration and starts a NKeys based nats test server
func StartNatsTestServer() (
	clientURL string, server *natsserver.NatsNKeyServer, certBundle certs.TestCertBundle, err error) {

	certBundle = certs.CreateTestCertBundle()
	hubNatsServer := natsserver.NewNatsNKeyServer(
		certBundle.ServerCert, certBundle.CaCert)

	appCfg := natsserver.NatsNKeysConfig{
		Port:       9990,
		CaCert:     certBundle.CaCert,
		ServerCert: certBundle.ServerCert,
		Debug:      true,
	}
	clientURL, err = hubNatsServer.Start(appCfg)
	return clientURL, hubNatsServer, certBundle, err
}
