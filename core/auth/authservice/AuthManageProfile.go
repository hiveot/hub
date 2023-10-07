package authservice

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"log/slog"
)

// AuthManageProfile is the capability for clients to view and update their own profile.
// This implements the IManageProfile interface.
//
// This implements the IAuthManageProfile interface.
type AuthManageProfile struct {
	// Client record persistence
	store auth.IAuthnStore
	// hub client for subscribing to requests
	hc hubclient.IHubClient
	// action subscription
	actionSub hubclient.ISubscription

	// message server for updating authn
	msgServer msgserver.IMsgServer
	// CA certificate for validating cert
	caCert *x509.Certificate
}

// GetProfile returns a client's profile
func (svc *AuthManageProfile) GetProfile(clientID string) (profile auth.ClientProfile, err error) {
	clientProfile, err := svc.store.GetProfile(clientID)
	return clientProfile, err
}

// HandleRequest unmarshal and invoke requests published by hub clients
func (svc *AuthManageProfile) HandleRequest(action *hubclient.RequestMessage) error {
	if action.ClientID == "" {
		return fmt.Errorf("missing clientID in action request. deviceID='%s', thingID='%s'",
			action.DeviceID, action.ThingID)
	}
	slog.Info("handleClientActions", slog.String("actionID", action.ActionID))
	switch action.ActionID {
	case auth.GetProfileReq:
		// use the current client
		profile, err := svc.GetProfile(action.ClientID)
		if err == nil {
			resp := auth.GetProfileResp{Profile: profile}
			reply, _ := ser.Marshal(&resp)
			err = action.SendReply(reply, nil)
		}
		return err
	case auth.NewTokenReq:
		req := &auth.NewTokenArgs{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		newToken, err := svc.NewToken(action.ClientID, req.Password)
		if err == nil {
			resp := auth.NewTokenResp{Token: newToken}
			reply, _ := ser.Marshal(resp)
			err = action.SendReply(reply, nil)
		}
		return err
	case auth.RefreshTokenReq:
		newToken, err := svc.Refresh(action.ClientID)
		if err == nil {
			resp := auth.RefreshResp{NewToken: newToken}
			reply, _ := ser.Marshal(resp)
			err = action.SendReply(reply, nil)
		}
		return err
	case auth.UpdateNameReq:
		req := &auth.UpdateNameArgs{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = svc.UpdateName(action.ClientID, req.NewName)
		if err == nil {
			err = action.SendAck()
		}
		return err
	case auth.UpdatePasswordReq:
		req := &auth.UpdatePasswordArgs{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = svc.UpdatePassword(action.ClientID, req.NewPassword)
		if err == nil {
			err = action.SendAck()
		}
		return err
	case auth.UpdatePubKeyReq:
		req := &auth.UpdatePubKeyArgs{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = svc.UpdatePubKey(action.ClientID, req.NewPubKey)
		if err == nil {
			err = action.SendAck()
		}
		return err
	default:
		return fmt.Errorf("Unknown user action '%s' for client '%s'", action.ActionID, action.ClientID)
	}
}

// NewToken validates a password and issues an authn token. A public key must be on file.
func (svc *AuthManageProfile) NewToken(clientID string, password string) (newToken string, err error) {
	clientProfile, err := svc.store.VerifyPassword(clientID, password)
	if err != nil {
		return "", err
	}
	authInfo := msgserver.ClientAuthInfo{
		ClientID:   clientProfile.ClientID,
		ClientType: clientProfile.ClientType,
		PubKey:     clientProfile.PubKey,
		Role:       clientProfile.Role,
	}
	newToken, err = svc.msgServer.CreateToken(authInfo)
	return newToken, err
}

// notification handler invoked when clients have been updated
// this invokes a reload of server authn
func (svc *AuthManageProfile) onChange() {
	// wait with applying credential changes to allow a response to be send
	go svc.msgServer.ApplyAuth(svc.store.GetAuthClientList())
}

// Refresh issues a new token for the authenticated user.
// This returns a refreshed token that can be used to connect to the messaging server
// the old token must be a valid jwt token belonging to the clientID
func (svc *AuthManageProfile) Refresh(clientID string) (newToken string, err error) {
	// verify the token
	clientProfile, err := svc.store.GetProfile(clientID)
	if err != nil {
		return "", err
	}
	//err = svc.msgServer.ValidateToken(
	//	clientID, clientProfile.PubKey, oldToken, "", "")
	//if err != nil {
	//	return "", fmt.Errorf("error validating oldToken of client %s: %w", clientID, err)
	//}
	authInfo := msgserver.ClientAuthInfo{
		ClientID:   clientProfile.ClientID,
		ClientType: clientProfile.ClientType,
		PubKey:     clientProfile.PubKey,
		Role:       clientProfile.Role,
	}
	newToken, err = svc.msgServer.CreateToken(authInfo)
	return newToken, err
}

// Start subscribes to the actions for management and client capabilities
// Register the binding subscription using the given connection
func (svc *AuthManageProfile) Start() (err error) {
	if svc.hc != nil {
		svc.actionSub, _ = svc.hc.SubServiceRPC(
			auth.AuthProfileCapability, svc.HandleRequest)
	}
	return err
}

// Stop removes subscriptions
func (svc *AuthManageProfile) Stop() {
	if svc.actionSub != nil {
		svc.actionSub.Unsubscribe()
		svc.actionSub = nil
	}
}

// UpdateName
func (svc *AuthManageProfile) UpdateName(clientID string, displayName string) (err error) {
	clientProfile, err := svc.store.GetProfile(clientID)
	clientProfile.DisplayName = displayName
	err = svc.store.Update(clientID, clientProfile)
	// this doesn't affect authentication
	return err
}

func (svc *AuthManageProfile) UpdatePassword(clientID string, newPassword string) (err error) {
	slog.Info("UpdatePassword", "clientID", clientID)
	_, err = svc.GetProfile(clientID)
	if err != nil {
		return err
	}
	err = svc.store.SetPassword(clientID, newPassword)
	if err != nil {
		return err
	}
	svc.onChange()
	return err
}

func (svc *AuthManageProfile) UpdatePubKey(clientID string, newPubKey string) (err error) {
	slog.Info("UpdatePubKey", "clientID", clientID)
	clientProfile, err := svc.store.GetProfile(clientID)
	if err != nil {
		return err
	}
	clientProfile.PubKey = newPubKey
	err = svc.store.Update(clientID, clientProfile)
	if err != nil {
		return err
	}
	// run in the background so a response can be sent
	go svc.onChange()
	return err
}

// NewAuthManageProfile returns a user profile management capability.
//
//	store holds the authentication client records
//	caCert is an optional CA used to verify certificates. Use nil to not authn using client certs
func NewAuthManageProfile(
	store auth.IAuthnStore,
	caCert *x509.Certificate,
	hc hubclient.IHubClient,
	msgServer msgserver.IMsgServer,
) *AuthManageProfile {

	svc := &AuthManageProfile{
		store:     store,
		hc:        hc,
		msgServer: msgServer,
		caCert:    caCert,
	}
	return svc
}
