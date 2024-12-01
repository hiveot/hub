package router

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/digitwin/store"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/connections"
	"github.com/teris-io/shortid"
	"sync"
)

// ActionHandler is the API for service action handling
type ActionHandler func(msg *transports.ThingMessage) (stat transports.RequestStatus)

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
	digitwinAction ActionHandler
	dtwService     *service.DigitwinService
	authnAction    ActionHandler
	authzAction    ActionHandler
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
	msg *transports.ThingMessage, replyTo transports.IServerConnection) (
	stat transports.RequestStatus) {

	var isHandled = true

	// middleware: authorize the request.
	// TODO: use a middleware chain
	if !svc.hasPermission(msg.SenderID, msg.Operation, msg.ThingID) {
		stat.Failed(msg, fmt.Errorf("Unauthorized"))
	}

	// first handle send-and-forget operations (agent publications)
	switch msg.Operation {
	// operations with nothing to reply
	case vocab.HTOpPublishEvent:
		svc.HandlePublishEvent(msg)
	case vocab.HTOpUpdateProperty:
		svc.HandleUpdateProperty(msg)
	case vocab.HTOpUpdateMultipleProperties:
		svc.HandleUpdateMultipleProperties(msg)
	case vocab.HTOpUpdateActionStatus, vocab.HTOpUpdateActionStatuses:
		svc.HandleUpdateActionStatus(msg)
	case vocab.HTOpUpdateTD:
		svc.HandleUpdateTD(msg)
	default:
		isHandled = false
	}
	if isHandled {
		stat.Completed(msg, nil, nil)
		return stat
	}

	if msg.CorrelationID == "" {
		msg.CorrelationID = "action-" + shortid.MustGenerate()
	}
	// TODO: use a middleware chain
	svc.hasPermission(msg.SenderID, msg.Operation, msg.ThingID)

	switch msg.Operation {
	case vocab.OpInvokeAction:
		stat = svc.HandleInvokeAction(msg, replyTo.GetConnectionID())
	case vocab.HTOpLogin:
		stat = svc.HandleLogin(msg)
	case vocab.HTOpLogout:
		stat = svc.HandleLogout(msg)
	case vocab.HTOpRefresh:
		stat = svc.HandleLoginRefresh(msg)
	case vocab.HTOpReadAllEvents:
		stat = svc.HandleReadAllEvents(msg)
	case vocab.OpReadAllProperties:
		stat = svc.HandleReadAllProperties(msg)
	case vocab.HTOpReadEvent:
		stat = svc.HandleReadEvent(msg)
	case vocab.OpReadProperty:
		stat = svc.HandleReadProperty(msg)
	case vocab.OpWriteProperty:
		stat = svc.HandleWriteProperty(msg, replyTo.GetConnectionID())
	default:
		err := fmt.Errorf("Unknown operation '%s' from client '%s'", msg.Operation, msg.SenderID)
		stat.Failed(msg, err)
	}
	return stat
}

//// HandleMessage routes updates from agents to consumers
//func (svc *DigitwinRouter) HandleMessage(msg *transports.ThingMessage) {
//
//	// middleware: authorize the request.
//	// TODO: use a middleware chain
//	svc.hasPermission(msg.SenderID, msg.Operation, msg.ThingID)
//
//	switch msg.Operation {
//	case vocab.HTOpPublishEvent:
//		svc.HandlePublishEvent(msg)
//	case vocab.HTOpUpdateProperty:
//		svc.HandleUpdateProperty(msg)
//	case vocab.HTOpUpdateMultipleProperties:
//		svc.HandleUpdateMultipleProperties(msg)
//	case vocab.HTOpUpdateActionStatus, vocab.HTOpUpdateActionStatuses:
//		svc.HandleUpdateActionStatus(msg)
//	case vocab.HTOpUpdateTD:
//		svc.HandleUpdateTD(msg)
//	}
//}
//
//// HandleRequest routers requests from consumers to the digital twin and on to agents
//// The clcid is the client connectionID used when sending an asynchronous reply.
//func (svc *DigitwinRouter) HandleRequest(
//	request *transports.ThingMessage, replyTo string) (stat transports.RequestStatus) {
//	// assign a requestID if none given
//	if request.CorrelationID == "" {
//		request.CorrelationID = "action-" + shortid.MustGenerate()
//	}
//	// TODO: use a middleware chain
//	svc.hasPermission(request.SenderID, request.Operation, request.ThingID)
//
//	switch request.Operation {
//	case vocab.OpInvokeAction:
//		stat = svc.HandleInvokeAction(request, replyTo)
//	case vocab.HTOpLogin:
//		stat = svc.HandleLogin(request)
//	case vocab.HTOpLogout:
//		stat = svc.HandleLogout(request)
//	case vocab.HTOpRefresh:
//		stat = svc.HandleLoginRefresh(request)
//	case vocab.HTOpReadAllEvents:
//		stat = svc.HandleReadAllEvents(request)
//	case vocab.OpReadAllProperties:
//		stat = svc.HandleReadAllProperties(request)
//	case vocab.HTOpReadEvent:
//		stat = svc.HandleReadEvent(request)
//	case vocab.OpReadProperty:
//		stat = svc.HandleReadProperty(request)
//	case vocab.OpWriteProperty:
//		stat = svc.HandleWriteProperty(request, replyTo)
//	}
//	return stat
//}

// NewDigitwinRouter instantiates a new hub messaging router
// Use SetTransport to link to a transport for forwarding messages to
// agents and consumers.
//
//	dtwStore is used to update the digital twin status
//	tb is the transport binding for forwarding service requests
func NewDigitwinRouter(
	dtwService *service.DigitwinService,
	digitwinAction ActionHandler,
	authnAgent ActionHandler,
	authzAgent ActionHandler,
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
