package router

import (
	"fmt"
	"github.com/hiveot/hub/lib/things"
)

// RouterService for routing events, actions and rpc requests
// This is protocol agnostic and intended for use by any of the protocol bindings
//
// The RouterService maps the request address to the registered handler
type RouterService struct {
	cfg *RouterConfig
}

// Handler passes an incoming message through the middleware chain and on to the registered handlers.
//
// The middleware chain is intended to validate, enrich, and process the event, action and rpc messages.
//
// TODO: can this be standardized to using the http Handler and use the chi-co router?
//
//	Con: All protocol bindings would need to 'simulate' the http request and reply handler.
//	Pro: It allows existing middleware to be used, such as rate control.
func (svc *RouterService) Handler(ev *things.ThingValue) ([]byte, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// Start the middleware chain
func (svc *RouterService) Start() error {

	return nil
}

// Stop the middleware chain
func (svc *RouterService) Stop() {

}

// NewRouter creates a new instance of the message router.
// Call Start() to initialize it.
func NewRouter(cfg *RouterConfig) *RouterService {

	mw := &RouterService{
		cfg: cfg,
	}
	return mw
}
