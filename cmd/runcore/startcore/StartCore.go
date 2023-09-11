package startcore

import (
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/core/auth/authservice"
	"github.com/hiveot/hub/core/config"
	"github.com/hiveot/hub/core/mqttmsgserver"
	"github.com/hiveot/hub/core/natsmsgserver"
	"time"
)

// Runtimes
var MsgServer msgserver.IMsgServer
var AuthService *authservice.AuthService

// Start starts the core services
// This starts:
// 1. the nats or mqtt core depending on the config
// 2. the auth service
// 3. the certs service
// 4. the launcher service
func Start(cfg *config.HubCoreConfig) (clientURL string, err error) {

	if cfg.Core == "nats" || cfg.Core == "" {
		// start the embedded NATS messaging MsgServer
		if !cfg.NatsServer.NoAutoStart {
			MsgServer = natsmsgserver.NewNatsMsgServer(&cfg.NatsServer, auth.DefaultRolePermissions)
			clientURL, err = MsgServer.Start()
			if err != nil {
				return "", err
			}
		}
	} else if cfg.Core == "mqtt" {
		if !cfg.MqttServer.NoAutoStart {
			MsgServer = mqttmsgserver.NewMqttMsgServer(&cfg.MqttServer, auth.DefaultRolePermissions)
			clientURL, err = MsgServer.Start()
			if err != nil {
				return "", err
			}
		}
	} else {
		return "", fmt.Errorf("unknown core: %s", cfg.Core)
	}

	// start the auth service service
	if !cfg.Auth.NoAutoStart {
		// nats requires brcypt passwords
		cfg.Auth.Encryption = auth.PWHASH_BCRYPT
		AuthService, err = authservice.StartAuthService(cfg.Auth, MsgServer)
		if err != nil {
			return clientURL, err
		}
	}
	return clientURL, nil
}

// Stop stops the Hub core services
func Stop() {
	if AuthService != nil {
		AuthService.Stop()
		AuthService = nil
	}

	if MsgServer != nil {
		MsgServer.Stop()
		MsgServer = nil
	}
	// give background tasks time to stop
	time.Sleep(time.Millisecond * 10)
}
