package transports

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime/api"
)

// DummyRouter for implementing test hooks defined in IHubRouter
type DummyRouter struct {
	OnAction func(msg *hubclient.ThingMessage, replyTo string) hubclient.RequestStatus
	OnEvent  func(msg *hubclient.ThingMessage)

	authenticator api.IAuthenticator
}

func (svc *DummyRouter) HandleMessage(msg *hubclient.ThingMessage) {
	switch msg.Operation {
	case vocab.HTOpPublishEvent:
		svc.OnEvent(msg)
	}
}
func (svc *DummyRouter) HandleRequest(msg *hubclient.ThingMessage, replyTo string) (stat hubclient.RequestStatus) {
	stat.Failed(msg, fmt.Errorf("Unknown operation '%s'", msg.Operation))
	switch msg.Operation {
	case vocab.HTOpLogin:
		var args authn.UserLoginArgs
		utils.Decode(msg.Data, &args)
		token, err := svc.authenticator.Login(args.ClientID, args.Password)
		stat.Completed(msg, token, err)
	case vocab.HTOpRefresh:
		var args authn.UserRefreshTokenArgs
		utils.Decode(msg.Data, &args)
		newToken, err := svc.authenticator.RefreshToken(msg.SenderID, args.ClientID, args.OldToken)
		stat.Completed(msg, newToken, err)
	case vocab.HTOpLogout:
		svc.authenticator.Logout(msg.SenderID)
		stat.Completed(msg, nil, nil)
	case vocab.OpInvokeAction:
		// if a hook is provided, call it first
		if svc.OnAction != nil {
			stat = svc.OnAction(msg, replyTo)
		}
	}
	return stat
}

func NewDummyRouter(authenticator api.IAuthenticator) *DummyRouter {
	return &DummyRouter{
		authenticator: authenticator,
	}
}
