package router

import (
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/digitwin/store"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/connections"
	"sync"
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
	digitwinAction transports.RequestHandler
	dtwService     *service.DigitwinService
	authnAction    transports.RequestHandler
	authzAction    transports.RequestHandler
	hasPermission  PermissionHandler

	// in-memory cache of active actions lookup by requestID
	activeCache map[string]ActiveRequestRecord
	// cache map usage mux
	mux sync.Mutex
	// connection manager for sending messages to agent or consumer
	cm *connections.ConnectionManager
}

// NewDigitwinRouter instantiates a new hub messaging router
// Use SetTransport to link to a transport for forwarding messages to
// agents and consumers.
//
//	dtwStore is used to update the digital twin status
//	tb is the transport binding for forwarding service requests
func NewDigitwinRouter(
	dtwService *service.DigitwinService,
	digitwinAction transports.RequestHandler,
	authnAgent transports.RequestHandler,
	authzAgent transports.RequestHandler,
	permissionHandler PermissionHandler,
	cm *connections.ConnectionManager,
) *DigitwinRouter {
	ar := &DigitwinRouter{
		dtwStore:       dtwService.DtwStore,
		cm:             cm,
		authnAction:    authnAgent,
		authzAction:    authzAgent,
		hasPermission:  permissionHandler,
		digitwinAction: digitwinAction,
		dtwService:     dtwService,
		activeCache:    make(map[string]ActiveRequestRecord),
	}
	return ar
}
