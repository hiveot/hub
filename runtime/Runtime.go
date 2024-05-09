package runtime

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn/authnagent"
	"github.com/hiveot/hub/runtime/authn/service"
	"github.com/hiveot/hub/runtime/authz"
	"github.com/hiveot/hub/runtime/authz/authzagent"
	"github.com/hiveot/hub/runtime/digitwin/digitwinagent"
	service4 "github.com/hiveot/hub/runtime/digitwin/service"
	service2 "github.com/hiveot/hub/runtime/history/service"
	"github.com/hiveot/hub/runtime/middleware"
	"github.com/hiveot/hub/runtime/protocols"
)

const DefaultDigiTwinStoreFilename = "digitwin.kvbtree"

// Runtime is the Hub runtime
type Runtime struct {
	cfg *RuntimeConfig

	//AuthnStore api.IAuthnStore
	AuthnSvc      *service.AuthnService
	AuthzSvc      *authz.AuthzService
	DigitwinStore buckets.IBucketStore
	DigitwinSvc   *service4.DigitwinService
	HistorySvc    *service2.HistoryService
	Middleware    *middleware.Middleware
	ProtocolMgr   *protocols.ProtocolsManager
}

func (r *Runtime) Start(env *plugin.AppEnvironment) error {
	err := r.cfg.Setup(env)

	if err != nil {
		return err
	}
	// setup ingress middleware.
	r.Middleware = middleware.NewMiddleware()

	// startup
	// setup the Authentication service
	r.AuthnSvc, err = service.StartAuthnService(&r.cfg.Authn)
	if err != nil {
		return err
	}
	r.AuthzSvc, err = authz.StartAuthzService(&r.cfg.Authz, r.AuthnSvc.AuthnStore)
	if err != nil {
		return err
	}

	// the protocol manager receives messages from clients (source) and
	// sends messages to connected clients (sink)
	//
	r.ProtocolMgr, err = StartProtocolsManager(
		&r.cfg.Protocols, r.cfg.ServerKey, r.cfg.ServerCert, r.cfg.CaCert,
		r.AuthnSvc.SessionAuth, r.Middleware.HandleMessage)

	// The digitwin service directs the message flow between agents and consumers
	// It receives messages from the middleware and uses the protocol manager
	// to send messages to clients.
	r.DigitwinSvc, err = service4.StartDigitwinService(env.StoresDir, r.ProtocolMgr)

	// The middleware validates messages and passes them on to the digitwin service
	if err == nil {
		r.Middleware.SetMessageHandler(r.DigitwinSvc.HandleMessage)
	}

	// last, connect the embedded services via a direct client
	embeddedBinding := r.ProtocolMgr.GetEmbedded()
	cl1 := embeddedBinding.NewClient(service4.DigitwinAgentID)
	_, err = digitwinagent.StartDigiTwinAgent(r.DigitwinSvc, cl1)

	if err == nil {
		cl2 := embeddedBinding.NewClient(authnagent.AuthnAgentID)
		_, err = authnagent.StartAuthnAgent(r.AuthnSvc, cl2)
	}
	if err == nil {
		cl3 := embeddedBinding.NewClient(authzagent.AuthzAgentID)
		_, err = authzagent.StartAuthzAgent(r.AuthzSvc, cl3)
	}
	// last, set the handlers for f
	return err
}

func StartProtocolsManager(cfg *protocols.ProtocolsConfig,
	serverKey keys.IHiveKey, serverCert *tls.Certificate, caCert *x509.Certificate,
	sessionAuth api.IAuthenticator, handler api.MessageHandler) (
	svc *protocols.ProtocolsManager, err error) {

	pm := protocols.NewProtocolManager(
		cfg, serverKey, serverCert, caCert, sessionAuth)

	if err == nil {
		err = pm.Start(handler)
	}
	return pm, err
}

func (r *Runtime) Stop() {
	if r.ProtocolMgr != nil {
		r.ProtocolMgr.Stop()
	}
	if r.HistorySvc != nil {
		r.HistorySvc.Stop()
	}
	if r.DigitwinSvc != nil {
		r.DigitwinSvc.Stop()
	}
	if r.AuthzSvc != nil {
		r.AuthzSvc.Stop()
	}
	if r.AuthnSvc != nil {
		r.AuthnSvc.Stop()
	}
}

func NewRuntime(cfg *RuntimeConfig) *Runtime {
	r := &Runtime{
		cfg: cfg,
	}
	return r
}
