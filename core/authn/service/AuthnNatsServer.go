package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/hub"
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/lib/ser"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/slog"
	"time"
)

//
//const (
//	ActionAddUser        = "addUser"
//	ActionGetProfile     = "getProfile"
//	ActionListClients    = "listClients"
//	ActionLogin          = "login"
//	ActionLogout         = "logout"
//	ActionRefresh        = "refresh"
//	ActionRemoveClient   = "removeClient"
//	ActionResetPassword  = "resetPassword"
//	ActionUpdateName     = "updateName"
//	ActionUpdatePassword = "updatePassword"
//)

// AuthnNatsServer is a NATS binding for authn service
// Subjects: things.authn.*.{action}
type AuthnNatsServer struct {
	service    *AuthnService
	hc         hub.IHubClient
	signingKey nkeys.KeyPair
}

// HandleCallOut for custom auth
// Using callout:https://github.com/nats-io/nats-architecture-and-design/blob/main/adr/ADR-26.md
//  0. The server will generate a server keypair and an xkey keypair on startup,
//     which will be used to sign and optionally encrypt the authorization requests.
//     Q: where to get these key pairs?
//  1. Callout issuer, is a public account nkey, in server configuration mode
//  2. The response must be signed by this issuer
//  3. The subject must be the public user nkey from the request
//     (In operator mode, the response must be signed by the account bound to the client)
//  4. the authorization service runs isolated in its own account,
//     Q: why not use the system account?
//  5. The request payload is a JWT claim containing:
//     - nats.client_info.id, kind, rtt, server, user, ver
//     - user in nats.client_info.user  - is this the same as below?
//     - password, if any, in client_opts["pass"]
//     - user in nats.client_opts.user  - is this the login id???
//     - TLS information if client certificates are used
//     - public user nkey for any response that the server accepts
//     - user_nkey  ???
//     - aud   - audience with Account key?
//     - iss   - issuer with ?server? key
//     - sub   - public user nkey for any response that the server accepts
//     - server_id.host,id,name of the server
//     server_id.id is the server signing key??
//     - client_tls.verified_chains with client cert in pem format
//
// The response:
//   - signed AuthorizationResponse with User JWT or error message
//   - signed with the private key of the callout issuer (account signing key)
//   - subject: public user nkey from the request (user_nkey from the req)
//   - aud: public key of issuing server ? is this the callout issuer???
func (svr *AuthnNatsServer) HandleCallOut(msg *nats.Msg) {
	slog.Info("received authcallout")

	reqClaims, err := jwt.DecodeAuthorizationRequestClaims(string(msg.Data))
	if err != nil {
		return
	}

	slog.Info("received authcallout", slog.String("userID", reqClaims.ConnectOptions.Name))
	userNKeyPub := reqClaims.UserNkey
	server := reqClaims.Server
	client := reqClaims.ClientInformation
	connectOpts := reqClaims.ConnectOptions
	tlsInfo := reqClaims.TLS

	_ = client
	_ = tlsInfo
	userID := connectOpts.Name
	// todo: verify request
	// todo, lookup user name
	//newToken, err := svr.service.CreateUserToken(userID, "todo", userNKeyPub, 123)

	uc := jwt.NewUserClaims(userID)
	// does the signing key from seed differ from the signing key themselves? how?
	//signingAccountKeyKP := svr.signingKey
	signingSeed, _ := svr.signingKey.Seed()
	signingAccountKeyKP, _ := nkeys.FromSeed(signingSeed)
	accountKeyPub, _ := svr.signingKey.PublicKey()
	uc.IssuerAccount = accountKeyPub // optional?
	// only in operator mode: set audience to the global account name
	//uc.Audience = accountName // $G for global
	uc.Name = "todo" // todo: lookup name
	//uc.UserPermissionLimits = *limits
	vr := jwt.CreateValidationResults()
	uc.Validate(vr)
	if len(vr.Errors()) != 0 {
		err = fmt.Errorf("validation error: %w", vr.Errors()[0])
	}
	newToken, err := uc.Encode(signingAccountKeyKP)

	// create and send the response
	respClaims := jwt.NewAuthorizationResponseClaims(userNKeyPub)
	respClaims.Audience = server.ID // public key of issuing server
	//respClaims.Error = "some error occurred"
	respClaims.Jwt = newToken

	respClaims.IssuedAt = time.Now().Unix()
	respClaims.Expires = time.Now().Add(time.Duration(999) * time.Second).Unix()
	// signingKey must be the issuer keys
	response, err := respClaims.Encode(svr.signingKey)
	err = msg.Respond([]byte(response))
	_ = err
}
func (natsrv *AuthnNatsServer) handleClientActions(action *hub.ActionMessage) error {
	slog.Info("handleClientActions", slog.String("actionID", action.ActionID))
	switch action.ActionID {
	case authn.NewTokenAction:
		req := &authn.NewTokenReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		// extra check, the sender's clientID must match the requested token client
		if action.ClientID != req.ClientID {
			err = fmt.Errorf("Client '%s' cannot request token for user '%s'", action.ClientID, req.ClientID)
			return err
		}
		newToken, err := natsrv.service.NewToken(
			action.ClientID, req.Password, req.PubKey)
		if err == nil {
			resp := authn.NewTokenResp{JwtToken: newToken}
			reply, _ := ser.Marshal(resp)
			action.SendReply(reply)
		}
		return err
	case authn.GetProfileAction:
		// use the current client
		profile, err := natsrv.service.GetClientProfile(action.ClientID)
		if err == nil {
			resp := authn.GetProfileResp{Profile: profile}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case authn.RefreshAction:
		req := &authn.RefreshReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		newToken, err := natsrv.service.Refresh(action.ClientID, req.OldToken)
		if err == nil {
			resp := authn.RefreshResp{JwtToken: newToken}
			reply, _ := ser.Marshal(resp)
			action.SendReply(reply)
		}
		return err
	case authn.UpdateNameAction:
		req := &authn.UpdateNameReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = natsrv.service.UpdateName(req.ClientID, req.NewName)
		if err == nil {
			action.SendAck()
		}
		return err
	case authn.UpdatePasswordAction:
		req := &authn.UpdatePasswordReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = natsrv.service.ResetPassword(req.ClientID, req.NewPassword)
		if err == nil {
			action.SendAck()
		}
		return err
	default:
		return nil
	}
}

func (natsrv *AuthnNatsServer) handleManageActions(action *hub.ActionMessage) error {
	slog.Info("handleManageActions",
		slog.String("actionID", action.ActionID),
		"my addr", natsrv)

	// TODO: doublecheck the caller is an admin or service
	switch action.ActionID {
	case authn.AddUserAction:
		req := authn.AddUserReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = natsrv.service.AddUser(req.UserID, req.Name, req.Password)
		if err == nil {
			action.SendAck()
		}
		return err
	case authn.GetClientProfileAction:
		req := authn.GetClientProfileReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		profile, err := natsrv.service.GetClientProfile(req.ClientID)
		if err == nil {
			resp := authn.GetProfileResp{Profile: profile}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case authn.ListClientsAction:
		clientList, err := natsrv.service.ListClients()
		if err == nil {
			resp := authn.ListClientsResp{Profiles: clientList}
			reply, _ := ser.Marshal(resp)
			action.SendReply(reply)
		}
		return err
	case authn.RemoveClientAction:
		req := &authn.RemoveClientReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = natsrv.service.RemoveClient(req.ClientID)
		if err == nil {
			action.SendAck()
		}
		return err
	default:
		//err := errors.New("invalid action '" + action.ActionID + "'")
		return nil
	}
}

// Start subscribes to the actions for management and client capabilities
func (natsrv *AuthnNatsServer) Start() {
	_ = natsrv.hc.SubActions(authn.ManageAuthnCapability, natsrv.handleManageActions)
	_ = natsrv.hc.SubActions(authn.ClientAuthnCapability, natsrv.handleClientActions)
	//_ = natsrv.hc.Subscribe(server.AuthCalloutSubject, )
}

// Stop removes subscriptions
func (natsrv *AuthnNatsServer) Stop() {
	//natsrv.hc.UnSubscribeAll()
}

// NewAuthnNatsServer create a nats binding for the authn service
//
//	svc is the authn service to bind to.
//	hc is the hub client, connected using the service credentials
func NewAuthnNatsServer(signingKey nkeys.KeyPair, svc *AuthnService, hc hub.IHubClient) *AuthnNatsServer {
	an := &AuthnNatsServer{
		service:    svc,
		hc:         hc,
		signingKey: signingKey,
	}
	return an
}
