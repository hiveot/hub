package middleware

import (
	"fmt"
	"github.com/hiveot/hub/lib/things"
)

// Middleware chain for handling events
type Middleware struct {
	cfg        *MiddlewareConfig
	auth       *Auth
	dir        *Directory
	valueStore *ValueStore
}

// Handler passes an incoming message through the middleware chain
// The middleware chain is intended to validate, enrich, and process the event, action and rpc messages.
//
// TODO: can this be standardized to using the http Handler?
// All protocol bindings then need to 'simulate' the http request and reply handler.
// It allows existing middleware to be used, such as rate control.
func (mw *Middleware) Handler(ev *things.ThingValue) ([]byte, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// Start the middleware chain
func (mw *Middleware) Start() {

}

// Stop the middleware chain
func (mw *Middleware) Stop() {

}

// NewMiddleware creates a new instance of the middleware chain.
// Call Start() to initialize it.
func NewMiddleware(cfg *MiddlewareConfig,
	auth *authn.AuthService,
	dir *directory.DirectoryService,
	valueStore *valuestore.ValueStore) *Middleware {

	mw := &Middleware{
		auth:       auth,
		dir:        dir,
		valueStore: valueStore,
		cfg:        cfg,
	}
	return mw
}
