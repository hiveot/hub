package runtime

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/runtime/authn/service"
	service2 "github.com/hiveot/hub/runtime/authz/service"
	service4 "github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/middleware"
	"github.com/hiveot/hub/runtime/transports"
	"log/slog"
)

const DefaultDigiTwinStoreFilename = "digitwin.kvbtree"

// Runtime is the Hub runtime. This is the bare-bone core of the hub that operates the
// communication protocols with services for auth, inbox, outbox and directory.
type Runtime struct {
	cfg *RuntimeConfig

	//AuthnStore api.IAuthnStore
	AuthnSvc      *service.AuthnService
	AuthzSvc      *service2.AuthzService
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
	// start authorization service and hook into the middleware
	r.AuthzSvc, err = service2.StartAuthzService(&r.cfg.Authz, r.AuthnSvc.AuthnStore)
	if err != nil {
		return err
	}
	r.Middleware.AddMiddlewareHandler(r.AuthzSvc.HasPubPermission)

	// the protocol manager receives messages from clients (source) and
	// sends messages to connected clients (sink)
	r.TransportsMgr, err = transports.StartTransportManager(
		&r.cfg.Transports,
		r.cfg.ServerKey,
		r.cfg.ServerCert,
		r.cfg.CaCert,
		r.AuthnSvc.SessionAuth,
		r.Middleware.HandleMessage)
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

	// last:
	// ensure authz and digitwin are registered agents
	embeddedBinding := r.TransportsMgr.GetEmbedded()

	if err == nil {
		cl2 := embeddedBinding.NewClient(authn.AdminAgentID)
		_, err = service.StartAuthnAgent(r.AuthnSvc, cl2)
	}
	if err == nil {
		// provide access to the authz agent
		prof := authn.ClientProfile{
			ClientID:   authz.AdminAgentID,
			ClientType: authn.ClientTypeService,
		}
		_ = r.AuthnSvc.AuthnStore.Add(authz.AdminAgentID, prof)
		_ = r.AuthnSvc.AuthnStore.SetRole(authz.AdminAgentID, authn.ClientRoleService)
		cl3 := embeddedBinding.NewClient(authz.AdminAgentID)
		_, err = service2.StartAuthzAgent(r.AuthzSvc, cl3)
	}
	if err == nil {
		prof := authn.ClientProfile{
			ClientID:   digitwin.DirectoryAgentID,
			ClientType: authn.ClientTypeService,
		}
		_ = r.AuthnSvc.AuthnStore.Add(digitwin.DirectoryAgentID, prof)
		_ = r.AuthnSvc.AuthnStore.SetRole(digitwin.DirectoryAgentID, authn.ClientRoleService)
		cl1 := embeddedBinding.NewClient(digitwin.DirectoryAgentID)
		_, err = service4.StartDigiTwinAgent(r.DigitwinSvc, cl1)
	}
	return err
}

func (r *Runtime) Stop() {
	if r.AuthnSvc != nil {
		r.AuthnSvc.Stop()
	}
	if r.AuthzSvc != nil {
		r.AuthzSvc.Stop()
	}
	if r.DigitwinSvc != nil {
		r.DigitwinSvc.Stop()
	}
	if r.TransportsMgr != nil {
		r.TransportsMgr.Stop()
	}
	slog.Info("HiveOT Hub Runtime Stopped")
}

func NewRuntime(cfg *RuntimeConfig) *Runtime {
	r := &Runtime{
		cfg: cfg,
	}
	return r
}
