package router

import (
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
)

// ActionHandler that processes an action type message and returns a result
type ActionHandler func(msg *things.ThingMessage) ([]byte, error)

// EventHandler that processes an event type message and returns a result
type EventHandler func(msg *things.ThingMessage) error

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

	// Handler of published events
	eventHandlers map[string]EventHandler
	// Handler of published actions
	actionHandlers map[string]ActionHandler
}

// AddActionHandler adds a handler for an action message.
// Action messages can be sent by both agents and consumers
//
// For RPC method calls, the thingID identifies the service ID and the key the method name.
// Use an empty thingID and key to register a default handler.
func (svc *MessageRouter) AddActionHandler(thingID string, key string, handler ActionHandler) {
	addr := fmt.Sprintf("%s/%s/%s", vocab.MessageTypeAction, thingID, key)
	svc.actionHandlers[addr] = handler
}

// AddEventHandler adds a handler for event messages send by from agents.
// Use an empty thingID and key to register a default handler.
func (svc *MessageRouter) AddEventHandler(thingID string, key string, handler EventHandler) {
	addr := fmt.Sprintf("%s/%s/%s", vocab.MessageTypeEvent, thingID, key)
	svc.eventHandlers[addr] = handler
}

// AddMiddlewareHandler appends handler to the middleware chain.
// Message will be passed in the order the handlers are added
func (svc *MessageRouter) AddMiddlewareHandler(handler MiddlewareHandler) {
	svc.mwChain = append(svc.mwChain, handler)
}

// HandleActionMessage passes an incoming action request through the middleware chain and on to
// the registered handlers.
// Intended for handling action requests from consumers and services.
//
// The middleware chain is intended to validate, enrich, and process the event, action and rpc messages.
//
// TODO: can this be standardized to using the http Handler and use the chi-co router?
//
//	Con: All protocol bindings would need to 'simulate' the http request and reply handler.
//	Pro: It allows existing middleware to be used, such as rate control.
func (svc *MessageRouter) HandleActionMessage(tv *things.ThingMessage) ([]byte, error) {
	var err error

	//
	for _, handler := range svc.mwChain {
		tv, err = handler(tv)
		if err != nil {
			return nil, err
		}
	}
	addr := fmt.Sprintf("%s/%s/%s", vocab.MessageTypeAction, tv.ThingID, tv.Key)

	pubHandler, found := svc.actionHandlers[addr]
	if !found {
		// use a default message handler if set
		addr = fmt.Sprintf("%s//", tv.MessageType)
		pubHandler, found = svc.actionHandlers[addr]
	}
	if !found {
		return nil, fmt.Errorf("no handler for messageType '%s', thingID '%s', key '%s' from sender '%s'",
			tv.MessageType, tv.ThingID, tv.Key, tv.SenderID)
	}
	return pubHandler(tv)
}

// HandleEventMessage passes an incoming event through the middleware chain and on to the registered handlers.
// Intended for handling events from the protocol bindings and services.
//
// The middleware chain is intended to validate, enrich, and process the event, action and rpc messages.
func (svc *MessageRouter) HandleEventMessage(tv *things.ThingMessage) error {
	var err error

	//
	for _, handler := range svc.mwChain {
		tv, err = handler(tv)
		if err != nil {
			return err
		}
	}
	addr := fmt.Sprintf("%s/%s/%s", vocab.MessageTypeEvent, tv.ThingID, tv.Key)
	pubHandler, found := svc.eventHandlers[addr]
	if !found {
		// use a default message handler if set
		addr = fmt.Sprintf("%s//", tv.MessageType)
		pubHandler, found = svc.eventHandlers[addr]
	}
	if !found {
		return fmt.Errorf("no handler for messageType '%s', thingID '%s', key '%s' from sender '%s'",
			tv.MessageType, tv.ThingID, tv.Key, tv.SenderID)
	}
	return pubHandler(tv)
}

// HandleMessage passes an incoming event or action message through the middleware chain and on to
// the registered handlers.
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
	addr := fmt.Sprintf("%s/%s/%s", vocab.MessageTypeAction, tv.ThingID, tv.Key)

	pubHandler, found := svc.actionHandlers[addr]
	if !found {
		// use a default message handler if set
		addr = fmt.Sprintf("%s//", tv.MessageType)
		pubHandler, found = svc.actionHandlers[addr]
	}
	if !found {
		return nil, fmt.Errorf("no handler for messageType '%s', thingID '%s', key '%s' from sender '%s'",
			tv.MessageType, tv.ThingID, tv.Key, tv.SenderID)
	}
	return pubHandler(tv)
}

// NewMessageRouter creates a new instance of the message router.
// Call Start() to initialize it.
func NewMessageRouter(cfg *RouterConfig) *MessageRouter {

	mw := &MessageRouter{
		cfg:            cfg,
		mwChain:        make([]MiddlewareHandler, 0),
		actionHandlers: make(map[string]ActionHandler),
		eventHandlers:  make(map[string]EventHandler),
	}
	return mw
}
