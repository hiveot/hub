package runtime

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/runtime/authn/service"
	service2 "github.com/hiveot/hub/runtime/authz/service"
	service4 "github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/hubrouter"
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
	AuthnAgent    *service.AuthnAgent
	AuthzAgent    *service2.AuthzAgent
	dtwStore      *service4.DigitwinStore
	DigitwinSvc   *service4.DigitwinService
	HubRouter     *hubrouter.HubRouter
	TransportsMgr *transports.TransportManager
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

	// startup
	// setup the Authentication service
	r.AuthnSvc, err = service.StartAuthnService(&r.cfg.Authn)
	if err != nil {
		return err
	}
	r.AuthnAgent = service.StartAuthnAgent(r.AuthnSvc)

	// start authorization service and add its account
	r.AuthzSvc, err = service2.StartAuthzService(&r.cfg.Authz, r.AuthnSvc.AuthnStore)
	if err != nil {
		return err
	}
	// provide access to the authz agent
	prof := authn.ClientProfile{
		ClientID:   authz.AdminAgentID,
		ClientType: authn.ClientTypeService,
	}
	_ = r.AuthnSvc.AuthnStore.Add(authz.AdminAgentID, prof)
	_ = r.AuthnSvc.AuthnStore.SetRole(authz.AdminAgentID, string(authz.ClientRoleService))

	r.AuthzAgent, err = service2.StartAuthzAgent(r.AuthzSvc)

	// The digitwin service directs the message flow between agents and consumers
	// It receives messages from the middleware and uses the protocol manager
	// to send messages to clients.
	r.DigitwinSvc, r.dtwStore, err = service4.StartDigitwinService(env.StoresDir)
	dtwAgent := service4.NewDigitwinAgent(r.DigitwinSvc)
	// The transport passes incoming messages on to the hub-router, which in
	// turn updates the digital twin and forwards the requests.
	r.HubRouter = hubrouter.NewHubRouter(r.DigitwinSvc,
		dtwAgent, r.AuthnAgent, r.AuthzAgent)

	// the protocol manager receives messages from clients (source) and
	// sends messages to connected clients (sink)
	r.TransportsMgr, err = transports.StartTransportManager(
		&r.cfg.Transports,
		r.cfg.ServerCert,
		r.cfg.CaCert,
		r.AuthnSvc.SessionAuth,
		r.HubRouter,
		r.DigitwinSvc)
	if err != nil {
		return err
	}
	// outgoing messages are handled by the sub-protocols of this transport
	r.DigitwinSvc.SetTransportHook(r.TransportsMgr)
	r.HubRouter.SetTransport(r.TransportsMgr)
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
