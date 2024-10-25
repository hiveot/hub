package subprotocols

import (
	"errors"
	"net/http"
)

const SessionContextID = "session"
const ContextClientID = "clientID"

// GetSessionIdFromContext returns the session and clientID for the given request
//
// This should not be used in an SSE session.
//func GetSessionIdFromContext(r *http.Request) (sessionID string, clientID string, err error) {
//	ctxSession := r.Context().Value(SessionContextID)
//	if ctxSession == nil {
//		return "", "", errors.New("no session in context")
//	}
//	cs := ctxSession.(*sessions2.ClientSession)
//	return cs.SessionID, cs.ClientID, nil
//}

// GetClientIdFromContext returns the clientID for the given request
//
// This should not be used in an SSE session.
func GetClientIdFromContext(r *http.Request) (clientID string, err error) {
	ctxClientID := r.Context().Value(ContextClientID)
	if ctxClientID == nil {
		return "", errors.New("no session in context")
	}
	clientID = ctxClientID.(string)
	return clientID, nil
}
