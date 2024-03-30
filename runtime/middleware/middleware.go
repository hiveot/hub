package middleware

import "github.com/hiveot/hub/lib/things"

// Middleware pass an incoming event value through the middleware chain
// The middleware chain is intended to validate, enrich, and process the event, action and rpc messages.
// Each middleware
func Middleware(ev *things.ThingValue) ([]byte, error) {
}
