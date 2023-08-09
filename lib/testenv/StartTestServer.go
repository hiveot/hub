package testenv

import (
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/core/msgserver/natsserver"
)

func StartTestServer(storeDir string, authBundle *TestAuthBundle) (
	clientURL string, server *natsserver.HubNatsServer, err error) {

	// run the test server
	serverCfg := &msgserver.MsgServerConfig{
		Host: "127.0.0.1",
		Port: 9990,
		//AppAccountName: authBundle.AppAccountName,
	}
	serverCfg.InitConfig("", storeDir)
	hubNatsServer := natsserver.NewHubNatsServer(
		authBundle.ServerCert, authBundle.CaCert)

	serverOpts := hubNatsServer.CreateServerConfig(
		serverCfg,
		authBundle.OperatorJWT,
		authBundle.SystemAccountJWT,
		authBundle.AppAccountJWT,
		authBundle.ServiceKey)
	serverOpts.Debug = true
	clientURL, err = hubNatsServer.Start(serverOpts)
	return clientURL, hubNatsServer, err
}
