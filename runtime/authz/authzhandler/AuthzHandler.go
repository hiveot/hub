package authzhandler

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authz"
	"github.com/hiveot/hub/runtime/router"
)

// AuthzHandler serves the message based interface to the authz service API.
// This converts the request messages into API calls and converts the result
// back to a reply message, if any.
// The main entry point is the HandleMessage function.
type AuthzHandler struct {
	svc *authz.AuthzService
}

// HandleMessage an event or action message for the authz service
func (h *AuthzHandler) HandleMessage(msg *things.ThingMessage) ([]byte, error) {
	if msg.MessageType == vocab.MessageTypeAction {
		// handle authn admin actions
		// handle authn client actions
		switch msg.Key {
		case api.GetClientRoleMethod:
			return h.GetClientRole(msg)
		case api.SetClientRoleMethod:
			return h.SetClientRole(msg)
		}
	}
	return nil, fmt.Errorf("Unknown action '%s' for service '%s'", msg.Key, msg.ThingID)
}

func (h *AuthzHandler) GetClientRole(
	msg *things.ThingMessage) (reply []byte, err error) {

	var args api.GetClientRoleArgs
	err = json.Unmarshal(msg.Data, &args)
	if err == nil {
		role, err2 := h.svc.GetClientRole(args.ClientID)
		err = err2
		if err == nil {
			resp := api.GetClientRoleResp{ClientID: args.ClientID, Role: role}
			reply, err = json.Marshal(resp)
		}
	}
	return reply, err
}

func (h *AuthzHandler) SetClientRole(
	msg *things.ThingMessage) (reply []byte, err error) {
	var args api.SetClientRoleArgs
	err = json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.SetClientRole(args.ClientID, args.Role)
	}
	return nil, err
}

// NewAuthzHandler creates a new instance of the messaging handler for the
// authorization service.
func NewAuthzHandler(svc *authz.AuthzService) router.MessageHandler {
	decoder := AuthzHandler{svc: svc}
	return decoder.HandleMessage
}
