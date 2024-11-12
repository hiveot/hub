package httpcontext

import (
	"context"
	"github.com/hiveot/hub/lib/tlsserver"
	"github.com/hiveot/hub/runtime/api"
	"log/slog"
	"net/http"
)

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
func AddSessionFromToken(userAuthn api.IAuthenticator) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//var cs *sessions2.ClientSession
			bearerToken, err := tlsserver.GetBearerToken(r)
			if err != nil {
				errMsg := "AddSessionFromToken: " + err.Error()
				http.Error(w, errMsg, http.StatusUnauthorized)
				slog.Warn(errMsg)
				return
			}
			//check if the token is properly signed
			clientID, sid, err := userAuthn.ValidateToken(bearerToken)
			if err != nil || clientID == "" {
				errMsg := "Invalid session token"
				http.Error(w, errMsg, http.StatusUnauthorized)

				slog.Warn("Invalid session token:",
					"err", err, "clientID", clientID)
				return
			}

			if err != nil {
				// If no session is found then the session token is invalid. This can
				// happen after the user logs out.
				slog.Warn("Session is not valid:", "sid", sid, "err", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			//// the session must belong to the client
			//if cs.GetClientID() != clientID {
			//	slog.Error("AddSessionToContext: SenderID in session does not match jwt clientID",
			//		"jwt clientID", clientID,
			//		"session clientID", cs.GetClientID())
			//	w.WriteHeader(http.StatusUnauthorized)
			//	time.Sleep(time.Second)
			//	return
			//}
			// make session available in context
			//ctx := context.WithValue(r.Context(), subprotocols.SessionContextID, cs)
			ctx := context.WithValue(r.Context(), SessionContextID, sid)
			ctx = context.WithValue(ctx, ContextClientID, clientID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

//
//// GetSessionIdFromContext returns the session and clientID for the given request
////
//// This should not be used in an SSE session.
//func GetSessionIdFromContext(r *http.Request) (sessionID string, clientID string, err error) {
//	ctxSession := r.Context().Value(SessionContextID)
//	if ctxSession == nil {
//		return "", "", errors.New("no session in context")
//	}
//	cs := ctxSession.(*sessions2.ClientSession)
//	return cs.GetSessionID(), cs.GetClientID(), nil
//}
