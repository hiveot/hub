package httpcontext

import (
	"errors"
	"net/http"
)

const SessionContextID = "session"
const ContextClientID = "clientID"

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
