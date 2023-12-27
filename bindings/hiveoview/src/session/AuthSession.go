package session

import (
	"context"
	"log/slog"
	"net/http"
)

// AuthSession middleware that authenticates a client's session.
// Intended for secured routes that require a valid session.
//
// This middleware adds a request context value named "session", containing
// the ClientSession instance of the authenticated client.
//
// This uses a cookie to store the sessionID.
// TODO: use a secure http cookie (after switching to TLS)
//
// If a valid session wasn't found this redirects to the loginPath and returns a 302 (redirect)
//
//	sm is the session manager used to validate the session.
//	loginPath is the URL to redirect to when auth fails
func AuthSession(sm *SessionManager, loginPath string) func(next http.Handler) http.Handler {
	// return the actual middleware, which needs access to the session manager
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var si *ClientSession

			// determine the session key
			sessionCookie, err := r.Cookie("session")
			if err != nil {
				slog.Warn("no session cookie", "url", r.URL)
			} else {
				slog.Info("found session", "sessionID", sessionCookie.Value)
				si, err = sm.GetSession(sessionCookie.Value)
				if err != nil {
					slog.Warn("not a valid session ", "sessionID", sessionCookie.Value)
				}
			}
			if err != nil {
				// no session cookie was found, redirect
				//w.WriteHeader(http.StatusUnauthorized)
				http.Redirect(w, r, loginPath, http.StatusFound)
				return
			}

			// add the session to the context
			ctx := context.WithValue(r.Context(), "session", si)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
