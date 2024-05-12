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
// This handle action messages with the AuthnAdminServiceID ThingID.
func (h *AuthnAdminHandler) HandleMessage(msg *things.ThingMessage) (stat api.DeliveryStatus) {
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
	stat.MessageID = msg.MessageID
	stat.Status = api.DeliveryFailed
	stat.Error = fmt.Sprintf("unknown action '%s' for service '%s'", msg.Key, msg.ThingID)
	return stat
}

func (h *AuthnAdminHandler) AddClient(
	msg *things.ThingMessage) (stat api.DeliveryStatus) {

	var args api.AddClientArgs

	stat.MessageID = msg.MessageID
	stat.Status = api.DeliveryCompleted
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.AddClient(args.ClientType, args.ClientID, args.DisplayName, args.PubKey, args.Password)
	}
	if err != nil {
		stat.Error = err.Error()
	}
	return stat
}

func (h *AuthnAdminHandler) GetClientProfile(
	msg *things.ThingMessage) (stat api.DeliveryStatus) {

	stat.MessageID = msg.MessageID
	stat.Status = api.DeliveryCompleted
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
	if err != nil {
		stat.Error = err.Error()
	}
	return stat
}

func (h *AuthnAdminHandler) GetProfiles(
	msg *things.ThingMessage) (stat api.DeliveryStatus) {

	stat.MessageID = msg.MessageID
	stat.Status = api.DeliveryCompleted
	profList, err := h.svc.GetProfiles()
	if err == nil {
		resp := api.GetProfilesResp{Profiles: profList}
		stat.Reply, err = json.Marshal(resp)
	}
	if err != nil {
		stat.Error = err.Error()
	}
	return stat
}

func (h *AuthnAdminHandler) RemoveClient(
	msg *things.ThingMessage) (stat api.DeliveryStatus) {

	stat.MessageID = msg.MessageID
	stat.Status = api.DeliveryCompleted
	var args api.RemoveClientArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.RemoveClient(args.ClientID)
	}
	if err != nil {
		stat.Error = err.Error()
	}
	return stat
}
func (h *AuthnAdminHandler) UpdateClientProfile(
	msg *things.ThingMessage) (stat api.DeliveryStatus) {

	stat.MessageID = msg.MessageID
	stat.Status = api.DeliveryCompleted
	var args api.UpdateClientProfileArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.UpdateClientProfile(args.Profile)
	}
	if err != nil {
		stat.Error = err.Error()
	}
	return stat
}
func (h *AuthnAdminHandler) UpdateClientPassword(
	msg *things.ThingMessage) (stat api.DeliveryStatus) {

	stat.MessageID = msg.MessageID
	stat.Status = api.DeliveryCompleted
	var args api.UpdateClientPasswordArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.UpdateClientPassword(args.ClientID, args.Password)
	}
	if err != nil {
		stat.Error = err.Error()
	}
	return stat
}

// NewAuthnAdminHandler creates a new instance of the messaging handler for the
// authentication admin service.
func NewAuthnAdminHandler(svc *service.AuthnAdminService) api.MessageHandler {
	decoder := AuthnAdminHandler{svc: svc}
	return decoder.HandleMessage
}
