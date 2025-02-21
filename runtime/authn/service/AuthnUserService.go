package service

import (
	"github.com/hiveot/hub/messaging"
	authn "github.com/hiveot/hub/runtime/authn/api"
	"github.com/hiveot/hub/runtime/authn/authnstore"
	"github.com/hiveot/hub/runtime/authn/config"
	"log/slog"
)

// AuthnUserService handles authentication and authorization of regular users
// such as agents, services and end-users.
type AuthnUserService struct {
	authnStore authnstore.IAuthnStore

	cfg *config.AuthnConfig

	// the authenticator for jwt tokens
	sessionAuth messaging.IAuthenticator
}

// GetProfile returns a client's profile
func (svc *AuthnUserService) GetProfile(
	senderID string) (resp authn.ClientProfile, err error) {

	prof, err := svc.authnStore.GetProfile(senderID)
	return prof, err
}

// Login with password and return a new session token
func (svc *AuthnUserService) Login(_ string, args authn.UserLoginArgs) (token string, err error) {

	token, err = svc.sessionAuth.Login(args.ClientID, args.Password)
	return token, err
}

// Logout and remove the client session
func (svc *AuthnUserService) Logout(senderID string) error {
	svc.sessionAuth.Logout(senderID)
	return nil
}

// RefreshToken requests a new token based on the old token
func (svc *AuthnUserService) RefreshToken(
	senderID string, args string) (newToken string, err error) {

	newToken, err = svc.sessionAuth.RefreshToken(senderID, args)
	return newToken, err
}

// Start the user facing authentication service.
func (svc *AuthnUserService) Start() error {
	slog.Info("starting AuthnService")
	return nil
}

// Stop the service, unsubscribe and disconnect from the server
func (svc *AuthnUserService) Stop() {
	slog.Info("Stopping AuthnUserService")
}

func (svc *AuthnUserService) UpdateName(senderID string, newName string) error {

	slog.Info("UpdateName", "clientID", senderID, "newName", newName)
	prof, err := svc.authnStore.GetProfile(senderID)
	if err == nil {
		prof.DisplayName = newName
		err = svc.authnStore.UpdateProfile(senderID, prof)
	}
	if err != nil {
		slog.Error("Failed changing name",
			"clientID", senderID, "err", err.Error())
	}
	return err
}
func (svc *AuthnUserService) UpdatePassword(senderID string, password string) error {

	slog.Info("SetClientPassword", "senderID", senderID)
	err := svc.authnStore.SetPassword(senderID, password)
	if err != nil {
		slog.Error("Failed changing password",
			"senderID", senderID, "err", err.Error())
	}
	return err
}
func (svc *AuthnUserService) UpdatePubKey(senderID string, pubKeyPEM string) error {

	slog.Info("UpdatePubKey", "clientID", senderID)
	prof, err := svc.authnStore.GetProfile(senderID)
	if err == nil {
		prof.PubKey = pubKeyPEM
		err = svc.authnStore.UpdateProfile(senderID, prof)
	}
	if err != nil {
		slog.Error("Failed updating public key",
			"clientID", senderID, "err", err.Error())
	}
	return err
}

// ValidateToken verifies that the given token is valid
//func (svc *AuthnUserService) ValidateToken(senderID string, token string) (
//	resp authn.UserValidateTokenResp, err error) {
//
//	clientID, sid, err := svc.sessionAuth.ValidateToken(token)
//	if err == nil && clientID != senderID {
//		err = fmt.Errorf("ClientID doesn't match senderID")
//	}
//	resp.ClientID = clientID
//	resp.SessionID = sid
//	resp.Error = err.Error()
//	return resp, nil
//}

// NewAuthnUserService creates an end-user authentication service instance for
// logging in and managing a user's own profile. This service is accessible by any
// client.
// This service works in conjunction with the authentication store and the
// session authenticator. The latter is created by the admin service or can be instantiated
// using a signing key.
//
// This service does not have start/stop functions. The authnStore and authenticator
// must be operational before using this service.
//
//	authnStore is the client and credentials store. Must be opened before starting this service.
//	cm is the connection manager used to close all connections on logout
//	sessionAuth is the authenticator returned by the admin service.
func NewAuthnUserService(
	cfg *config.AuthnConfig,
	authnStore authnstore.IAuthnStore,
	authenticator messaging.IAuthenticator) *AuthnUserService {

	authnSvc := &AuthnUserService{
		cfg:         cfg,
		authnStore:  authnStore,
		sessionAuth: authenticator,
	}
	return authnSvc
}
