package testenv

import (
	"github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/core/msgserver/mqttmsgserver"
	"github.com/hiveot/hub/core/msgserver/mqttmsgserver/service"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient/mqtthubclient"
	"os"
	"path"
)

// StartMqttTestServer creates a MQTT test server.
func StartMqttTestServer() (
	msgServer msgserver.IMsgServer,
	certBundle certs.TestCertBundle,
	err error) {

	certBundle = certs.CreateTestCertBundle()

	serverCfg := &mqttmsgserver.MqttServerConfig{
		Host:         "localhost",
		Port:         9990,
		WSPort:       9991,
		CaCert:       certBundle.CaCert,
		CaKey:        certBundle.CaKey,
		InMemUDSName: mqtthubclient.MqttInMemUDSTest,
		LogLevel:     "info",
	}
	tmpDir := path.Join(os.TempDir(), "mqtt-testserver")
	_ = os.RemoveAll(tmpDir)
	err = serverCfg.Setup(tmpDir, tmpDir, false)
	if err == nil {
		mqttServer := service.NewMqttMsgServer(serverCfg, auth.DefaultRolePermissions)
		err = mqttServer.Start()
		msgServer = mqttServer
	}
	return msgServer, certBundle, err
}
