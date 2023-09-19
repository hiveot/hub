package testenv

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/core/mqttmsgserver"
	"github.com/hiveot/hub/core/natsmsgserver"
	"github.com/hiveot/hub/lib/certs"
	"os"
	"path"
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
			natsServer := natsmsgserver.NewNatsMsgServer(serverCfg, auth.DefaultRolePermissions)
			serverURL, err = natsServer.Start()
			msgServer = natsServer
			if applyTestClients {
				_ = natsServer.ApplyAuth(NatsTestClients)
			}
		}
	} else {
		serverCfg := &mqttmsgserver.MqttServerConfig{
			Host:   "",
			Port:   9990,
			CaCert: certBundle.CaCert,
			CaKey:  certBundle.CaKey,
			Debug:  true,
		}
		tmpDir := path.Join(os.TempDir(), "mqtt-testserver")
		_ = os.RemoveAll(tmpDir)
		err = serverCfg.Setup(tmpDir, tmpDir, false)
		if err == nil {
			mqttServer := mqttmsgserver.NewMqttMsgServer(serverCfg, auth.DefaultRolePermissions)
			serverURL, err = mqttServer.Start()
			msgServer = mqttServer
			if applyTestClients {
				_ = mqttServer.ApplyAuth(MqttTestClients)
			}
		}
	}
	return serverURL, msgServer, certBundle, err
}
