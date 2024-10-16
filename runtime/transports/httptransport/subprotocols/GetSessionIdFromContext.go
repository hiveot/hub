package subprotocols

import (
	"errors"
	sessions2 "github.com/hiveot/hub/runtime/transports/sessions"
	"net/http"
)

const SessionContextID = "session"

// GetSessionIdFromContext returns the session and clientID for the given request
//
// This should not be used in an SSE session.
func GetSessionIdFromContext(r *http.Request) (sessionID string, clientID string, err error) {
	ctxSession := r.Context().Value(SessionContextID)
	if ctxSession == nil {
		return "", "", errors.New("no session in context")
	}
	cs := ctxSession.(*sessions2.ClientSession)
	return cs.GetSessionID(), cs.GetClientID(), nil
}
