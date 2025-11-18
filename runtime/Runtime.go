package runtime

import (
	"log/slog"
	"os"
	"path"
	"time"

	"github.com/hiveot/hivekit/go/logging"
	"github.com/hiveot/hivekit/go/messaging"
	"github.com/hiveot/hivekit/go/server"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/hiveot/hub/lib/plugin"
	authn "github.com/hiveot/hub/runtime/authn/api"
	"github.com/hiveot/hub/runtime/authn/service"
	authz "github.com/hiveot/hub/runtime/authz/api"
	service2 "github.com/hiveot/hub/runtime/authz/service"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/runtime/digitwin/router"
	service4 "github.com/hiveot/hub/runtime/digitwin/service"
	jsoniter "github.com/json-iterator/go"
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
	TransportsMgr  *server.TransportManager

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
//func (r *Runtime) GetForm(op string, protocol string) (f *td.Form) {
//	srv := r.TransportsMgr.GetServer(protocol)
//	if srv != nil {
//		return srv.GetForm(op, "", "")
//	}
//	return nil
//}

// GetConnectURL returns the URL for connecting with the given protocol type.
// If the protocol is not available, the https fallback is returned.
func (r *Runtime) GetConnectURL() string {
	return r.TransportsMgr.GetConnectURL()
}

// GetTD returns the TD with the given digitwin ID.
// This passes the request to the digitwin directory store.
// Intended for testing to get a TD.
func (r *Runtime) GetTD(dThingID string) (td *td.TD) {
	tdJSON, err := r.DigitwinSvc.DirSvc.RetrieveThing("", dThingID)
	if err != nil {
		return nil
	}
	_ = jsoniter.UnmarshalFromString(tdJSON, &td)
	return td
}

// Start the Hub runtime.
// This starts the runtime authn, authz, digitwin and transport services.
func (r *Runtime) Start(env *plugin.AppEnvironment) error {
	slog.Warn("Starting HiveOT runtime")
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
	// This creates an authenticator for authenticating connections and adding security scheme to TD.
	// The transport manager calls SetAuthServerURI() to set the authentication endpoint for including in the TD.
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
	if err != nil {
		return err
	}

	// 3: The digitwin service provides a directory for digital twin Things and notifies
	// of changes to digital twin state.
	prof = authn.ClientProfile{
		ClientID:   digitwin.ThingDirectoryAgentID,
		ClientType: authn.ClientTypeService,
	}
	_ = r.AuthnSvc.AuthnStore.Add(digitwin.ThingDirectoryAgentID, prof)
	_ = r.AuthnSvc.AuthnStore.SetRole(digitwin.ThingDirectoryAgentID, string(authz.ClientRoleService))
	r.DigitwinSvc, _, err = service4.StartDigitwinService(
		env.StoresDir, r.SendNotification, r.cfg.ProtocolsConfig.IncludeForms)
	if err != nil {
		return err
	}
	dtwAgent := service4.NewDigitwinAgent(r.DigitwinSvc)

	// 4: The digitwin router receives all incoming requests, responses and notifications
	// from agents and consumers.
	//
	//  Digital twin requests are forward to the digital twin service.
	//  Authentication requests are forwarded to the authn service
	//	Authorization requests are forwarded to the authz service
	//
	// Actions for remote Things are forwarded to the Things using the transport
	// manager, if set. The transport manager can be set with 'SetTransportServer'.
	//
	// Internal services are authn,authz and the digital twin directory and value service.
	r.DigitwinRouter = router.NewDigitwinRouter(
		r.DigitwinSvc,
		dtwAgent.HandleRequest,
		r.AuthnAgent.HandleRequest,
		r.AuthzAgent.HandleAction,
		r.AuthzAgent.HasPermission,
		nil)

	// 5: Create the transports but do not start yet.
	//
	// The authenticator provided by the authn service (authnSvc.SessionAuth) is used to
	// both authenticate connections and to add the securityScheme to TDs in TransportManager.AddTDForms.
	//
	// Note if you're trying to find out where the digital twin TDs get their forms and
	// security fields then the answer is in TransportManager.AddTDForms.
	// On startup the TransportManager is given the AuthnSvc.SessionAuth authenticator and
	// it updates it with the login URI. In AddTDForms it uses the authenticator to set the
	// security scheme and uses each of the protocols to set the forms for affordances.
	// Look for AddTDForms in each protocol. This is a rather roundabout way of getting
	// the security scheme in the TD but necessary as it is pluggable.
	// See AuthnSvc above
	r.TransportsMgr = server.NewTransportManager(
		&r.cfg.ProtocolsConfig,
		r.cfg.ServerCert,
		r.cfg.CaCert,
		r.AuthnSvc.SessionAuth,
		r.DigitwinRouter.HandleNotification,
		r.DigitwinRouter.HandleRequest,
		r.DigitwinRouter.HandleResponse,
	)
	// the router needs the transport to forward requests and send responses
	r.DigitwinRouter.SetTransportServer(r.TransportsMgr)

	// generated digitwin TDs include forms produced by transport protocols
	r.DigitwinSvc.SetFormsHook(r.TransportsMgr.AddTDForms)

	// 6. Setup logging of requests and notifications
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

	// 7: Setup the directory with TDs and permissions for agents and consumers
	// Add the TDs of the built-in services (authn,authz,directory,values) to the directory
	_ = r.DigitwinSvc.DirSvc.UpdateThing(authn.AdminAgentID, authn.AdminTD)
	_ = r.DigitwinSvc.DirSvc.UpdateThing(authn.UserAgentID, authn.UserTD)
	_ = r.DigitwinSvc.DirSvc.UpdateThing(authz.AdminAgentID, authz.AdminTD)
	// The digital twin of the directoryTD will be used by discovery (below) to publish the .well-know/wot
	_ = r.DigitwinSvc.DirSvc.UpdateThing(digitwin.ThingDirectoryAgentID, digitwin.ThingDirectoryTD)
	_ = r.DigitwinSvc.DirSvc.UpdateThing(digitwin.ThingValuesAgentID, digitwin.ThingValuesTD)

	// set agent permissions to update the directory
	// agents can update the directory with their TDs
	err = r.AuthzSvc.SetPermissions(digitwin.ThingDirectoryAgentID, authz.ThingPermissions{
		AgentID: digitwin.ThingDirectoryAgentID,
		ThingID: digitwin.ThingDirectoryServiceID,
		Allow:   []authz.ClientRole{authz.ClientRoleAgent},
	})
	if err != nil {
		slog.Error("failed SetPermissions. Continuing...", "err", err.Error())
	}
	// anyone else can read the directory, except those with no role
	err = r.AuthzSvc.SetPermissions(digitwin.ThingDirectoryAgentID, authz.ThingPermissions{
		AgentID: digitwin.ThingDirectoryAgentID,
		ThingID: digitwin.ThingDirectoryServiceID,
		Deny:    []authz.ClientRole{authz.ClientRoleNone},
	})
	if err != nil {
		slog.Error("failed SetPermissions. Continuing...", "err", err.Error())
	}
	// with the router in place activate the transport server to receive and send messages
	err = r.TransportsMgr.Start()
	if err != nil {
		return err
	}

	// 8: last, start discovery and exploration of the digital twin directory
	if r.cfg.ProtocolsConfig.EnableDiscovery {
		// the directory TD was added above (r.DigitwinSvc.DirSvc.UpdateThing)
		dirTDJson, err := r.DigitwinSvc.DirSvc.RetrieveThing(digitwin.ThingDirectoryAgentID, digitwin.ThingDirectoryDThingID)
		if err == nil {
			protocolsCfg := r.cfg.ProtocolsConfig
			err = r.TransportsMgr.StartDiscovery(
				r.cfg.ProtocolsConfig.DiscoveryInstanceName, protocolsCfg.DirectoryTDPath, dirTDJson,
			)
		}
		if err != nil {
			slog.Error("failed starting discovery. Continuing anyways...", "err", err.Error())
		}
	}
	return err
}

// SendNotification sends an event or property response message to subscribers.
// This simply forwards the notification to the transport manager.
func (r *Runtime) SendNotification(notif *messaging.NotificationMessage) {
	r.TransportsMgr.SendNotification(notif)
}

func (r *Runtime) Stop() {
	slog.Warn("Stopping the HiveOT runtime")
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
