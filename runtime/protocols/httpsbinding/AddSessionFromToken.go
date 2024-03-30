package httpsbinding

import (
	"context"
	"errors"
	"fmt"
	"github.com/hiveot/hub/runtime/authn/jwtauth"
	"log/slog"
	"net/http"
	"time"
)

const SessionContextID = "session"

// AddSessionFromToken middleware decodes the bearer session token in the authorization header
// and adds the corresponding ClientSession object to the request context.
//
// The session can be retrieved from the request context using GetSessionFromContext()
//
// The client session contains the caller's ID, and stats for the current session.
// If no valid session is found this will reply with an unauthorized status code.
//
// pubKey is the public key from the keypair used in creating the session token.
func AddSessionFromToken(pubKey interface{}) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			tokenString := r.Header.Get("Authorization")
			if tokenString == "" {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(w, "Missing authorization header")
				return
			}
			tokenString = tokenString[len("Bearer "):]
			//
			//check if the token is properly signed
			cid, sid, err := jwtauth.DecodeSessionToken(tokenString, pubKey, "", "")
			if err != nil {
				slog.Warn("Invalid session token:", "err", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// next obtain the client and session IDs from the claims
			cs, err := sessionmanager.GetSession(sid)
			if err != nil {
				slog.Warn("DecodeSessionToken: Request without a valid session. Please re-authenticate.",
					slog.String("remoteAdd", r.RemoteAddr),
					slog.String("url", r.URL.String()))
				w.WriteHeader(http.StatusUnauthorized)
				time.Sleep(time.Second)
				return
			}

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
