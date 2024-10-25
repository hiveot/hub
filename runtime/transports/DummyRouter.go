package transports

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime/api"
)

// DummyRouter for implementing test hooks defined in IHubRouter
type DummyRouter struct {
	OnAction func(senderID, dThingID, name string, val any, msgID string, cid string) any
	OnEvent  func(agentID, thingID, name string, val any, msgID string)

	authenticator api.IAuthenticator
}

func (svc *DummyRouter) HandleInvokeAction(
	senderID string, dThingID string, actionName string, input any, reqID string, cid string) (
	status string, output any, messageID string, err error) {
	// if a hook is provided, call it first
	if svc.OnAction != nil {
		output = svc.OnAction(senderID, dThingID, actionName, input, messageID, cid)
	}
	return vocab.ProgressStatusDelivered, output, reqID, nil
}

func (svc *DummyRouter) HandleInvokeActionProgress(agentID string, data any) error {
	return nil
}
func (svc *DummyRouter) HandleLogin(data any) (reply any, err error) {
	var args authn.UserLoginArgs
	utils.Decode(data, &args)
	token, err := svc.authenticator.Login(args.ClientID, args.Password)
	return token, err
}
func (svc *DummyRouter) HandleLoginRefresh(clientID string, data any) (reply any, err error) {
	var args authn.UserRefreshTokenArgs
	utils.Decode(data, &args)
	newToken, err := svc.authenticator.RefreshToken(clientID, args.ClientID, args.OldToken)
	return newToken, err
}
func (svc *DummyRouter) HandleLogout(clientID string) {
}

func (svc *DummyRouter) HandlePublishEvent(
	agentID string, thingID string, name string, value any, messageID string) error {
	if svc.OnEvent != nil {
		svc.OnEvent(agentID, thingID, name, value, messageID)
	}
	return nil
}

func (svc *DummyRouter) HandlePublishProperty(
	agentID string, thingID string, propName string, value any, reqID string) error {
	return nil
}
func (svc *DummyRouter) HandlePublishTD(agentID string, args any) error {
	return nil
}

// HandleQueryAction returns the action status
func (svc *DummyRouter) HandleQueryAction(
	consumerID string, dThingID string, name string) (reply any, err error) {
	return nil, nil
}

// HandleQueryAllActions returns the status of all actions of a Thing
func (svc *DummyRouter) HandleQueryAllActions(clientID string, dThingID string) (reply any, err error) {
	return nil, nil
}

// HandleReadEvent consumer reads a digital twin thing's event value
func (svc *DummyRouter) HandleReadEvent(consumerID string, dThingID string, name string) (reply any, err error) {
	return nil, nil
}

// HandleReadAllEvents consumer reads all digital twin thing's event values
func (svc *DummyRouter) HandleReadAllEvents(consumerID string, dThingID string) (reply any, err error) {
	return nil, nil
}

// HandleReadProperty consumer reads a digital twin thing's property value
func (svc *DummyRouter) HandleReadProperty(consumerID string, dThingID string, name string) (reply any, err error) {
	return nil, nil
}

// HandleReadAllProperties handles reading all digital twin thing's property values
func (svc *DummyRouter) HandleReadAllProperties(senderID string, dThingID string) (reply any, err error) {
	return nil, nil
}

// HandleReadTD consumer reads a digital twin thing's TD
func (svc *DummyRouter) HandleReadTD(consumerID string, args any) (reply any, err error) {
	return nil, nil
}

// HandleReadAllTDs consumer reads all digital twin thing's TD
func (svc *DummyRouter) HandleReadAllTDs(consumerID string) (reply any, err error) {
	return nil, nil
}

func (svc *DummyRouter) HandleWriteProperty(
	dThingID string, name string, newValue any, consumerID string) (
	status string, messageID string, err error) {

	return "", "", nil
}

func NewDummyRouter(authenticator api.IAuthenticator) *DummyRouter {
	return &DummyRouter{
		authenticator: authenticator,
	}
}
