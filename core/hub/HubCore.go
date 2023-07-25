package hub

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authn/service"
	"github.com/hiveot/hub/core/authn/service/unpwstore"
	"github.com/hiveot/hub/core/authz"
	service2 "github.com/hiveot/hub/core/authz/service"
	"github.com/hiveot/hub/core/config"
	"github.com/hiveot/hub/core/server"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/nats-io/nkeys"
	"path"
)

// HubCore with core services for authnBinding, authz, directory and history
type HubCore struct {
	config *config.HubCoreConfig

	// Server keys and certs. These are readonly
	AppAcctName string
	AppAcctKey  nkeys.KeyPair
	CaCert      *x509.Certificate
	CaKey       *ecdsa.PrivateKey
	ServerCert  *tls.Certificate

	// Server runtime
	Server *server.HubNatsServer

	// authn runtime
	authnBinding *service.AuthnMsgBinding
	AuthnSvc     *service.AuthnService

	// authz runtime
	authzStore      *service2.AclFileStore
	authzJetStream  *service2.AuthzJetStream
	authzMsgBinding *service2.AuthzMsgBinding
	AuthzSvc        *service2.AuthzService
}

// Start the Hub messaging Server and core services
// This runs setup(false) to ensure the core has all it needs
// This panics if anything goes wrong
func (core *HubCore) Start() (clientURL string) {
	var err error
	cfg := core.config

	core.AppAcctKey, core.ServerCert, core.CaCert, core.CaKey =
		core.config.Setup(false)
	core.AppAcctName = core.config.Server.AppAccountName

	// start the embedded NATS messaging Server
	if !cfg.Server.NoAutoStart {
		core.Server = server.NewHubNatsServer(
			&cfg.Server, core.AppAcctKey, core.ServerCert, core.CaCert)
		clientURL, err = core.Server.Start()
		if err != nil {
			panic(err.Error())
		}
	}

	// start the authnBinding service
	if !cfg.Authn.NoAutoStart {
		pwStore := unpwstore.NewPasswordFileStore(core.config.Authn.PasswordFile)
		core.AuthnSvc = service.NewAuthnService(
			pwStore,
			core.config.Server.AppAccountName,
			core.AppAcctKey,
			core.CaCert)

		err = core.AuthnSvc.Start()
		if err != nil {
			panic(err.Error())
		}
		// use an adhoc nkey to connect to the nats Server
		authnNKey, _ := nkeys.CreateUser()
		err = core.Server.AddAppAcctServiceKey(authnNKey)
		if err != nil {
			panic(err.Error())
		}
		nc, err := core.Server.ConnectInProc(authn.AuthnServiceName, authnNKey)
		if err != nil {
			panic(err.Error())
		}
		hc := hubclient.NewHubClient()
		hc.ConnectWithNC(nc, authn.AuthnServiceName)
		// AuthnMsgBinding connects to the message bus and (un)marshals messages
		core.authnBinding = service.NewAuthnMsgBinding(core.AuthnSvc)
		err = core.authnBinding.Start(hc)
		if err != nil {
			panic(err.Error())
		}

		// Hook into the nats service callout authentication
		authnVerifier := service.NewAuthnNatsVerify(core.AuthnSvc)
		core.Server.SetAuthnVerifier(authnVerifier.VerifyAuthnReq)
	}
	// start the authz service
	if !cfg.Authz.NoAutoStart {
		// AuthzFileStore stores passwords in file
		authzFile := path.Join(cfg.Authz.GroupsDir, authz.DefaultAclFilename)
		core.authzStore = service2.NewAuthzFileStore(authzFile)
		err = core.authzStore.Open()
		if err != nil {
			panic("Failed to open the authz store: " + err.Error())
		}
		// AuthzJetStream applies groups to nats jetstream using an adhoc service connection
		authzNKey, _ := nkeys.CreateUser()
		err = core.Server.AddAppAcctServiceKey(authzNKey)
		if err != nil {
			panic(err.Error())
		}
		nc, err := core.Server.ConnectInProc(authz.AuthzServiceName, authzNKey)
		if err != nil {
			panic("Failed to open the connection to the nats Server: " + err.Error())
		}
		core.authzJetStream = service2.NewAuthzJetStream(nc)
		// The service forwards requests to the store and jetstream
		core.AuthzSvc = service2.NewAuthzService(core.authzStore, core.authzJetStream)
		// AuthzMsgBinding connects authz to the message bus and (un)marshals messages
		hc := hubclient.NewHubClient()
		hc.ConnectWithNC(nc, authz.AuthzServiceName)
		core.authzMsgBinding = service2.NewAuthzMsgBinding(core.AuthzSvc, hc)
		err = core.authzMsgBinding.Start()
		if err != nil {
			panic("Unable to bind to the messaging Server: " + err.Error())
		}
	}
	return clientURL
}

// Stop the Server
func (core *HubCore) Stop() {
	if core.authnBinding != nil {
		core.authnBinding.Stop()
	}
	if core.authzMsgBinding != nil {
		core.authzMsgBinding.Stop()
	}
	if core.authzJetStream != nil {
		core.authzJetStream.Stop()
	}
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
func NewHubCore(config *config.HubCoreConfig) *HubCore {

	hs := &HubCore{config: config}
	return hs
}
