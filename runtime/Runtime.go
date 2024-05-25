package runtime

import (
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn/authnagent"
	"github.com/hiveot/hub/runtime/authn/service"
	"github.com/hiveot/hub/runtime/authz"
	"github.com/hiveot/hub/runtime/authz/authzagent"
	"github.com/hiveot/hub/runtime/digitwin/digitwinagent"
	service4 "github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/middleware"
	"github.com/hiveot/hub/runtime/transports"
)

const DefaultDigiTwinStoreFilename = "digitwin.kvbtree"

// Runtime is the Hub runtime. This is the bare-bone core of the hub that operates the
// communication protocols with services for auth, inbox, outbox and directory.
type Runtime struct {
	cfg *RuntimeConfig

	//AuthnStore api.IAuthnStore
	AuthnSvc      *service.AuthnService
	AuthzSvc      *authz.AuthzService
	DigitwinStore buckets.IBucketStore
	DigitwinSvc   *service4.DigitwinService
	Middleware    *middleware.Middleware
	TransportsMgr *transports.TransportsManager
}

// Start the Hub runtime
// This verifies and repairs the setup if needed by creating missing directories and
// generating the server keys and certificate files if missing.
// This uses the directory structure obtained from the app environment.
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
	r.TransportsMgr, err = transports.StartProtocolManager(
		&r.cfg.Transports, r.cfg.ServerKey, r.cfg.ServerCert, r.cfg.CaCert,
		r.AuthnSvc.SessionAuth, r.Middleware.HandleMessage)
	if err != nil {
		return err
	}

	// The digitwin service directs the message flow between agents and consumers
	// It receives messages from the middleware and uses the protocol manager
	// to send messages to clients.
	r.DigitwinSvc, err = service4.StartDigitwinService(env.StoresDir, r.TransportsMgr)

	// The middleware validates messages and passes them on to the digitwin service
	if err == nil {
		r.Middleware.SetMessageHandler(r.DigitwinSvc.HandleMessage)
	}

	// last, connect the embedded services via a direct client
	embeddedBinding := r.TransportsMgr.GetEmbedded()
	cl1 := embeddedBinding.NewClient(digitwinagent.DigiTwinAgentID)
	_, err = digitwinagent.StartDigiTwinAgent(r.DigitwinSvc, cl1)

	if err == nil {
		cl2 := embeddedBinding.NewClient(api.AuthnAgentID)
		_, err = authnagent.StartAuthnAgent(r.AuthnSvc, cl2)
	}
	if err == nil {
		cl3 := embeddedBinding.NewClient(authzagent.AuthzAgentID)
		_, err = authzagent.StartAuthzAgent(r.AuthzSvc, cl3)
	}
	// last, set the handlers for f
	return err
}

func (r *Runtime) Stop() {
	if r.TransportsMgr != nil {
		r.TransportsMgr.Stop()
	}
	//if r.HistorySvc != nil {
	//	r.HistorySvc.Stop()
	//}
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
