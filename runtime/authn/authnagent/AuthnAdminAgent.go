// Package authnsrv serves authn messages to the service API
package authnagent

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn/service"
)

// AuthnAdminHandler serves the message based interface to the authn service API.
// This converts the request messages into API calls and converts the result
// back to a reply message, if any.
// The main entry point is the HandleMessage function.
type AuthnAdminHandler struct {
	svc *service.AuthnAdminService
}

// HandleMessage an event or action message for the authn admin service
// This handle action messages with the AuthnManageServiceID ThingID.
func (h *AuthnAdminHandler) HandleMessage(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	if msg.MessageType == vocab.MessageTypeAction {
		// handle authn admin actions
		// handle authn client actions
		switch msg.Key {
		case api.AddAgentMethod:
			return h.AddAgent(msg)
		case api.AddConsumerMethod:
			return h.AddConsumer(msg)
		case api.AddServiceMethod:
			return h.AddService(msg)
		case api.GetClientProfileMethod:
			return h.GetClientProfile(msg)
		case api.GetProfilesMethod:
			return h.GetProfiles(msg)
		case api.NewAuthTokenMethod:
			return h.NewAuthToken(msg)
		case api.RemoveClientMethod:
			return h.RemoveClient(msg)
		case api.SetClientPasswordMethod:
			return h.SetClientPassword(msg)
		case api.UpdateClientProfileMethod:
			return h.UpdateClientProfile(msg)
		}
	}
	err := fmt.Errorf("unknown action '%s' for service '%s'", msg.Key, msg.ThingID)
	stat.Failed(msg, err)
	return stat
}

func (h *AuthnAdminHandler) AddAgent(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	var args api.AddAgentArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		token, err2 := h.svc.AddAgent(args.AgentID, args.DisplayName, args.PubKey)
		err = err2
		if err == nil {
			resp := api.AddAgentResp{Token: token}
			stat.Reply, err = json.Marshal(resp)
		}
	}
	stat.Completed(msg, err)
	return stat
}
func (h *AuthnAdminHandler) AddConsumer(
	msg *things.ThingMessage) (stat api.DeliveryStatus) {

	var args api.AddConsumerArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.AddConsumer(args.ClientID, args.DisplayName, args.Password)
	}
	stat.Completed(msg, err)
	return stat
}

func (h *AuthnAdminHandler) AddService(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	var args api.AddServiceArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		token, err2 := h.svc.AddService(args.AgentID, args.DisplayName, args.PubKey)
		err = err2
		if err == nil {
			resp := api.AddServiceResp{Token: token}
			stat.Reply, err = json.Marshal(resp)
		}
	}
	stat.Completed(msg, err)
	return stat
}

func (h *AuthnAdminHandler) GetClientProfile(
	msg *things.ThingMessage) (stat api.DeliveryStatus) {

	var args api.GetClientProfileArgs
	err := json.Unmarshal(msg.Data, &args)

	if err == nil {
		prof, err2 := h.svc.GetClientProfile(args.ClientID)
		err = err2
		if err == nil {
			resp := api.GetClientProfileResp{Profile: prof}
			stat.Reply, err = json.Marshal(resp)
		}
	}
	stat.Completed(msg, err)
	return stat
}

func (h *AuthnAdminHandler) GetProfiles(
	msg *things.ThingMessage) (stat api.DeliveryStatus) {

	profList, err := h.svc.GetProfiles()
	if err == nil {
		resp := api.GetProfilesResp{Profiles: profList}
		stat.Reply, err = json.Marshal(resp)
	}
	stat.Completed(msg, err)
	return stat
}

func (h *AuthnAdminHandler) NewAuthToken(
	msg *things.ThingMessage) (stat api.DeliveryStatus) {
	var token string
	var args api.NewAuthTokenArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		token, err = h.svc.NewAuthToken(args.ClientID, args.ValiditySec)
		resp := api.NewAuthTokenResp{Token: token}
		stat.Reply, err = json.Marshal(resp)
	}
	stat.Completed(msg, err)
	return stat
}
func (h *AuthnAdminHandler) RemoveClient(
	msg *things.ThingMessage) (stat api.DeliveryStatus) {

	var args api.RemoveClientArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.RemoveClient(args.ClientID)
	}
	stat.Completed(msg, err)
	return stat
}

func (h *AuthnAdminHandler) SetClientPassword(
	msg *things.ThingMessage) (stat api.DeliveryStatus) {

	var args api.SetClientPasswordArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.SetClientPassword(args.ClientID, args.Password)
	}
	stat.Completed(msg, err)
	return stat
}

func (h *AuthnAdminHandler) UpdateClientProfile(
	msg *things.ThingMessage) (stat api.DeliveryStatus) {

	var args api.UpdateClientProfileArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.UpdateClientProfile(args.Profile)
	}
	stat.Completed(msg, err)
	return stat
}

// NewAuthnAdminHandler creates a new instance of the messaging handler for the
// authentication admin service.
func NewAuthnAdminHandler(svc *service.AuthnAdminService) api.MessageHandler {
	decoder := AuthnAdminHandler{svc: svc}
	return decoder.HandleMessage
}
