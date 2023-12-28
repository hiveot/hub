package session

import (
	"context"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"log/slog"
	"net/http"
)

const SessionContextID = "session"

// GetClientSession returns the client session from the request context
// or nil if no client session was found
func GetClientSession(r *http.Request) *ClientSession {
	csAny := r.Context().Value(SessionContextID)
	if csAny == nil {
		return nil
	}
	return csAny.(*ClientSession)
}

// AuthenticateSession middleware redirects the request to login if no active session is found.
// An active session is a session that has a hub connection.
//
//	redirectPath is the URL to redirect to when auth fails or ""
func AuthenticateSession(redirectPath string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// from context
			cs := GetClientSession(r)

			if cs == nil {
				// no session was found, redirect
				http.Redirect(w, r, redirectPath, http.StatusFound)
				return
			}
			if cs.GetStatus().ConnectionStatus != transports.Connected {
				// session is not active, redirect
				// TODO: consider allowing a disconnected client as long as
				// a valid auth token is present. This allows to notify
				// the client that the connection has dropped.
				http.Redirect(w, r, redirectPath, http.StatusFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AddSessionToContext middleware adds a ClientSession object to the request context.
// This uses the session cookie to determine the sessionID. If no valid sessionID
// is found then the session context is not set.
//
// A valid session can exists that has no hub connection, so always check
// HubClient.IsConnected() before using this to publish/subscribe.
//
//	sm is the session manager used to lookup the session.
func AddSessionToContext(sm *SessionManager) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var cs *ClientSession

			// lookup the session if a cookie has the key
			// The name 'session' is used for both the cookie and the context and
			// is read in the 'login' flow.
			sessionCookie, err := r.Cookie("session")
			if err != nil {
				slog.Debug("no session cookie", "url", r.URL)
			} else {
				slog.Debug("found session", "sessionID", sessionCookie.Value)
				cs, err = sm.GetSession(sessionCookie.Value)
				if err != nil {
					slog.Warn("not a valid session ", "sessionID", sessionCookie.Value)
				}
			}

			if cs != nil {
				// make the session is available through the context
				ctx := context.WithValue(r.Context(), SessionContextID, cs)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}
