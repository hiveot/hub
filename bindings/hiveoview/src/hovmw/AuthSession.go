package hovmw

import (
	"context"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"log/slog"
	"net/http"
)

// AuthSession middleware factory that authenticates a valid client session.
// Intended for secured routes that require a valid session.
//
// This middleware adds a request context value named "session", containing
// the SessionInfo instance of the authenticated client.
//
// The middleware returns a 404 if no valid session was found.
func AuthSession(sm *session.SessionManager) func(next http.Handler) http.Handler {
	// return the actual middleware, which needs access to the session manager
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var si *session.SessionInfo

			// determine the session key
			sessionCookie, err := r.Cookie("session")
			if err != nil {
				slog.Warn("unauthorized", "url", r.URL)
			} else {
				si, err = sm.GetSession(sessionCookie.Value)
				if err != nil {
					slog.Warn("not a valid session ", "sessionID", sessionCookie.Value)
				}
			}
			if err != nil {
				// unauthenticated, no session cookie was found
				w.WriteHeader(http.StatusUnauthorized)
				// TBD: where to redirect to login?
				return
			}

			// add the session to the context
			ctx := context.WithValue(r.Context(), "session", si)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
