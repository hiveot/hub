package hub

import (
	"github.com/hiveot/hub/core/authn/service"
	"github.com/hiveot/hub/core/authn/service/unpwstore"
	"github.com/hiveot/hub/core/authz"
	"github.com/hiveot/hub/core/config"
	"github.com/hiveot/hub/core/server"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/nats-io/nkeys"
)

// HubCore with core services for authn, authz, directory and history
type HubCore struct {
	config config.HubCoreConfig
	//serverCert *tls.Certificate
	//caCert     *x509.Certificate
	server *server.HubNatsServer
	authn  *service.AuthnBinding
	authz  authz.IAuthz
	//dir        directory.IDirectory
	//hist       history.IHistory
}

// Start the Hub messaging server and core services
func (core *HubCore) Start() (clientURL string, err error) {

	// start the messaging service
	core.server = server.NewHubNatsServer(core.config.Server)
	clientURL, err = core.server.Start()
	if err != nil {
		return
	}

	// start the authn service
	pwStore := unpwstore.NewPasswordFileStore(core.config.Authn.PasswordFile)
	authnService := service.NewAuthnService(
		core.config.Server.AppAccountName,
		core.config.Server.AppAccountKey,
		pwStore,
		core.config.Server.CaCert)

	// connect to the nats server to handle requests
	// use an adhoc nkey to connect
	authnNKey, _ := nkeys.CreateUser()
	_ = core.server.AddServiceKey(authnNKey)
	nc, err := core.server.ConnectInProc(
		core.config.Authn.ServiceID, authnNKey)
	if err != nil {
		return "", err
	}
	hc := hubclient.NewHubClient()
	_ = hc.ConnectWithNC(nc, core.config.Authn.ServiceID)
	core.authn = service.NewAuthnNatsBinding(authnService)
	core.authn.Start(hc)

	// start the callout authentication handler
	authnVerifier := service.NewAuthnNatsVerify(authnService)
	core.server.SetAuthnVerifier(authnVerifier.VerifyAuthnReq)

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
func NewHubCore(config config.HubCoreConfig) *HubCore {

	hs := &HubCore{config: config}
	return hs
}
