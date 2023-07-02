package natsserver

import (
	"encoding/json"
	"errors"
	"github.com/hiveot/hub/api/go/thing"
	service2 "github.com/hiveot/hub/core/authn/service"
	"github.com/hiveot/hub/lib/hubclient"
)

const (
	ActionAddUser        = "addUser"
	ActionGetProfile     = "getProfile"
	ActionListClients    = "listClients"
	ActionLogin          = "login"
	ActionLogout         = "logout"
	ActionRefresh        = "refresh"
	ActionRemoveClient   = "removeClient"
	ActionResetPassword  = "resetPassword"
	ActionUpdateName     = "updateName"
	ActionUpdatePassword = "updatePassword"
)

// AuthnNatsServer is a NATS binding for authn service
// Subjects: things.authn.*.{action}
type AuthnNatsServer struct {
	service *service2.AuthnService
	hc      *hubclient.HubClient
}

func (natsrv *AuthnNatsServer) handleAction(subject string, clientID string, action *thing.ThingValue) {
	switch action.ID {
	case ActionAddUser:
		password := string(action.Data)
		err := natsrv.service.AddUser(clientID, password)
		natsrv.hc.SendReply(subject, err, "")
	case ActionGetProfile:
		prof, err := natsrv.service.GetProfile(clientID)
		profJson, _ := json.Marshal(prof)
		natsrv.hc.SendReply(subject, err, profJson)
		//
	case ActionListClients:
		clientList, err := natsrv.service.ListClients()
		clJson, _ := json.Marshal(clientList)
		natsrv.hc.SendReply(subject, err, clJson)
		//
	case ActionLogin:
		password := string(action.Data)
		token, err := natsrv.service.Login(clientID, password)
		natsrv.hc.SendReply(subject, err, token)
		//
	case ActionLogout:
		//
	case ActionRefresh:
		//
	case ActionRemoveClient:
		//
	case ActionResetPassword:
		//
	case ActionUpdateName:
		//
	case ActionUpdatePassword:
		//
	default:
		err := errors.New("invalid action '" + action.ID + "'")
		natsrv.hc.SendReply(subject, err, "")
	}
}

// Start subscribes to the actions
func (natsrv *AuthnNatsServer) Start() {
	_ = natsrv.hc.SubActions(natsrv.handleAction)
}

// Stop removes subscriptions
func (natsrv *AuthnNatsServer) Stop() {

}

// NewAuthnNats create a nats binding for the authn service
func NewAuthnNats(hc *hubclient.HubClient, svc *service2.AuthnService) *AuthnNatsServer {
	an := &AuthnNatsServer{
		service: svc,
		hc:      hc,
	}
	return an
}
