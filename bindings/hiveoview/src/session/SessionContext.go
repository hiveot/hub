package session

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

const SessionContextID = "session"

// AddSessionToContext middleware adds a ClientSession object to the request context.
//
// If no valid session is found and no cookie with auth token is present
// then HandleLogout is invoked.
func AddSessionToContext() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			cs, err := GetSession(w, r)
			if err != nil || !cs.IsActive() {
				slog.Warn("Request without an active session. Redirect to login.",
					slog.String("remoteAdd", r.RemoteAddr),
					slog.String("url", r.URL.String()))
				time.Sleep(time.Second)
				SessionLogout(w, r)
				return
			}

			status := cs.GetStatus()
			slog.Info("found session",
				slog.String("clientID", status.ClientID),
				slog.String("connected", string(status.ConnectionStatus)))

			// make the session is available through the context
			ctx := context.WithValue(r.Context(), SessionContextID, cs)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

//
