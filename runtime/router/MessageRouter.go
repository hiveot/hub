package router

import (
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
)

// MessageHandler that processes an event or action type message and returns a result
type MessageHandler func(msg *things.ThingMessage) ([]byte, error)

// MiddlewareHandler for processing a message through the middleware chain
type MiddlewareHandler func(msg *things.ThingMessage) (*things.ThingMessage, error)

// MessageRouter is a simple router for events, actions and rpc request messages.
//
// This passes a message of a specific type to the registered middleware and on to the
// registered handler of the message type.
type MessageRouter struct {
	cfg *RouterConfig

	// middleware registered handlers
	mwChain []MiddlewareHandler //func(value *things.ThingMessage) (*things.ThingMessage, error)

	// Handler of hub messages by thingID/serviceID
	messageHandlers map[string]MessageHandler

	// Handler of events
	eventHandlers []MessageHandler
}

// AddEventHandler adds a handler for events
func (svc *MessageRouter) AddEventHandler(handler MessageHandler) {
	svc.eventHandlers = append(svc.eventHandlers, handler)
}

// AddServiceHandler adds a handler directed at a service.
// Only a single handler per serviceID (thingID) is allowed. Intended for
// embedded services.
// Use an empty thingID/serviceID and key to register the default handler.
func (svc *MessageRouter) AddServiceHandler(serviceID string, handler MessageHandler) {
	svc.messageHandlers[serviceID] = handler
}

// AddMiddlewareHandler appends handler to the middleware chain.
// Message will be passed in the order the handlers are added
func (svc *MessageRouter) AddMiddlewareHandler(handler MiddlewareHandler) {
	svc.mwChain = append(svc.mwChain, handler)
}

// HandleMessage passes an incoming message through the middleware chain and on to
// the registered handlers.
//
// Events are handled separate from targeted messages such as actions.
// Actions are passed to the handler registered for the thingID. If no handler
// exists then the default handler is invoked.
//
// The default handler can lookup the destination device and decide to queue or
// forward the request, based on the policy. (TODO)
//
// The middleware chain is intended to validate, enrich, and process the event, action and rpc messages.
func (svc *MessageRouter) HandleMessage(tv *things.ThingMessage) ([]byte, error) {
	var err error

	//
	for _, handler := range svc.mwChain {
		tv, err = handler(tv)
		if err != nil {
			return nil, err
		}
	}
	// events are handled separate from targeted messages
	if tv.MessageType == vocab.MessageTypeEvent {
		for _, h := range svc.eventHandlers {
			_, _ = h(tv)
		}
		return nil, nil
	}
	msgHandler, found := svc.messageHandlers[tv.ThingID]
	if !found {
		// check for the default handler
		msgHandler, found = svc.messageHandlers[""]
	}
	if !found {
		return nil, fmt.Errorf("no handler for messageType '%s', thingID '%s', key '%s' from sender '%s'",
			tv.MessageType, tv.ThingID, tv.Key, tv.SenderID)
	}
	reply, err := msgHandler(tv)
	return reply, err
}

// NewMessageRouter creates a new instance of the message router.
// Call Start() to initialize it.
func NewMessageRouter(cfg *RouterConfig) *MessageRouter {

	mw := &MessageRouter{
		cfg:             cfg,
		mwChain:         make([]MiddlewareHandler, 0),
		messageHandlers: make(map[string]MessageHandler),
		eventHandlers:   make([]MessageHandler, 0),
	}
	return mw
}
