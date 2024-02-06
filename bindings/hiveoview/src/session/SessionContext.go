package session

import (
	"context"
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"log/slog"
	"net/http"
	"time"
)

const SessionContextID = "session"

// AddSessionToContext middleware adds a ClientSession object to the request context.
//
// If no valid session is found and no cookie with auth token is present
// then SessionLogout is invoked.
// If an inactive session is found then try to activate it using the cookie's auth token.
func AddSessionToContext() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var clientID string
			var hc *hubclient.HubClient

			// get the session
			cs, claims, err := sessionmanager.GetSessionFromCookie(r)
			if cs != nil && cs.IsActive() {
				// active session exists. make it available in the context
				ctx := context.WithValue(r.Context(), SessionContextID, cs)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			// session doesn't exist. Attempt to reconnect if claims exist.
			if claims == nil {
				err = fmt.Errorf("AddSessionToContext: session doesn't exist: %w", err)
			} else {
				clientID, _ = claims.GetSubject()
				authToken := claims.AuthToken

				hc, err = sessionmanager.ConnectWithToken(clientID, authToken)
			}
			// activate the session with the new connection
			if err == nil {
				cs, err = sessionmanager.ActivateNewSession(w, r, hc)
			}
			if err != nil {
				slog.Warn("AddSessionToContext: Request without an auth cookie. Redirect to login.",
					slog.String("remoteAdd", r.RemoteAddr),
					slog.String("url", r.URL.String()))
				time.Sleep(time.Second)
				SessionLogout(w, r)
				return
			}

			//status := cs.GetStatus()
			//slog.Info("found session",
			//	slog.String("clientID", status.ClientID),
			//	slog.String("connected", string(status.ConnectionStatus)))

			// make the session is available through the context
			ctx := context.WithValue(r.Context(), SessionContextID, cs)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

//

// GetSessionFromContext returns the session object for the given request from the context.
//
// This should not be used in an SSE session.
func GetSessionFromContext(r *http.Request) (*ClientSession, error) {
	ctxSession := r.Context().Value(SessionContextID)
	if ctxSession == nil {
		return nil, errors.New("no session in context")
	}
	return ctxSession.(*ClientSession), nil
}
