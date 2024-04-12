// Package httpsbinding with handling of messaging to and from services
package httpsbinding

import (
	"encoding/json"
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
	"io"
	"log/slog"
	"net/http"
)

// handlePostRPC handles a rpc request posted by a consumer
func (svc *HttpsBinding) handlePostRPC(w http.ResponseWriter, r *http.Request) {
	svc.onRequest(vocab.MessageTypeAction, w, r)
}

// handlePostRPC handles a login request posted by a consumer
func (svc *HttpsBinding) handlePostLogin(w http.ResponseWriter, r *http.Request) {
	// credentials are in a json payload
	authMsg := make(map[string]string)
	data, _ := io.ReadAll(r.Body)
	err := json.Unmarshal(data, &authMsg)
	if err != nil {
		slog.Warn("handleLogin failed getting credentials", "err", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	loginID := authMsg["login"]
	password := authMsg["password"]
	authToken, err := svc.sessionAuth.Login(loginID, password, "")
	if err != nil {
		// missing bearer token
		slog.Warn("handleLogin bad login", "clientID", loginID)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, _ = w.Write([]byte(authToken))
	w.WriteHeader(http.StatusOK)
}
func (svc *HttpsBinding) handlePostRefresh(w http.ResponseWriter, r *http.Request) {

}

// SendRPC an rpc requestmessage to the destination
// This returns an error if no session for the destination is available
func (svc *HttpsBinding) SendRPC(message *things.ThingMessage) ([]byte, error) {
	return nil, fmt.Errorf("not yet implemented")
	// TODO: track subscriptions
	// TODO: publish to SSE handlers of subscribed clients
}
