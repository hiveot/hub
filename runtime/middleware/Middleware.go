package middleware

import (
	"fmt"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
)

// MiddlewareHandler for processing a message through the middleware chain until an error
// is returned.
// TODO: add functions for rate control and authorization
type MiddlewareHandler func(msg *things.ThingMessage) (*things.ThingMessage, error)

// MessageHandler handles the message after the middleware
type MessageHandler func(msg *things.ThingMessage) api.DeliveryStatus

// Middleware service for passing events and actions through a chain of services
type Middleware struct {
	// the receiver of messages that have passed through the middleware
	handler MessageHandler

	// middleware registered handlers
	mwChain []MiddlewareHandler //func(value *things.ThingMessage) (*things.ThingMessage, error)

}

// AddMiddlewareHandler appends handler to the middleware chain.
// Message will be passed in the order the handlers are added
func (svc *Middleware) AddMiddlewareHandler(handler MiddlewareHandler) {
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
func (svc *Middleware) HandleMessage(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	var err error

	for _, handler := range svc.mwChain {
		msg, err = handler(msg)
		if err != nil {
			stat.Failed(msg, err)
			return stat
		}
	}
	if svc.handler != nil {
		return svc.handler(msg)
	}
	err = fmt.Errorf("No handler for messages is set")
	stat.Failed(msg, err)
	return stat
}

// SetMessageHandler sets the handler for messages that are passed through
// the middleware chain.
func (mw *Middleware) SetMessageHandler(handler MessageHandler) {
	mw.handler = handler
}

// NewMiddleware creates a new instance of the messaging middleware chain.
// The message handler will process the message on success
func NewMiddleware() *Middleware {

	mw := &Middleware{
		mwChain: make([]MiddlewareHandler, 0),
	}
	return mw
}
