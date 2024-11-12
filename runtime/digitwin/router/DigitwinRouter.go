package router

import (
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/connections"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/digitwin/store"
	"sync"
)

// DigitwinRouter implements the action, event and property flows to and from the
// digital twin.
//
// This uses the transport binding to forward actions and write property requests
// to the agents, and event and property updates to the consumer.
type DigitwinRouter struct {
	dtwStore *store.DigitwinStore

	// internal services are directly invoked
	digitwinAction    api.ActionHandler
	dtwService        *service.DigitwinService
	authnAction       api.ActionHandler
	authzAction       api.ActionHandler
	permissionHandler api.PermissionHandler

	// in-memory cache of active actions lookup by requestID
	activeCache map[string]*ActionFlowRecord
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
	digitwinAction api.ActionHandler,
	authnAgent api.ActionHandler,
	authzAgent api.ActionHandler,
	permissionHandler api.PermissionHandler,
	cm *connections.ConnectionManager,
) *DigitwinRouter {
	ar := &DigitwinRouter{
		dtwStore:          dtwService.DtwStore,
		cm:                cm,
		authnAction:       authnAgent,
		authzAction:       authzAgent,
		permissionHandler: permissionHandler,
		digitwinAction:    digitwinAction,
		dtwService:        dtwService,
		activeCache:       make(map[string]*ActionFlowRecord),
	}
	return ar
}
