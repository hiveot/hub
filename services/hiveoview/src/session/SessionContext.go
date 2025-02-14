package session

import (
	"context"
	"errors"
	"github.com/teris-io/shortid"
	"log/slog"
	"net/http"
	"time"
)

const ClientSessionContextID = "session"
const SessionManagerContextID = "sm"

// AddSessionToContext middleware adds a WebClientSession object to the request context.
//
// This uses the bearer token to identify the client ID and derive the connection-ID
// using the cid header:  connectionID := {clientID}-{cid}
//
// If no valid session is found and no cookie with auth token is present then
// invoke SessionLogout.
func AddSessionToContext(sm *WebSessionManager) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//var hc transports.IConsumer

			// get the current connection object
			cs, clientID, cid, authToken, err := sm.GetSessionFromCookie(r)
			if err != nil {
				// FIXME: if this is an SSE connection request then ask to back off
				slog.Warn("Hiveoview.AddSessionToContext: No valid authentication. Redirect to login.",
					slog.String("remoteAdd", r.RemoteAddr),
					slog.String("path", r.URL.String()),
					slog.String("err", err.Error()))

				// FIXME: the sse connection needs the login session cookie; either that or fix bearer token auth

				//// set retry to a large number
				//// see https://javascript.info/server-sent-events#reconnection
				//errMsg := fmt.Sprintf("retry: %s\nevent:%s\n\n",
				//	"10000", "logout")
				//http.Error(w, errMsg, http.StatusUnauthorized)
				//w.(http.Flusher).Flush()
				time.Sleep(time.Second)
				SessionLogout(w, r)
				return
			}
			if cs != nil {
				if !cs.IsActive() {
					// this should normally not happen unless logout is called
					slog.Error("Session available but it is disconnected from the hub")
					http.Error(w, "session has no hub connection", http.StatusInternalServerError)
					// this should normally not happening unless logout is called
					return
				}
				// active session exists. make it available in the context
				ctx := context.WithValue(r.Context(), SessionManagerContextID, sm)
				ctx = context.WithValue(ctx, ClientSessionContextID, cs)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			// Session doesn't exist with the given connection ID.
			// generate a CID if one doesnt exist.
			// Open a new session and reconnect to the hub using the given auth token
			// This can also be the result of an SSE reconnect, in which case the followup Serve
			// will link to this session.
			cid = "WC-" + shortid.MustGenerate()
			slog.Info("AddSessionToContext. New webclient session. Authenticate using its bearer token",
				slog.String("clientID", clientID),
				slog.String("cid", cid),
				slog.String("remoteAddr", r.RemoteAddr),
				slog.String("Method", r.Method),
				slog.String("RequestURI", r.RequestURI),
			)
			cs, err = sm.ConnectWithToken(w, r, clientID, cid, authToken)
			if err != nil {
				slog.Warn("AddSessionToContext: Session is no longer valid. Redirect to login.",
					slog.String("remoteAdd", r.RemoteAddr),
					slog.String("url", r.URL.String()))
				time.Sleep(time.Second)
				SessionLogout(w, r)
				return
			}
			//status := cs.GetStatus()
			//slog.Info("found session",
			//	slog.String("clientID", status.SenderID),
			//	slog.String("connected", string(status.ConnectionStatus)))
			if !cs.IsActive() {
				slog.Error("Session just added but shows as disconnected from the hub")
				http.Error(w, "session has no hub connection", http.StatusInternalServerError)
				return
			}

			// make the session is available through the context
			ctx := context.WithValue(r.Context(), SessionManagerContextID, sm)
			ctx = context.WithValue(ctx, ClientSessionContextID, cs)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

//

// GetSessionFromContext returns the session manager and client session instance
// from the request context.
// This returns an error if the session manager is not found.
// Intended for use by http handlers.
// Note: This should not be used in an SSE session.
func GetSessionFromContext(r *http.Request) (
	*WebSessionManager, *WebClientSession, error) {

	ctxSessionManager := r.Context().Value(SessionManagerContextID)
	ctxClientSession := r.Context().Value(ClientSessionContextID)
	if ctxSessionManager == nil || ctxClientSession == nil {
		return nil, nil, errors.New("no session in context")
	}
	clientSession := ctxClientSession.(*WebClientSession)
	sm := ctxSessionManager.(*WebSessionManager)
	//hc := session.GetConsumer()
	return sm, clientSession, nil
}
