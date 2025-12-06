package router

import (
	"log/slog"
	"sync"

	"github.com/hiveot/hivekit/go/lib/messaging"
	"github.com/hiveot/hivekit/go/lib/servers"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/digitwin/store"
)

// ActionHandler is the API for service action handling
//type ActionHandler func(msg *transports.ThingMessage) (stat transports.ActionStatus)

// PermissionHandler is the handler that authorizes the sender to perform an operation
//
//	senderID is the account ID of the consumer or agent
//	operation is one of the predefined operations, eg WotOpReadEvent
//	dThingID is the ID of the digital twin Thing the request applies to
type PermissionHandler func(senderID, operation, dThingID string) bool

// DigitwinRouter implements the action, event and property flows to and from the
// digital twin.
//
// This uses the transport binding to forward actions and write property requests
// to the agents, and event and property updates to the consumer.
type DigitwinRouter struct {
	dtwStore *store.DigitwinStore

	// internal services are directly invoked
	digitwinAction messaging.RequestHandler
	dtwService     *service.DigitwinService
	authnAction    messaging.RequestHandler
	authzAction    messaging.RequestHandler
	hasPermission  PermissionHandler

	// in-memory cache of active actions lookup by correlationID
	activeCache map[string]ActiveRequestRecord
	// cache map usage mux
	mux sync.Mutex
	// connection manager for forwarding messages to agents or consumers
	transportServer servers.IMessageServer

	// logging of requests and response
	requestLogger *slog.Logger
	// logging of notifications
	notifLogger *slog.Logger
}

// SetNotifLogger replaces the default logger for notifications
func (r *DigitwinRouter) SetNotifLogger(logger *slog.Logger) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.notifLogger = logger
}

// SetRequestLogger replaces the default logger for requests
func (r *DigitwinRouter) SetRequestLogger(logger *slog.Logger) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.requestLogger = logger
}

// SetTransportServer sets the transport server for routing outgoing messages
func (r *DigitwinRouter) SetTransportServer(srv servers.IMessageServer) {
	r.transportServer = srv
}

// NewDigitwinRouter instantiates a new hub messaging router
// Use SetTransportServer for forwarding messages to non-local agents and consumers.
//
//	dtwService is used to update the digital twin status
//	digitwinAction build-in handler for digital twin directory and value requests
//	authnAgent built-in handler for authentication service requests
//	authzAgent built-in handler for authorization service requests
//	permissionHandler built-in handler for authorizing requests
//	transportServer is the transport binding for forwarding service requests
func NewDigitwinRouter(
	dtwService *service.DigitwinService,
	digitwinAction messaging.RequestHandler,
	authnAgent messaging.RequestHandler,
	authzAgent messaging.RequestHandler,
	permissionHandler PermissionHandler,
	transportServer servers.IMessageServer,
) *DigitwinRouter {
	r := &DigitwinRouter{
		dtwStore: dtwService.DtwStore,
		//cm:             cm,
		transportServer: transportServer,
		authnAction:     authnAgent,
		authzAction:     authzAgent,
		hasPermission:   permissionHandler,
		digitwinAction:  digitwinAction,
		dtwService:      dtwService,
		activeCache:     make(map[string]ActiveRequestRecord),
		requestLogger:   slog.Default(),
		notifLogger:     slog.Default(),
	}
	return r
}
