package hub

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authn/service"
	"github.com/hiveot/hub/core/authn/service/unpwstore"
	"github.com/hiveot/hub/core/authz"
	service2 "github.com/hiveot/hub/core/authz/service"
	"github.com/hiveot/hub/core/config"
	"github.com/hiveot/hub/core/server"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/nats-io/nkeys"
	"os"
)

// HubCore with core services for authn, authz, directory and history
type HubCore struct {
	config *config.HubCoreConfig
	//serverCert *tls.Certificate
	//caCert     *x509.Certificate
	server *server.HubNatsServer
	authn  *service.AuthnBinding
	authz  authz.IAuthz
	//dir        directory.IDirectory
	//hist       history.IHistory
	appAcctKey nkeys.KeyPair
	caCert     *x509.Certificate
	caKey      *ecdsa.PrivateKey
	serverCert *tls.Certificate
}

// Start the Hub messaging server and core services
func (core *HubCore) Start() (clientURL string, err error) {
	cfg := core.config

	// load the certs and keys
	core.caCert, err = certs.LoadX509CertFromPEM(core.config.Server.CaCertFile)
	if err == nil {
		core.caKey, err = certs.LoadKeysFromPEM(core.config.Server.CaKeyFile)
	}
	if err == nil {
		core.serverCert, err = certs.LoadTLSCertFromPEM(
			core.config.Server.ServerCertFile, core.config.Server.ServerKeyFile)
	}
	if err == nil {
		appAcctSeed, err2 := os.ReadFile(core.config.Server.AppAccountKeyFile)
		if err = err2; err != nil {
			core.appAcctKey, err = nkeys.FromSeed(appAcctSeed)
			// tbd do we need jwt claims on this key? if so, use cred
			//core.appAcctKey, err = nkeys.ParseDecoratedNKey(appAcctCred)

		}
	}
	if err != nil {
		return "", fmt.Errorf("unable to load certificates and keys: %w", err)
	}

	// start the messaging service
	if !cfg.Server.NoAutoStart {
		core.server = server.NewHubNatsServer(
			&cfg.Server, core.appAcctKey, core.serverCert, core.caCert)
		clientURL, err = core.server.Start()
		if err != nil {
			return
		}
	}

	// start the authn service
	if !cfg.Authn.NoAutoStart {
		pwStore := unpwstore.NewPasswordFileStore(core.config.Authn.PasswordFile)
		authnService := service.NewAuthnService(
			pwStore,
			core.config.Server.AppAccountName,
			core.appAcctKey,
			core.caCert)

		// connect to the nats server to handle requests
		// use an adhoc nkey to connect
		authnNKey, _ := nkeys.CreateUser()
		_ = core.server.AddServiceKey(authnNKey)
		nc, err := core.server.ConnectInProc(authn.AuthnServiceName, authnNKey)
		if err != nil {
			return "", fmt.Errorf("failed starting authn service: %w", err)
		}
		hc := hubclient.NewHubClient()
		_ = hc.ConnectWithNC(nc, authn.AuthnServiceName)
		core.authn = service.NewAuthnNatsBinding(authnService)
		core.authn.Start(hc)

		// start the callout authentication handler
		authnVerifier := service.NewAuthnNatsVerify(authnService)
		core.server.SetAuthnVerifier(authnVerifier.VerifyAuthnReq)
	}
	// start the authz service
	if !cfg.Authz.NoAutoStart {
		authz := service2.NewAuthzService(cfg.Authz.GroupsDir)
		err = authz.Start()
		if err != nil {
			return clientURL, fmt.Errorf("failed starting authz service: %w", err)
		}
	}
	return clientURL, nil
}

// Stop the server
func (core *HubCore) Stop() {
	core.authn.Stop()
	core.server.Stop()
}

// NewHubCore creates the hub core instance.
// This creates the messaging server and core services.
// config must have been loaded
func NewHubCore(config *config.HubCoreConfig) *HubCore {

	hs := &HubCore{config: config}
	return hs
}
