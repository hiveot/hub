package middleware

import "github.com/hiveot/hub/lib/things"

// Pass an incoming event value through the middleware chain

// Q: should we use the http middleware API???
//    option 1: sounds nice, but how would that work?
//    option 2: might not work with messagebus requests
//    option 3: roll your own

// information required:
// - request address
// - request payload
// - senderID
// - extensible context
// - session info with custom fields
func Middleware(ev *things.ThingValue) {

}
