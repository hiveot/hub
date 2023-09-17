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
func StartTestServer(core string) (
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
		}
	}
	return serverURL, msgServer, certBundle, err
}
