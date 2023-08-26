package natshubcore

import (
	authn2 "github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/core/authn/authnservice"
	"github.com/hiveot/hub/core/authn/authnstore"
	"github.com/hiveot/hub/core/authz/authzservice"
	"github.com/hiveot/hub/core/config"
	"github.com/hiveot/hub/core/msgserver/natsnkeyserver"
	"path"
)

// HubCore with core services for authnBinding, authz, directory and history
type HubCore struct {
	config *config.HubCoreConfig

	// Runtimes
	Server     *natsnkeyserver.NatsNKeyServer
	AuthnSvc   *authnservice.AuthnService
	authzStore *authzservice.AclFileStore

	//authzJetStream *natsauthz.NatsAuthzAppl
	//authzMsgBinding *authzservice.AuthzBinding
	AuthzSvc *authzservice.AuthzService
}

// Start the Hub messaging Server and core services with the given config
// This panics if anything goes wrong
func (core *HubCore) Start(cfg *config.HubCoreConfig) (clientURL string) {
	var err error
	core.config = cfg

	// start the embedded NATS messaging Server
	if !cfg.NatsServer.NoAutoStart {
		//// nats server configurator handles proper server config settings
		//natsConfigurator := natsserver.NewNatsConfigurator(
		//	&cfg.Server, core.ServerCert, core.CaCert,
		//	core.OperatorJWT, core.SystemJWT, core.AppAccountJWT, core.ServiceKey)

		core.Server = natsnkeyserver.NewNatsNKeyServer(&core.config.NatsServer)
		clientURL, err = core.Server.Start()
		if err != nil {
			panic(err.Error())
		}
	}

	// start the authn store, service and binding
	if !cfg.Authn.NoAutoStart {
		// nats requires brcypt passwords
		authnStore := authnstore.NewAuthnFileStore(cfg.Authn.PasswordFile, authn2.PWHASH_BCRYPT)
		core.AuthnSvc = authnservice.NewAuthnService(authnStore, core.Server)

		err = core.AuthnSvc.Start()
		if err != nil {
			panic(err.Error())
		}
	}
	// start the authz service
	if !cfg.Authz.NoAutoStart {
		// AuthzFileStore stores passwords in file
		authzFile := path.Join(cfg.Authz.DataDir, authz.DefaultAclFilename)
		core.authzStore = authzservice.NewAuthzFileStore(authzFile)
		err = core.authzStore.Open()
		if err != nil {
			panic("Failed to open the authz store: " + err.Error())
		}
		core.AuthzSvc = authzservice.NewAuthzService(core.authzStore, core.Server)
		err = core.AuthzSvc.Start()
		if err != nil {
			panic(err.Error())
		}
	}
	return clientURL
}

// Stop the Server
func (core *HubCore) Stop() {
	if core.AuthnSvc != nil {
		core.AuthnSvc.Stop()
	}
	if core.AuthzSvc != nil {
		core.AuthzSvc.Stop()
	}
	//if core.authzJetStream != nil {
	//	core.authzJetStream.Stop()
	//}
	if core.authzStore != nil {
		core.authzStore.Close()
	}
	if core.Server != nil {
		core.Server.Stop()
	}
}

// NewHubCore creates the hub core instance.
// This creates the messaging Server and core services.
// config must have been loaded
func NewHubCore() *HubCore {
	hs := &HubCore{}
	return hs
}
