package hubcore

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/core/auth/authservice"
	"github.com/hiveot/hub/core/config"
	"github.com/hiveot/hub/core/msgserver/natsnkeyserver"
)

// Runtimes
var msgServer *natsnkeyserver.NatsNKeyServer
var authService *authservice.AuthService

// StartCore starts the Hub messaging core services with the given config.
func StartCore(cfg *config.HubCoreConfig) (clientURL string, err error) {

	// start the embedded NATS messaging MsgServer
	if !cfg.NatsServer.NoAutoStart {
		msgServer = natsnkeyserver.NewNatsNKeyServer(&cfg.NatsServer)
		clientURL, err = msgServer.Start()
		if err != nil {
			return "", err
		}
	}

	// start the authn store, service and binding
	if !cfg.Auth.NoAutoStart {
		// nats requires brcypt passwords
		cfg.Auth.Encryption = auth.PWHASH_BCRYPT
		authService, err = authservice.StartAuthService(cfg.Auth, msgServer)
		if err != nil {
			return clientURL, err
		}
	}
	return clientURL, nil
}

// StopCore stops the Hub core services
func StopCore() {
	if authService != nil {
		authService.Stop()
	}

	if msgServer != nil {
		msgServer.Stop()
	}
}
