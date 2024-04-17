// Package authnmsg with server side messaging structs for use by clients
package authnhandler

import (
	"encoding/json"
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn/service"
	"github.com/hiveot/hub/runtime/router"
)

// AuthnUserHandler is the server side (un)marshaller for user messages.
// This converts the request messages into API calls and converts the result
// back to a reply message, if any.
// The main entry point is the HandleMessage function.
type AuthnUserHandler struct {
	svc *service.AuthnUserService
}

// HandleMessage an event or action message for the authn admin service
// This handle action messages with the AuthnAdminServiceID ThingID.
func (h *AuthnUserHandler) HandleMessage(msg *things.ThingMessage) ([]byte, error) {
	if msg.MessageType == vocab.MessageTypeAction {
		// handle authn client actions
		switch msg.Key {
		case api.GetProfileMethod:
			return h.HandleGetProfile(msg)
		case api.LoginMethod:
			return h.HandleLogin(msg)
		case api.RefreshTokenMethod:
			return h.HandleRefresh(msg)
		case api.UpdatePasswordMethod:
			return h.HandleUpdatePassword(msg)
		case api.UpdateNameMethod:
			return h.HandleUpdateName(msg)
		case api.UpdatePubKeyMethod:
			return h.HandleUpdatePubKey(msg)
		}
	}
	return nil, fmt.Errorf("Unknown action '%s' for service '%s'", msg.Key, msg.ThingID)
}
func (h *AuthnUserHandler) HandleGetProfile(
	msg *things.ThingMessage) (reply []byte, err error) {
	prof, err := h.svc.GetProfile(msg.SenderID)
	if err == nil {
		resp := api.GetProfileResp{Profile: prof}
		reply, err = json.Marshal(resp)
	}
	return reply, err
}

// FIXME: support sessionID in login token through context.
func (h *AuthnUserHandler) HandleLogin(
	msg *things.ThingMessage) (reply []byte, err error) {

	args := api.LoginArgs{}
	err = json.Unmarshal(msg.Data, &args)
	if err == nil {
		token, err2 := h.svc.Login(args.ClientID, args.Password, "")
		err = err2
		if err == nil {
			resp := api.LoginResp{Token: token}
			reply, err = json.Marshal(resp)
		}
	}
	return reply, err
}

func (h *AuthnUserHandler) HandleRefresh(
	msg *things.ThingMessage) (reply []byte, err error) {
	args := api.RefreshTokenArgs{}
	err = json.Unmarshal(msg.Data, &args)
	if err == nil {
		newToken, err2 := h.svc.RefreshToken(msg.SenderID, args.OldToken)
		err = err2
		if err == nil {
			resp := api.RefreshTokenResp{Token: newToken}
			reply, err = json.Marshal(resp)
		}
	}
	return reply, err
}
func (h *AuthnUserHandler) HandleUpdatePassword(msg *things.ThingMessage) ([]byte, error) {
	args := api.UpdatePasswordArgs{}
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.UpdatePassword(msg.SenderID, args.NewPassword)
	}
	return nil, err
}
func (h *AuthnUserHandler) HandleUpdateName(msg *things.ThingMessage) ([]byte, error) {
	args := api.UpdateNameArgs{}
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.UpdateName(msg.SenderID, args.NewName)
	}
	return nil, err
}

func (h *AuthnUserHandler) HandleUpdatePubKey(msg *things.ThingMessage) ([]byte, error) {
	var args api.UpdatePubKeyArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.UpdatePubKey(msg.SenderID, args.PubKeyPem)
	}
	return nil, err
}

// NewAuthnUserHandler creates a new instance of the messaging handler for the
// authentication admin service. Intended to be used by the router or for testing.
func NewAuthnUserHandler(svc *service.AuthnUserService) router.MessageHandler {
	decoder := AuthnUserHandler{svc: svc}
	return decoder.HandleMessage
}
