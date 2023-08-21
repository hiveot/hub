package natshubcore

import (
	authn2 "github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/core/authn/authnservice"
	"github.com/hiveot/hub/core/authn/authnstore"
	"github.com/hiveot/hub/core/authz/authzadapter"
	"github.com/hiveot/hub/core/authz/authzservice"
	"github.com/hiveot/hub/core/config"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/hiveot/hub/core/msgserver/natsserver"
	"path"
)

// HubCore with core services for authnBinding, authz, directory and history
type HubCore struct {
	config *config.HubCoreConfig

	// Runtimes
	Server     *natsserver.NatsNKeyServer
	AuthnSvc   *authnservice.AuthnService
	authzStore *authzservice.AclFileStore

	//authzJetStream *natsauthz.NatsAuthzAppl
	//authzMsgBinding *authzservice.AuthzServiceBinding
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

		core.Server = natsserver.NewNatsNKeyServer()
		clientURL, err = core.Server.Start(&core.config.NatsServer)
		if err != nil {
			panic(err.Error())
		}
	}

	// start the authn store, service and binding
	if !cfg.Authn.NoAutoStart {
		authnStore := authnstore.NewAuthnFileStore(cfg.Authn.PasswordFile)
		tokenizer := natsserver.NewNatsAuthnTokenizer(cfg.NatsServer.AppAccountKP, true)
		nc, err := core.Server.ConnectInProc(authn2.AuthnServiceName, nil)
		if err != nil {
			panic(err.Error())
		}
		hc, err := natshubclient.ConnectWithNC(nc, authn2.AuthnServiceName)
		if err != nil {
			panic(err.Error())
		}
		core.AuthnSvc = authnservice.NewAuthnService(authnStore, core.Server, tokenizer, hc)

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
		// establish another service connection
		nc, err := core.Server.ConnectInProc(authz.AuthzServiceName, nil)
		if err != nil {
			panic(err.Error())
		}
		hc, err := natshubclient.ConnectWithNC(nc, authz.AuthzServiceName)
		if err != nil {
			panic(err.Error())
		}
		js := hc.JS()
		// apply authz changes to nats jetstream
		authzJetStream, err := authzadapter.NewNatsAuthzAdapter(js)
		core.AuthzSvc = authzservice.NewAuthzService(core.authzStore, authzJetStream, hc)
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
