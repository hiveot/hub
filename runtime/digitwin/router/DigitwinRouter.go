package router

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/digitwin/store"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/connections"
	"github.com/teris-io/shortid"
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
func (svc *DigitwinRouter) HandleMessage(
	msg *transports.ThingMessage, replyTo transports.IServerConnection) {

	var isHandled = true

	// middleware: authorize the request.
	// TODO: use a middleware chain
	if !svc.hasPermission(msg.SenderID, msg.Operation, msg.ThingID) {
		err := fmt.Sprintf("Unauthorized. client '%s' does not have permission"+
			" to invoke operation '%s' on Thing '%s'",
			msg.SenderID, msg.Operation, msg.ThingID)
		slog.Warn(err)
		replyTo.SendError(msg.ThingID, msg.Name, err, msg.RequestID)
		return
	}

	// first handle send-and-forget operations (agent publications)
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
		svc.HandleUpdateTD(msg)
	case vocab.OpInvokeAction:
		svc.HandleInvokeAction(msg, replyTo)
	case vocab.OpWriteProperty:
		svc.HandleWriteProperty(msg, replyTo)
	default:
		isHandled = false
	}
	if isHandled {
		return
	}

	if msg.RequestID == "" {
		msg.RequestID = "action-" + shortid.MustGenerate()
	}
	// operations from embedded services with immediate result
	var err error
	var output any

	switch msg.Operation {
	case vocab.HTOpLogin:
		output, err = svc.HandleLogin(msg)
	case vocab.HTOpLogout:
		err = svc.HandleLogout(msg)
	case vocab.HTOpRefresh:
		output, err = svc.HandleLoginRefresh(msg)
	case vocab.HTOpReadAllEvents:
		output, err = svc.HandleReadAllEvents(msg)
	case vocab.OpReadAllProperties:
		output, err = svc.HandleReadAllProperties(msg)
	case vocab.HTOpReadEvent:
		output, err = svc.HandleReadEvent(msg)
	case vocab.OpReadProperty:
		output, err = svc.HandleReadProperty(msg)
	default:
		err = fmt.Errorf("Unknown operation '%s' from client '%s'", msg.Operation, msg.SenderID)
		slog.Warn(err.Error())
	}
	if err != nil {
		slog.Warn("HandleMessage failed", "err", err.Error())
		replyTo.SendError(msg.ThingID, msg.Name, err.Error(), msg.RequestID)
	} else {
		_ = replyTo.SendResponse(msg.ThingID, msg.Name, output, msg.RequestID)
	}
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
