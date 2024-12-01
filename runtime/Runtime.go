package runtime

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn/service"
	service2 "github.com/hiveot/hub/runtime/authz/service"
	"github.com/hiveot/hub/runtime/digitwin/router"
	service4 "github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/transports"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/hiveot/hub/wot/transports/connections"
	"log/slog"
)

// Runtime is the Hub runtime. This is the bare-bone core of the hub that operates the
// communication protocols with services for auth, inbox, outbox and directory.
type Runtime struct {
	cfg *RuntimeConfig

	AuthnSvc       *service.AuthnService
	AuthzSvc       *service2.AuthzService
	AuthnAgent     *service.AuthnAgent
	AuthzAgent     *service2.AuthzAgent
	DigitwinSvc    *service4.DigitwinService
	DigitwinRouter api.IDigitwinRouter
	CM             *connections.ConnectionManager
	TransportsMgr  *transports.ProtocolManager
}

// GetForm returns the form for an operation using a transport protocol binding
// These forms point to the use of a digital twin via the hub runtime.
// If the protocol is not found this returns a nil and might cause a panic
func (r *Runtime) GetForm(op string, protocol string) (f tdd.Form) {
	return r.TransportsMgr.GetForm(op, protocol)
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
	r.DigitwinSvc, _, err = service4.StartDigitwinService(env.StoresDir, r.CM)
	dtwAgent := service4.NewDigitwinAgent(r.DigitwinSvc)

	// The transport passes incoming messages on to the hub-router, which in
	// turn updates the digital twin and forwards the requests.
	r.DigitwinRouter = router.NewDigitwinRouter(
		r.DigitwinSvc,
		dtwAgent.HandleAction,
		r.AuthnAgent.HandleAction,
		r.AuthzAgent.HandleAction,
		r.AuthzAgent.HasPermission,
		r.CM)

	// the protocol manager receives messages from clients (source) and
	// sends messages to connected clients (sink)
	r.TransportsMgr, err = transports.StartProtocolManager(
		&r.cfg.Transports,
		r.cfg.ServerCert,
		r.cfg.CaCert,
		r.AuthnSvc.SessionAuth,
		r.DigitwinRouter,
		r.CM,
	)
	if err != nil {
		return err
	}
	// outgoing messages are handled by the sub-protocols of this transport
	r.DigitwinSvc.SetFormsHook(r.TransportsMgr.AddTDForms)

	// add the TDs of the built-in services (authn,authz,directory,values) to the directory
	_ = r.DigitwinSvc.DirSvc.UpdateTD(authn.AdminAgentID, authn.AdminTD)
	_ = r.DigitwinSvc.DirSvc.UpdateTD(authn.UserAgentID, authn.UserTD)
	_ = r.DigitwinSvc.DirSvc.UpdateTD(authz.AdminAgentID, authz.AdminTD)
	_ = r.DigitwinSvc.DirSvc.UpdateTD(digitwin.DirectoryAgentID, digitwin.DirectoryTD)
	_ = r.DigitwinSvc.DirSvc.UpdateTD(digitwin.ValuesAgentID, digitwin.ValuesTD)

	// agents can update to the directory
	// fixme: authn.AdminServiceID is not correct but will have to do for now
	_ = r.AuthzSvc.SetPermissions(authn.AdminServiceID, authz.ThingPermissions{
		AgentID: digitwin.DirectoryAgentID,
		ThingID: digitwin.DirectoryServiceID,
		Allow:   []authz.ClientRole{authz.ClientRoleAgent, ""},
	})
	// anyone else can read the directory
	// FIXME: differentiate per action based on TD default?
	_ = r.AuthzSvc.SetPermissions(authn.AdminServiceID, authz.ThingPermissions{
		AgentID: digitwin.DirectoryAgentID,
		ThingID: digitwin.DirectoryServiceID,
		Deny:    []authz.ClientRole{authz.ClientRoleNone, ""},
	})

	return err
}

func (r *Runtime) Stop() {
	nrConnections, _ := r.CM.GetNrConnections()

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
	r.CM.CloseAll()
	if nrConnections > 0 {
		slog.Warn(fmt.Sprintf(
			"HiveOT Hub Runtime stopped. Force closed %d connections", nrConnections))
	}
}

func NewRuntime(cfg *RuntimeConfig) *Runtime {
	r := &Runtime{
		cfg: cfg,
		CM:  connections.NewConnectionManager(),
	}
	return r
}
