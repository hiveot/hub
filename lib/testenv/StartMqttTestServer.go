package testenv

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/core/mqttmsgserver"
	"github.com/hiveot/hub/core/mqttmsgserver/service"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubcl/mqtthubclient"
	"os"
	"path"
)

// StartMqttTestServer creates a MQTT test server.
// applyTestClients registers the test clients with the server for direct usage.
//
// Note that test clients will be replaced when the 'auth' core service is started,
// so only use it instead of 'auth'.
func StartMqttTestServer(applyTestClients bool) (
	serverURL string,
	msgServer msgserver.IMsgServer,
	certBundle certs.TestCertBundle,
	err error) {

	certBundle = certs.CreateTestCertBundle()

	serverCfg := &mqttmsgserver.MqttServerConfig{
		Host:          "",
		Port:          9990,
		WSPort:        9991,
		CaCert:        certBundle.CaCert,
		CaKey:         certBundle.CaKey,
		Debug:         true,
		InProcUDSName: mqtthubclient.MqttInMemUDSTest,
	}
	tmpDir := path.Join(os.TempDir(), "mqtt-testserver")
	_ = os.RemoveAll(tmpDir)
	err = serverCfg.Setup(tmpDir, tmpDir, false)
	if err == nil {
		mqttServer := service.NewMqttMsgServer(serverCfg, auth.DefaultRolePermissions)
		serverURL, err = mqttServer.Start()
		msgServer = mqttServer
		if applyTestClients {
			_ = mqttServer.ApplyAuth(MqttTestClients)
		}
	}
	return serverURL, msgServer, certBundle, err
}
