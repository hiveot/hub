// Package authnsrv serves authn messages to the service API
package authnhandler

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn/service"
	"github.com/hiveot/hub/runtime/router"
)

// AuthnAdminHandler serves the message based interface to the authn service API.
// This converts the request messages into API calls and converts the result
// back to a reply message, if any.
// The main entry point is the HandleMessage function.
type AuthnAdminHandler struct {
	svc *service.AuthnAdminService
}

// HandleMessage an event or action message for the authn admin service
// This handle action messages with the AuthnAdminServiceID ThingID.
func (h *AuthnAdminHandler) HandleMessage(msg *things.ThingMessage) ([]byte, error) {
	if msg.MessageType == vocab.MessageTypeAction {
		// handle authn admin actions
		// handle authn client actions
		switch msg.Key {
		case api.AddClientMethod:
			return h.AddClient(msg)
		case api.GetClientProfileMethod:
			return h.GetClientProfile(msg)
		case api.GetProfilesMethod:
			return h.GetProfiles(msg)
		case api.RemoveClientMethod:
			return h.RemoveClient(msg)
		case api.UpdateClientProfileMethod:
			return h.UpdateClientProfile(msg)
		case api.UpdateClientPasswordMethod:
			return h.UpdateClientPassword(msg)
		}
	}
	return nil, fmt.Errorf("Unknown action '%s' for service '%s'", msg.Key, msg.ThingID)
}

func (h *AuthnAdminHandler) AddClient(msg *things.ThingMessage) ([]byte, error) {
	var args api.AddClientArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.AddClient(args.ClientType, args.ClientID, args.DisplayName, args.PubKey, args.Password)
	}
	return nil, err
}

func (h *AuthnAdminHandler) GetClientProfile(msg *things.ThingMessage) (reply []byte, err error) {
	var args api.GetClientProfileArgs
	err = json.Unmarshal(msg.Data, &args)
	if err == nil {
		prof, err2 := h.svc.GetClientProfile(args.ClientID)
		err = err2
		if err == nil {
			resp := api.GetClientProfileResp{Profile: prof}
			reply, err = json.Marshal(resp)
		}
	}
	return reply, err
}

func (h *AuthnAdminHandler) GetProfiles(msg *things.ThingMessage) (reply []byte, err error) {
	profList, err := h.svc.GetProfiles()
	if err == nil {
		resp := api.GetProfilesResp{Profiles: profList}
		reply, err = json.Marshal(resp)
	}
	return reply, err
}

func (h *AuthnAdminHandler) RemoveClient(msg *things.ThingMessage) ([]byte, error) {
	var args api.RemoveClientArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.RemoveClient(args.ClientID)
	}
	return nil, err
}
func (h *AuthnAdminHandler) UpdateClientProfile(msg *things.ThingMessage) ([]byte, error) {
	var args api.UpdateClientProfileArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.UpdateClientProfile(args.Profile)
	}
	return nil, err
}
func (h *AuthnAdminHandler) UpdateClientPassword(msg *things.ThingMessage) ([]byte, error) {
	var args api.UpdateClientPasswordArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.UpdateClientPassword(args.ClientID, args.Password)
	}
	return nil, err
}

// NewAuthnAdminHandler creates a new instance of the messaging handler for the
// authentication admin service. Intended to be used by the router or for testing.
func NewAuthnAdminHandler(svc *service.AuthnAdminService) router.MessageHandler {
	decoder := AuthnAdminHandler{svc: svc}
	return decoder.HandleMessage
}
