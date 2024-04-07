package router

import (
	"fmt"
	"github.com/hiveot/hub/lib/things"
)

// MessageHandler that processes a message and returns a result
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

	// Handler of published messages by message type
	handlers map[string]MessageHandler
}

// HandleMessage passes an incoming message through the middleware chain and on to the registered handlers.
// Intended for handling messages from the protocol bindings.
//
// The middleware chain is intended to validate, enrich, and process the event, action and rpc messages.
//
// TODO: can this be standardized to using the http Handler and use the chi-co router?
//
//	Con: All protocol bindings would need to 'simulate' the http request and reply handler.
//	Pro: It allows existing middleware to be used, such as rate control.
func (svc *MessageRouter) HandleMessage(tv *things.ThingMessage) ([]byte, error) {
	var err error
	for _, handler := range svc.mwChain {
		tv, err = handler(tv)
		if err != nil {
			return nil, err
		}
	}
	pubHandler, found := svc.handlers[tv.MessageType]
	if !found {
		return nil, fmt.Errorf("no handler for messageType '%s' from sender '%s'",
			tv.MessageType, tv.SenderID)
	}
	return pubHandler(tv)
}

// AddMessageTypeHandler adds a message handler for the given message type.
//
// Only a single handler per message type is supported. If a handler for the message type exists it is replaced.
func (svc *MessageRouter) AddMessageTypeHandler(msgType string, handler MessageHandler) {
	svc.handlers[msgType] = handler
}

// AddMiddlewareHandler appends handler to the middleware chain.
// Message will be passed in the order the handlers are added
func (svc *MessageRouter) AddMiddlewareHandler(handler MiddlewareHandler) {
	svc.mwChain = append(svc.mwChain, handler)
}

// NewMessageRouter creates a new instance of the message router.
// Call Start() to initialize it.
func NewMessageRouter(cfg *RouterConfig) *MessageRouter {

	mw := &MessageRouter{
		cfg:      cfg,
		mwChain:  make([]MiddlewareHandler, 0),
		handlers: make(map[string]MessageHandler),
	}
	return mw
}
