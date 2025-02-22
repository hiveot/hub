package runtime

import (
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/servers"
	authn "github.com/hiveot/hub/runtime/authn/api"
	"github.com/hiveot/hub/runtime/authn/service"
	authz "github.com/hiveot/hub/runtime/authz/api"
	service2 "github.com/hiveot/hub/runtime/authz/service"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/runtime/digitwin/router"
	service4 "github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"os"
	"path"
	"time"
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
	DigitwinRouter *router.DigitwinRouter
	TransportsMgr  *servers.TransportManager

	// logging of request and response messages
	requestLogger  *slog.Logger
	requestLogFile *os.File

	// logging of notification messages
	notifLogger  *slog.Logger
	notifLogFile *os.File

	// logging of all other runtime messages
	runtimeLogger  *slog.Logger
	runtimeLogFile *os.File
}

// GetForm returns the form for an operation using a transport protocol binding
// These forms point to the use of a digital twin via the hub runtime.
// If the protocol is not found this returns a nil and might cause a panic
func (r *Runtime) GetForm(op string, protocol string) (f *td.Form) {
	srv := r.TransportsMgr.GetServer(protocol)
	if srv != nil {
		return srv.GetForm(op, "", "")
	}
	return nil
}

// GetConnectURL returns the URL for connecting with the given protocol type.
// If the protocol is not available, the https fallback is returned.
func (r *Runtime) GetConnectURL() string {
	return r.TransportsMgr.GetConnectURL()
}

// GetTD returns the TD with the given digitwin ID.
// This passes the request to the digitwin directory store.
// Intended for testing to get a TD.
func (r *Runtime) GetTD(dThingID string) (td *td.TD) {
	tdJSON, err := r.DigitwinSvc.DirSvc.ReadTD("", dThingID)
	if err != nil {
		return nil
	}
	_ = jsoniter.UnmarshalFromString(tdJSON, &td)
	return td
}

// Start the Hub runtime.
// This starts the runtime authn, authz, digitwin and transport services.
func (r *Runtime) Start(env *plugin.AppEnvironment) error {
	err := r.cfg.Setup(env)
	if err != nil {
		return err
	}

	// setup logging sinks
	if r.cfg.RuntimeLog != "" {
		runtimeLogfileName := path.Join(env.LogsDir, r.cfg.RuntimeLog)
		logging.SetLogging(env.LogLevel, runtimeLogfileName)
	}

	// startup

	// 1: setup the Authentication service
	r.AuthnSvc, err = service.StartAuthnService(&r.cfg.Authn)
	if err != nil {
		return err
	}
	r.AuthnAgent = service.StartAuthnAgent(r.AuthnSvc)

	// 2: start authorization service and add its account
	r.AuthzSvc, err = service2.StartAuthzService(&r.cfg.Authz, r.AuthnSvc.AuthnStore)
	if err != nil {
		return err
	}
	// provide admin access to the authz agent
	prof := authn.ClientProfile{
		ClientID:   authz.AdminAgentID,
		ClientType: authn.ClientTypeService,
	}
	_ = r.AuthnSvc.AuthnStore.Add(authz.AdminAgentID, prof)
	_ = r.AuthnSvc.AuthnStore.SetRole(authz.AdminAgentID, string(authz.ClientRoleService))
	r.AuthzAgent, err = service2.StartAuthzAgent(r.AuthzSvc)

	// Start the transport servers.
	r.TransportsMgr, err = servers.StartTransportManager(
		&r.cfg.ProtocolsConfig,
		r.cfg.ServerCert,
		r.cfg.CaCert,
		r.AuthnSvc.SessionAuth,
	)

	// The digitwin service directs the message flow between agents and consumers
	// It receives messages from the middleware and uses the transport manager
	// to send messages to clients.
	r.DigitwinSvc, _, err = service4.StartDigitwinService(env.StoresDir, r.SendNotification)
	dtwAgent := service4.NewDigitwinAgent(r.DigitwinSvc)

	// The digitwin router receives all incoming messages from the transport.
	// The router passes requests to internal and external services and things
	// and returns responses to the request sender.
	// Internal services are authn,authz and the digital twin directory and value service.
	r.DigitwinRouter = router.NewDigitwinRouter(
		r.DigitwinSvc,
		dtwAgent.HandleRequest,
		r.AuthnAgent.HandleRequest,
		r.AuthzAgent.HandleAction,
		r.AuthzAgent.HasPermission,
		r.TransportsMgr)
	r.TransportsMgr.SetRequestHandler(r.DigitwinRouter.HandleRequest)
	r.TransportsMgr.SetResponseHandler(r.DigitwinRouter.HandleResponse)

	// When generating digitwin TDs use the forms produced by transport protocols
	r.DigitwinSvc.SetFormsHook(r.TransportsMgr.AddTDForms)

	// Setup  logging of requests and notifications
	if r.cfg.RequestLog != "" {
		requestLogfileName := path.Join(env.LogsDir, r.cfg.RequestLog)
		r.requestLogger, r.requestLogFile = logging.NewFileLogger(
			requestLogfileName, r.cfg.LogfileInJson)
		r.DigitwinRouter.SetRequestLogger(r.requestLogger)
	}
	if r.cfg.NotifLog != "" {
		notifLogfileName := path.Join(env.LogsDir, r.cfg.NotifLog)
		r.notifLogger, r.notifLogFile = logging.NewFileLogger(
			notifLogfileName, r.cfg.LogfileInJson)
		r.DigitwinRouter.SetNotifLogger(r.notifLogger)
	}
	if err != nil {
		return err
	}

	// Add the TDs of the built-in services (authn,authz,directory,values) to the directory
	_ = r.DigitwinSvc.DirSvc.UpdateTD(authn.AdminAgentID, authn.AdminTD)
	_ = r.DigitwinSvc.DirSvc.UpdateTD(authn.UserAgentID, authn.UserTD)
	_ = r.DigitwinSvc.DirSvc.UpdateTD(authz.AdminAgentID, authz.AdminTD)
	_ = r.DigitwinSvc.DirSvc.UpdateTD(digitwin.ThingDirectoryAgentID, digitwin.ThingDirectoryTD)
	_ = r.DigitwinSvc.DirSvc.UpdateTD(digitwin.ThingValuesAgentID, digitwin.ThingValuesTD)

	// start discovery and exploration of the digital twin directory
	if r.cfg.ProtocolsConfig.EnableDiscovery {
		dirTDJson, err := r.DigitwinSvc.DirSvc.ReadTD(digitwin.ThingDirectoryAgentID, digitwin.ThingDirectoryDThingID)
		if err == nil {
			protocolsCfg := r.cfg.ProtocolsConfig
			err = r.TransportsMgr.StartDiscovery(
				r.cfg.ProtocolsConfig.InstanceName, protocolsCfg.DirectoryTDPath, dirTDJson,
			)
		}
	}

	// set agent permissions to update the directory
	// agents can update to the directory
	_ = r.AuthzSvc.SetPermissions(authn.AdminServiceID, authz.ThingPermissions{
		AgentID: digitwin.ThingDirectoryAgentID,
		ThingID: digitwin.ThingDirectoryServiceID,
		Allow:   []authz.ClientRole{authz.ClientRoleAgent},
	})
	// anyone else can read the directory, except those with no role
	_ = r.AuthzSvc.SetPermissions(authn.AdminServiceID, authz.ThingPermissions{
		AgentID: digitwin.ThingDirectoryAgentID,
		ThingID: digitwin.ThingDirectoryServiceID,
		Deny:    []authz.ClientRole{authz.ClientRoleNone},
	})

	return err
}

// SendNotification sends an event or property response message to subscribers.
// This simply forwards the notification to the transport manager.
func (r *Runtime) SendNotification(notif *messaging.ResponseMessage) error {
	r.TransportsMgr.SendNotification(notif)
	return nil
}

func (r *Runtime) Stop() {
	// wait a little to allow ongoing connection closure to complete
	time.Sleep(time.Millisecond * 10)
	//nrConnections, _ := r.CM.GetNrConnections()

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
	if r.notifLogFile != nil {
		_ = r.notifLogFile.Close()
		r.notifLogFile = nil
	}
	if r.requestLogFile != nil {
		_ = r.requestLogFile.Close()
		r.requestLogFile = nil
	}
	if r.runtimeLogFile != nil {
		_ = r.runtimeLogFile.Close()
		r.runtimeLogFile = nil
	}
}

func NewRuntime(cfg *RuntimeConfig) *Runtime {
	r := &Runtime{
		cfg: cfg,
	}
	return r
}
