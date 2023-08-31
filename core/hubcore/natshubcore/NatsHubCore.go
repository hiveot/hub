package natshubcore

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/core/auth/authservice"
	"github.com/hiveot/hub/core/auth/authstore"
	"github.com/hiveot/hub/core/config"
	"github.com/hiveot/hub/core/msgserver/natsnkeyserver"
)

// HubCore with core services for authnBinding, authz, directory and history
type HubCore struct {
	config *config.HubCoreConfig

	// Runtimes
	MsgServer   *natsnkeyserver.NatsNKeyServer
	AuthService *authservice.AuthService
}

// Start the Hub messaging MsgServer and core services with the given config
// This panics if anything goes wrong
func (core *HubCore) Start(cfg *config.HubCoreConfig) (clientURL string) {
	var err error
	core.config = cfg

	// start the embedded NATS messaging MsgServer
	if !cfg.NatsServer.NoAutoStart {
		//// nats server configurator handles proper server config settings
		//natsConfigurator := natsserver.NewNatsConfigurator(
		//	&cfg.MsgServer, core.ServerCert, core.CaCert,
		//	core.OperatorJWT, core.SystemJWT, core.AppAccountJWT, core.ServiceKey)

		core.MsgServer = natsnkeyserver.NewNatsNKeyServer(&core.config.NatsServer)
		clientURL, err = core.MsgServer.Start()
		if err != nil {
			panic(err.Error())
		}
	}

	// start the authn store, service and binding
	if !cfg.Auth.NoAutoStart {
		// nats requires brcypt passwords
		authStore := authstore.NewAuthnFileStore(cfg.Auth.PasswordFile, auth.PWHASH_BCRYPT)
		core.AuthService = authservice.NewAuthnService(authStore, core.MsgServer)

		err = core.AuthService.Start()
		if err != nil {
			panic(err.Error())
		}
	}
	return clientURL
}

// Stop the MsgServer
func (core *HubCore) Stop() {
	if core.AuthService != nil {
		core.AuthService.Stop()
	}

	if core.MsgServer != nil {
		core.MsgServer.Stop()
	}
}

// NewHubCore creates the hub core instance.
// This creates the messaging MsgServer and core services.
// config must have been loaded
func NewHubCore() *HubCore {
	hs := &HubCore{}
	return hs
}
