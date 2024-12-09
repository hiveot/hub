package router

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/digitwin/store"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/connections"
	"log/slog"
	"sync"
)

// ActionHandler is the API for service action handling
//type ActionHandler func(msg *transports.ThingMessage) (stat transports.RequestStatus)

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
	activeCache map[string]ActionFlowRecord
	// cache map usage mux
	mux sync.Mutex
	// connection manager for sending messages to agent or consumer
	cm *connections.ConnectionManager
}

// HandleMessage routes updates from agents to consumers
// replyTo is only used if no immediate result is available
func (svc *DigitwinRouter) HandleMessage(
	msg *transports.ThingMessage, replyTo string) (completed bool, output any, err error) {

	// middleware: authorize the request.
	// TODO: use a middleware chain
	if !svc.hasPermission(msg.SenderID, msg.Operation, msg.ThingID) {
		err = fmt.Errorf("unauthorized. client '%s' does not have permission"+
			" to invoke operation '%s' on Thing '%s'",
			msg.SenderID, msg.Operation, msg.ThingID)
		slog.Warn(err.Error())
		return false, nil, err
	}

	// first handle remote operations that don't immediately return data
	completed = true
	switch msg.Operation {
	// operations with no immediate result
	case vocab.HTOpPublishEvent:
		svc.HandlePublishEvent(msg)
	case vocab.HTOpUpdateProperty:
		svc.HandleUpdateProperty(msg)
	case vocab.HTOpUpdateMultipleProperties:
		svc.HandleUpdateMultipleProperties(msg)
	case vocab.HTOpUpdateActionStatus, vocab.HTOpUpdateActionStatuses:
		svc.HandleActionResponse(msg)
	case vocab.HTOpUpdateTD:
		completed, output, err = svc.HandleUpdateTD(msg)
	case vocab.OpInvokeAction:
		completed, output, err = svc.HandleInvokeAction(msg, replyTo)
	case vocab.OpWriteProperty:
		completed, output, err = svc.HandleWriteProperty(msg, replyTo)

	// operations from embedded services with immediate result
	case vocab.HTOpLogin:
		completed, output, err = svc.HandleLogin(msg)
	case vocab.HTOpLogout:
		completed, output, err = svc.HandleLogout(msg)
	case vocab.HTOpRefresh:
		completed, output, err = svc.HandleLoginRefresh(msg)
	case vocab.HTOpReadAllEvents:
		completed, output, err = svc.HandleReadAllEvents(msg)
	case vocab.OpReadAllProperties:
		completed, output, err = svc.HandleReadAllProperties(msg)
	case vocab.HTOpReadEvent:
		completed, output, err = svc.HandleReadEvent(msg)
	case vocab.OpReadProperty:
		completed, output, err = svc.HandleReadProperty(msg)
	default:
		completed = false // oops, no such thing
		err = fmt.Errorf("unknown operation '%s' from client '%s'", msg.Operation, msg.SenderID)
		slog.Warn(err.Error())
	}
	if err != nil {
		slog.Warn("HandleMessage failed", "err", err.Error())
	}
	return completed, output, err
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
		activeCache:    make(map[string]ActionFlowRecord),
	}
	return ar
}
