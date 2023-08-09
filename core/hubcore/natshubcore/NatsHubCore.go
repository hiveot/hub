package natshubcore

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	authn2 "github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/core/authn/authnservice"
	"github.com/hiveot/hub/core/authn/authnstore"
	"github.com/hiveot/hub/core/authn/natsauthn"
	"github.com/hiveot/hub/core/authz/authzservice"
	"github.com/hiveot/hub/core/authz/natsauthz"
	"github.com/hiveot/hub/core/config/natsconfig"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/hiveot/hub/core/server/natsserver"
	"github.com/nats-io/nkeys"
	"path"
)

// HubCore with core services for authnBinding, authz, directory and history
type HubCore struct {
	config *natsconfig.HubNatsConfig

	// Server keys and certs. These are readonly
	//AppAcctName string
	//AppAcctKey  nkeys.KeyPair
	CaCert        *x509.Certificate
	CaKey         *ecdsa.PrivateKey
	ServerCert    *tls.Certificate
	OperatorKey   nkeys.KeyPair
	OperatorJWT   string
	SystemJWT     string
	AppAccountKey nkeys.KeyPair
	AppAccountJWT string
	ServiceKey    nkeys.KeyPair
	ServiceJWT    string

	// Server runtime
	Server *natsserver.HubNatsServer

	// authn runtime
	//authnBinding *AuthnServiceBinding
	AuthnSvc *authnservice.AuthnService

	// authz runtime
	authzStore *authzservice.AclFileStore
	//authzJetStream *natsauthz.NatsAuthzAppl
	//authzMsgBinding *authzservice.AuthzServiceBinding
	AuthzSvc *authzservice.AuthzService
}

// Start the Hub messaging Server and core services
// This runs setup(false) to ensure the core has all it needs
// This panics if anything goes wrong
func (core *HubCore) Start() (clientURL string) {
	var err error
	cfg := core.config

	core.ServerCert, core.CaCert, core.CaKey,
		core.OperatorKey, core.OperatorJWT, core.SystemJWT,
		core.AppAccountKey, core.AppAccountJWT,
		core.ServiceKey, core.ServiceJWT = core.config.Setup(false)

	// start the embedded NATS messaging Server
	if !cfg.Server.NoAutoStart {
		//// nats server configurator handles proper server config settings
		//natsConfigurator := natsserver.NewNatsConfigurator(
		//	&cfg.Server, core.ServerCert, core.CaCert,
		//	core.OperatorJWT, core.SystemJWT, core.AppAccountJWT, core.ServiceKey)

		core.Server = natsserver.NewHubNatsServer(core.ServerCert, core.CaCert)

		// nats server configurator handles proper server config settings
		natsOpts := core.Server.CreateServerConfig(&cfg.Server,
			core.OperatorJWT, core.SystemJWT, core.AppAccountJWT, core.ServiceKey)

		clientURL, err = core.Server.Start(natsOpts)
		if err != nil {
			panic(err.Error())
		}
	}

	// start the authnBinding service
	if !cfg.Authn.NoAutoStart {
		authnStore := authnstore.NewAuthnFileStore(core.config.Authn.PasswordFile)
		tokenizer := natsauthn.NewAuthnNatsTokenizer(core.AppAccountKey)
		nc, err := core.Server.ConnectInProc(core.ServiceJWT, core.ServiceKey)
		if err != nil {
			panic(err.Error())
		}
		hc := natshubclient.NewHubClient(core.ServiceKey)
		hc.ConnectWithNC(nc, authn2.AuthnServiceName)
		core.AuthnSvc = authnservice.NewAuthnService(authnStore, tokenizer, hc)

		err = core.AuthnSvc.Start()
		if err != nil {
			panic(err.Error())
		}
	}
	// start the authz service
	if !cfg.Authz.NoAutoStart {
		// AuthzFileStore stores passwords in file
		authzFile := path.Join(cfg.Authz.GroupsDir, authz.DefaultAclFilename)
		core.authzStore = authzservice.NewAuthzFileStore(authzFile)
		err = core.authzStore.Open()
		if err != nil {
			panic("Failed to open the authz store: " + err.Error())
		}
		// establish another service connection
		nc, err := core.Server.ConnectInProc(core.ServiceJWT, core.ServiceKey)
		if err != nil {
			panic(err.Error())
		}
		hc := natshubclient.NewHubClient(core.ServiceKey)
		hc.ConnectWithNC(nc, authz.AuthzServiceName)
		// apply authz changes to nats jetstream
		authzJetStream := natsauthz.NewNatsAuthzAppl(hc.JS())
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
func NewHubCore(config *natsconfig.HubNatsConfig) *HubCore {

	hs := &HubCore{config: config}
	return hs
}
