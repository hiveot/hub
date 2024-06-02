package sessions

import (
	"context"
	"errors"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/tlsserver"
	"log/slog"
	"net/http"
	"time"
)

const SessionContextID = "session"

// AddSessionFromToken middleware decodes the bearer session token in the authorization header
// and adds the corresponding ClientSession object to the request context.
//
// Session tokens can be provided through a bearer token or a client cookie. The token
// must match with an existing session ID.
//
// This distinguishes two types of tokens. Those with and those without a session ID.
// If the token contains a session ID then that session must exist or the token is invalid.
// User tokens are typically session tokens. Closing the session (logout) invalidates the token,
// even if it hasn't yet expired. Sessions are currently only stored in memory so a service
// restart also invalidates all session tokens.
//
// # Non-session tokens, are used by services and device agents. These tokens are generated
// on provisioning or token renewal and last until their expiry.
//
// The session can be retrieved from the request context using GetSessionFromContext()
//
// The client session contains the caller's ID, and stats for the current session.
// If no valid session is found this will reply with an unauthorized status code.
//
// pubKey is the public key from the keypair used in creating the session token.
func AddSessionFromToken(sessionAuth api.IAuthenticator) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var cs *ClientSession
			bearerToken, err := tlsserver.GetBearerToken(r)
			if err != nil {
				errMsg := "AddSessionFromToken: " + err.Error()
				http.Error(w, errMsg, http.StatusUnauthorized)
				slog.Warn(errMsg)
				return
			}
			//check if the token is properly signed
			cid, sid, err := sessionAuth.ValidateToken(bearerToken)
			if err != nil || cid == "" {
				errMsg := "Invalid session token"
				http.Error(w, errMsg, http.StatusUnauthorized)

				slog.Warn("Invalid session token:",
					"err", err, "clientID", cid)
				return
			}

			if sid == "" {
				// service/devices don't have a session-id in their token. These tokens
				// remain valid until they expire. Use the client-id as the session ID.
				cs, err = sessionmanager.GetSession(cid)
				if cs == nil {
					cs, err = sessionmanager.NewSession(cid, r.RemoteAddr, cid)
				}
			} else {
				// A token with session-id must have a known session
				cs, err = sessionmanager.GetSession(sid)
			}
			if err != nil {
				// If no session is found then the session token is invalid. This can
				// happen after the user logs out.
				slog.Warn("Session is not valid:", "sid", sid, "err", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			// the session must belong to the client
			if cs.clientID != cid {
				slog.Error("AddSessionToContext: ClientID in session does not match jwt clientID",
					"jwt clientID", cid,
					"session clientID", cs.clientID)
				w.WriteHeader(http.StatusUnauthorized)
				time.Sleep(time.Second)
				return
			}
			// make session available in context
			ctx := context.WithValue(r.Context(), SessionContextID, cs)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

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
