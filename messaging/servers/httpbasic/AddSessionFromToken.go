package httpbasic

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/tputils/tlsserver"
)

// AddSessionFromToken middleware decodes the bearer session token in the authorization header.
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
// Non-session tokens, are used by services and device agents. These tokens are generated
// on provisioning or token renewal and last until their expiry.
//
// The session can be retrieved from the request context using GetSessionFromContext()
//
// The client session contains the client ID, and stats for the current session.
// If no valid session is found this will reply with an unauthorized status code.
//
// pubKey is the public key from the keypair used in creating the session token.
func AddSessionFromToken(userAuthn messaging.IAuthenticator) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			bearerToken, err := tlsserver.GetBearerToken(r)
			if err != nil {
				// see https://w3c.github.io/wot-discovery/#exploration-secboot
				// response with unauthorized and point to using the bearer token method
				errMsg := "AddSessionFromToken: " + err.Error()
				w.Header().Add("WWW-Authenticate", "Bearer")
				http.Error(w, errMsg, http.StatusUnauthorized)
				slog.Warn(errMsg)
				return
			}
			//check if the token is properly signed
			clientID, sid, err := userAuthn.ValidateToken(bearerToken)
			if err != nil || clientID == "" {
				w.Header().Add("WWW-Authenticate", "Bearer")
				http.Error(w, err.Error(), http.StatusUnauthorized)
				slog.Warn("AddSessionFromToken: Invalid session token:",
					"err", err, "clientID", clientID)
				return
			} else if clientID == "" {
				w.Header().Add("WWW-Authenticate", "Bearer")
				http.Error(w, "missing clientID", http.StatusUnauthorized)
				slog.Warn("AddSessionFromToken: Missing clientID")
				return
			}

			// make session available in context
			//ctx := context.WithValue(r.Context(), subprotocols.SessionContextID, cs)
			ctx := context.WithValue(r.Context(), SessionContextID, sid)
			ctx = context.WithValue(ctx, ContextClientID, clientID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
