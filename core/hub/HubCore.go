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
	"github.com/hiveot/hub/core/hubnats"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/nats-io/nkeys"
	"path"
)

// HubCore with core services for authnBinding, authz, directory and history
type HubCore struct {
	config *config.HubCoreConfig

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
	Server *hubnats.HubNatsServer

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

	core.ServerCert, core.CaCert, core.CaKey,
		core.OperatorKey, core.OperatorJWT, core.SystemJWT,
		core.AppAccountKey, core.AppAccountJWT,
		core.ServiceKey, core.ServiceJWT = core.config.Setup(false)

	// start the embedded NATS messaging Server
	if !cfg.Server.NoAutoStart {
		// nats server configurator handles proper server config settings
		natsConfigurator := hubnats.NewNatsConfigurator(
			&cfg.Server, core.ServerCert, core.CaCert,
			core.OperatorJWT, core.SystemJWT, core.AppAccountJWT, core.ServiceKey)

		core.Server = hubnats.NewHubNatsServer(
			core.ServerCert, core.CaCert, core.ServiceKey, core.ServiceJWT)

		clientURL, err = core.Server.Start(natsConfigurator.GetServerOpts())
		if err != nil {
			panic(err.Error())
		}
	}

	// start the authnBinding service
	if !cfg.Authn.NoAutoStart {
		pwStore := unpwstore.NewPasswordFileStore(core.config.Authn.PasswordFile)
		core.AuthnSvc = service.NewAuthnService(
			pwStore, core.CaCert, core.AppAccountKey)

		err = core.AuthnSvc.Start()
		if err != nil {
			panic(err.Error())
		}
		// use an adhoc nkey to connect to the nats Server
		//authnServiceKey, _ := nkeys.CreateUser()
		//authnServiceKeyPub, _ := authnServiceKey.PublicKey()
		//err = core.Server.AddServiceKey(authnServiceKeyPub)
		//if err != nil {
		//	panic(err.Error())
		//}
		//nc, err := core.Server.ConnectInProc(authn.AuthnServiceName, authnServiceKey)
		nc, err := core.Server.ConnectInProc(authn.AuthnServiceName)
		if err != nil {
			panic(err.Error())
		}
		hc := hubclient.NewHubClient()
		hc.ConnectWithNC(nc, authn.AuthnServiceName)
		// AuthnMsgBinding connects to the message bus and (un)marshals messages
		core.authnBinding = service.NewAuthnMsgBinding(core.AuthnSvc, hc)
		err = core.authnBinding.Start()
		if err != nil {
			panic(err.Error())
		}

		// Hook into the nats service callout authentication
		//authnVerifier := service.NewAuthnNatsVerify(core.AuthnSvc)
		//core.Server.InitCalloutHook(authnVerifier.VerifyAuthnReq)
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
		//authzNKey, _ := nkeys.CreateUser()
		//authzNKeyPub, _ := authzNKey.PublicKey()
		//err = core.Server.AddServiceKey(authzNKeyPub)
		//if err != nil {
		//	panic(err.Error())
		//}
		nc, err := core.Server.ConnectInProc(authz.AuthzServiceName)
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
