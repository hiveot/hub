package hubcore

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/core/auth/authservice"
	"github.com/hiveot/hub/core/config"
	"github.com/hiveot/hub/core/msgserver/natsnkeyserver"
	"time"
)

// Runtimes
var MsgServer *natsnkeyserver.NatsNKeyServer
var AuthService *authservice.AuthService

// StartCore starts the Hub messaging core services with the given config.
func StartCore(cfg *config.HubCoreConfig) (clientURL string, err error) {

	// start the embedded NATS messaging MsgServer
	if !cfg.NatsServer.NoAutoStart {
		MsgServer = natsnkeyserver.NewNatsNKeyServer(&cfg.NatsServer)
		clientURL, err = MsgServer.Start()
		if err != nil {
			return "", err
		}
	}

	// start the authn store, service and binding
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

// StopCore stops the Hub core services
func StopCore() {
	if AuthService != nil {
		AuthService.Stop()
	}

	if MsgServer != nil {
		MsgServer.Stop()
	}
	// give background tasks time to stop
	time.Sleep(time.Millisecond * 10)
}
